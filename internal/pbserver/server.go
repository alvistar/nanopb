package pbserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/alvistar/nanopb/internal/nwsclient"
	"github.com/alvistar/nanopb/internal/usclient"
	pb "github.com/alvistar/nanopb/nanoproto"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"runtime/debug"
)

type TransformF = func(interface{}) interface{}

type TransformOpt map[string]TransformF

var logger *log.Entry

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
	errInvalidToken    = status.Errorf(codes.Unauthenticated, "invalid token")
)

func str(s string) TransformF {
	return func(v interface{}) interface{} { return s }
}

func boolToStr() TransformF {
	return func(v interface{}) interface{} {
		if b, ok := v.(bool); ok {
			if b {
				return "true"
			} else {
				return "false"
			}
		} else {
			return "false"
		}
	}
}

func getAction(message proto.Message, action string, options TransformOpt) (string, error) {
	m := jsonpb.Marshaler{
		OrigName: true,
	}
	orig, _ := m.MarshalToString(message)

	jsonParsed, _ := gabs.ParseJSON([]byte (orig))
	_, _ = jsonParsed.Set(action, "action")

	for k, v := range options {
		_, _ = jsonParsed.Set(v(jsonParsed.Path(k).Data()), k)
	}

	return jsonParsed.String(), nil
}

type Server struct {
	USConfig      *usclient.ConfNode
	usClient      usclient.IUSClient
	wsClient      nwsclient.WSClient
	PubKey        []byte
	LocalAccounts bool
}

func (server *Server) Init(l *log.Logger) {
	server.usClient = &usclient.USClient{}
	server.usClient.Init(server.USConfig, l)
	//server.loadPubKey("key.pem")
	server.wsClient = nwsclient.WSClient{LocalAccounts: server.LocalAccounts}
	server.wsClient.Init(l)

	if l == nil {
		l = log.New()
	}

	logger = l.WithFields(log.Fields{"component": "npb_server"})
}

func (server *Server) loadPubKey(filename string) {
	keyData, e := ioutil.ReadFile(filename)
	if e != nil {
		panic(e.Error())
	}

	server.PubKey = keyData
}

func (server *Server) handler(request string, reply proto.Message) ( error) {
	logger.Debug("IPC -< ", request)

	jreply, err := server.usClient.Get([]byte(request))

	if err != nil {
		logger.Error("error from nano ipc: %s", err)
		return err
	}

	if err := jsonpb.UnmarshalString(string(jreply), reply); err != nil {
		// Try getting json error

		if jsonParsed, err := gabs.ParseJSON(jreply); err == nil {
			apiErr, ok := jsonParsed.Path("error").Data().(string)
			if ok {
				return errors.New(apiErr)
			}
		}

		logger.Error("error unmarshalling json: ", err)
		logger.Error(string(jreply))
		debug.PrintStack()
		return err
	}

	return nil
}

func (server *Server) unsubscribe(channel *chan pb.SubscriptionEntry) {
	logger.Debug("unsubscribing channel")
	server.wsClient.Unsubscribe(channel)
}

func (server *Server) Subscribe(request *pb.SubscribeRequest, stream pb.Nano_SubscribeServer) error {
	ch := make(chan pb.SubscriptionEntry)
	server.wsClient.Subscribe(&ch, request.Accounts)
	for entry := range ch {
		if err := stream.Send(&entry); err != nil {
			server.unsubscribe(&ch)
			return err
		}
	}

	server.unsubscribe(&ch)

	return nil
}

// valid validates the authorization.
func valid(authorization []string, key []byte) bool {
	if len(authorization) < 1 {
		return false
	}

	jkey, _ := jwt.ParseRSAPublicKeyFromPEM(key)

	token, err := jwt.Parse(authorization[0], func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return jkey, nil
	})

	if err != nil {
		log.Printf("error validating token:%s", err)
		return false
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		fmt.Println(claims["some"], claims["nbf"])
	} else {
		log.Printf("error validating token:%s", err)
	}

	return true
}

// EnsureValidToken ensures a valid token exists within a request's metadata. If
// the token is missing or invalid, the interceptor blocks execution of the
// handler and returns an error. Otherwise, the interceptor invokes the unary
// handler.
func EnsureValidToken(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMissingMetadata
	}

	// The keys within metadata.MD are normalized to lowercase.
	// See: https://godoc.org/google.golang.org/grpc/metadata#New
	if !valid(md["auth-token-bin"], info.Server.(*Server).PubKey) {
		return nil, errInvalidToken
	}
	// Continue execution of handler after ensuring a valid token.
	return handler(ctx, req)
}

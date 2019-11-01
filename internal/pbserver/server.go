package pbserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/alvistar/gonano/internal/nanoclient"
	"github.com/alvistar/gonano/internal/nwsclient"
	pb "github.com/alvistar/gonano/nanoproto"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/zput/zxcTool/ztLog/zt_formatter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"path"
	"runtime"
	"runtime/debug"
)

type TransformF = func (interface{}) interface{}

type TransformOpt map[string] TransformF

var logger *log.Entry

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
	errInvalidToken    = status.Errorf(codes.Unauthenticated, "invalid token")
)

func str(s string) TransformF {
	return func (v interface{})interface{} { return s}
}

func boolToStr() TransformF {
	return func (v interface{})interface{} {
		if b, ok :=v.(bool); ok {
			if b {return "true"} else {return "false"}
		} else {
			return "false"
		}
	}
}

func getAction(message proto.Message, action string, options TransformOpt) (string , error) {
	m := jsonpb.Marshaler{
		OrigName: true,
	}
	orig, _:= m.MarshalToString(message)

	jsonParsed, _ := gabs.ParseJSON([]byte (orig))
	_, _ = jsonParsed.Set(action, "action")

	for k,v := range options {
		_, _ = jsonParsed.Set(v(jsonParsed.Path(k).Data()), k)
	}

	return jsonParsed.String(), nil
}

type Server struct {
	client         nanoclient.INanoClient
	wsClient       nwsclient.WSClient
	Pubkey         []byte
	Authentication bool
}

func (server *Server) Init() {
	server.Authentication = false
	server.client = & nanoclient.NanoClient{}
	server.client.Init()
	server.loadPubKey("key.pem")
	server.wsClient = nwsclient.WSClient{}
	server.wsClient.Init()

	l := log.New()

	l.SetFormatter(&zt_formatter.ZtFormatter{
		Formatter:        nested.Formatter{
			HideKeys: true,
			FieldsOrder: []string{"component"},
		},
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
		},
	})

	l.SetReportCaller(true)
	l.SetLevel(log.DebugLevel)

	logger = l.WithFields(log.Fields{"component": "npb_server"})
}

func (server *Server) loadPubKey(filename string) {
	keyData, e := ioutil.ReadFile(filename)
	if e != nil {
		panic(e.Error())
	}

	server.Pubkey = keyData
}

func (server *Server) handler(request string, reply proto.Message) ( error) {
	logger.Debug("IPC -< ", request)

	jreply, err := server.client.Get([]byte(request))

	if err != nil {
		log.Printf("error from nano ipc: %s", err)
		return  err}

	if err := jsonpb.UnmarshalString(string(jreply), reply); err != nil {
		// Try getting json error

		if jsonParsed, err:= gabs.ParseJSON(jreply); err == nil {
			apiErr, ok :=jsonParsed.Path("error").Data().(string)
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



func (server *Server) Subscribe(request *pb.SubscribeRequest, stream pb.Nano_SubscribeServer) error {
	ch := make(chan pb.SubscriptionEntry)
	server.wsClient.Subscribe(&ch, request.Accounts)
	for entry := range ch {
		if err := stream.Send(&entry); err != nil {
			return err
		}
	}
	return nil
}

// valid validates the authorization.
func valid(authorization []string, key []byte) bool {
	if len(authorization) < 1 {
		return false
	}

	jkey, _ := jwt.ParseRSAPublicKeyFromPEM(key)

	token, err:= jwt.Parse(authorization[0], func(token *jwt.Token) (interface{}, error) {
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
	if info.Server.(*Server).Authentication == false {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMissingMetadata
	}

	// The keys within metadata.MD are normalized to lowercase.
	// See: https://godoc.org/google.golang.org/grpc/metadata#New
	if !valid(md["auth-token-bin"], info.Server.(*Server).Pubkey) {
		return nil, errInvalidToken
	}
	// Continue execution of handler after ensuring a valid token.
	return handler(ctx, req)
}
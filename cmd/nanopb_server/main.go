/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

//go:generate protoc -I ../../nanoproto --go_out=plugins=grpc:../../nanoproto ../../nanoproto/nano.proto

// Package main implements a Server for Greeter service.
package main

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
	"net"
	"path"
	"runtime"
	"runtime/debug"
)

const (
	port = ":50051"
)

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
	errInvalidToken    = status.Errorf(codes.Unauthenticated, "invalid token")
)

type Server struct {
	client         nanoclient.INanoClient
	wsClient       nwsclient.WSClient
	pubkey         []byte
	authentication bool
}

var logger *log.Entry

func getAction(message proto.Message, action string, options map[string]string) (string , error) {
	m := jsonpb.Marshaler{}
	orig, _:= m.MarshalToString(message)

	jsonParsed, _ := gabs.ParseJSON([]byte (orig))
	_, _ = jsonParsed.Set(action, "action")

	for k,v := range options {
		_, _ = jsonParsed.Set(v, k)
	}

	return jsonParsed.String(), nil
}

func (server *Server) init () {
	server.authentication = false
	server.client = & nanoclient.NanoClient{}
	server.client.Init()
	server.loadPubKey("key.pem")
	server.wsClient = nwsclient.WSClient{}
	server.wsClient.Init()
}

func (server *Server) loadPubKey(filename string) {
	keyData, e := ioutil.ReadFile(filename)
	if e != nil {
		panic(e.Error())
	}

	server.pubkey = keyData
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

func (server *Server) AccountsBalances(ctx context.Context, pbRequest *pb.AccountsBalancesRequest) (*pb.AccountsBalancesReply, error) {

	request, _ := getAction(pbRequest, "accounts_balances", nil)

	reply := pb.AccountsBalancesReply {}

	if err:=server.handler(request, &reply); err == nil {
		return &reply, nil
	} else {
		return nil, err
	}

}

func (server *Server) BlocksInfo(ctx context.Context, pbRequest *pb.BlocksInfoRequest) (*pb.BlocksInfoReply, error) {
	request, _ := getAction(pbRequest, "blocks_info",
		map[string]string{"json_block":"true"} )

	reply := pb.BlocksInfoReply {}

	if err:=server.handler(request, &reply); err == nil {
		return &reply, nil
	} else {
		return nil, err
	}
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

// ensureValidToken ensures a valid token exists within a request's metadata. If
// the token is missing or invalid, the interceptor blocks execution of the
// handler and returns an error. Otherwise, the interceptor invokes the unary
// handler.
func ensureValidToken(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if info.Server.(*Server).authentication == false {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMissingMetadata
	}

	// The keys within metadata.MD are normalized to lowercase.
	// See: https://godoc.org/google.golang.org/grpc/metadata#New
	if !valid(md["auth-token-bin"], info.Server.(*Server).pubkey) {
		return nil, errInvalidToken
	}
	// Continue execution of handler after ensuring a valid token.
	return handler(ctx, req)
}



func main() {
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

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		// The following grpc.ServerOption adds an interceptor for all unary
		// RPCs. To configure an interceptor for streaming RPCs, see:
		// https://godoc.org/google.golang.org/grpc#StreamInterceptor
		grpc.UnaryInterceptor(ensureValidToken),
		// Enable TLS for all incoming connections.
		// grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
	}


	s := grpc.NewServer(opts...)
	server := &Server{authentication:false}
	server.pubkey = nil
	server.init()
	pb.RegisterNanoServer(s, server)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

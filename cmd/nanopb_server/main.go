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

//go:generate protoc -I ../nanoproto --go_out=plugins=grpc:../nanoproto ../nanoproto/nano.proto

// Package main implements a Server for Greeter service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alvistar/gonano/nanoclient"
	pb "github.com/alvistar/gonano/nanoproto"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"log"
	"net"
)

const (
	port = ":50051"
)

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
	errInvalidToken    = status.Errorf(codes.Unauthenticated, "invalid token")
)

type Server struct {
	client nanoclient.INanoClient
	WSClient wsclient;
	pubkey []byte
}
func (server *Server) init () {
	server.client = & nanoclient.NanoClient{}
	server.client.Init()
	server.loadPubKey("key.pem")
	server.WSClient = WSClient{}

}

func (server *Server) loadPubKey(filename string) {
	keyData, e := ioutil.ReadFile(filename)
	if e != nil {
		panic(e.Error())
	}

	server.pubkey = keyData
}

func (server *Server) handler(request interface{}, reply proto.Message) error {
	data, err := json.Marshal(request)

	if err != nil {
		log.Printf("error marshaling json: %s", err)
		return  err}

	jreply, err := server.client.Get(data)

	if err != nil {
		log.Printf("error from nano ipc: %s", err)
		return  err}

	if err := jsonpb.UnmarshalString(string(jreply), reply); err != nil {
		log.Printf("error unmarshalling json: %s", err)
		return err
	}

	return nil
}

func (server *Server) BlocksInfo(ctx context.Context, pbRequest *pb.BlocksInfoRequest) (*pb.BlocksInfoReply, error) {
	request := map[string]interface{} {
		"action": "blocks_info",
		"json_block": "true",
		"hashes": pbRequest.Hashes,
	}

	reply := pb.BlocksInfoReply {}

	if err:=server.handler(request, &reply); err != nil {
		return nil, err
	}

	return &reply, nil
}

// valid validates the authorization.
func valid(authorization []string, key []byte) bool {
	if len(authorization) < 1 {
		return false
	}

	jkey, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(key))

	token, err:= jwt.Parse(authorization[0], func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
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
	server := &Server{}
	server.pubkey = nil
	server.init()
	pb.RegisterNanoServer(s, server)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

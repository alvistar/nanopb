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

//go:generate protoc -I ../nanoproto --go_out=plugins=grpc:../nanoproto ../nanoproto/nanoproto.proto

// Package main implements a Server for Greeter service.
package main

import (
	"context"
	"encoding/json"
	"github.com/alvistar/gonano/nanoclient"
	pb "github.com/alvistar/gonano/nanoproto"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"
	"log"
	"net"
)

const (
	port = ":50051"
)

type Server struct {
	client nanoclient.INanoClient
}
func (server *Server) init () {
	server.client = & nanoclient.NanoClient{}
	server.client.Init()
}

func (server *Server) BlocksInfo(ctx context.Context, pbRequest *pb.BlocksInfoRequest) (*pb.BlocksInfoReply, error) {
	type BlockInfoRequest struct {
		Action string `json:"action"`
		JsonBlock string `json:"json_block"`
		Hashes []string `json:"hashes"`
	}
	
	request := BlockInfoRequest{
		Action:    "blocks_info",
		JsonBlock: "true",
		Hashes:    pbRequest.Hashes,
	}

	data,_ := json.Marshal(request)

	jreply,_ := server.client.Get(data)

	//jsonparser.ObjectEach(jreply,
	//			func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
	//				fmt.Printf("Key: '%s'\n Value: '%s'\n Type: %s\n", string(key), string(value), dataType)
	//				return nil
	//			}, "blocks")

	println(string(jreply))
	reply := pb.BlocksInfoReply {}

	if err := jsonpb.UnmarshalString(string(jreply), &reply); err != nil {
		log.Printf("error unmarshalling json: %s", err)
		return nil, err
	}

	return &reply, nil
}

// SayHello implements helloworld.GreeterServer
//func (s *Server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
//	log.Printf("Received: %v", in.GetName())
//	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
//}


func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	server := &Server{}
	server.init()
	pb.RegisterNanoServer(s, server)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

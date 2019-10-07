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
	"github.com/alvistar/gonano/nanoclient"
	pb "github.com/alvistar/gonano/nanoproto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
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

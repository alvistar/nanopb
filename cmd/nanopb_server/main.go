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
	"github.com/alvistar/gonano/internal/pbserver"
	pb "github.com/alvistar/gonano/nanoproto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

const (
	port = ":50051"
)

func main() {


	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		// The following grpc.ServerOption adds an interceptor for all unary
		// RPCs. To configure an interceptor for streaming RPCs, see:
		// https://godoc.org/google.golang.org/grpc#StreamInterceptor
		grpc.UnaryInterceptor(pbserver.EnsureValidToken),
		// Enable TLS for all incoming connections.
		// grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
	}


	s := grpc.NewServer(opts...)
	server := &pbserver.Server{Authentication:false}
	server.Pubkey = nil
	server.Init()
	pb.RegisterNanoServer(s, server)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

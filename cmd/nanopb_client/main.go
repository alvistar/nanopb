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

// Package main implements a client for Greeter service.
package main

import (
	"context"
	pb "github.com/alvistar/gonano/nanoproto"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"io"
	"log"
	"time"
)

const (
	address     = "localhost:50051"
	defaultName = "world"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewNanoClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	request := pb.BlocksInfoRequest{
		Hashes:[]string{"87434F8041869A01C8F6F263B87972D7BA443A72E0A97D7A3FD0CCC2358FD6F9"},
	}
	r, err := c.BlocksInfo(ctx, &request)
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf(proto.MarshalTextString(r))

	sub := &pb.SubscribeRequest{
		Accounts: []string {"nano_1jrd1ri7dfo1gyh9iqqmtfk1aq64oi9c57xixtjdosfjwmxpkebpuruuen34"},
	}

	stream, err := c.Subscribe(context.Background(), sub)

	if err != nil {
		log.Fatal("Error creating stream")
	}

	for {
		entry, err := stream.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal("Fatal error ", err)
		}

		log.Println(entry)
	}
}

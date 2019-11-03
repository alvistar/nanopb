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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/alvistar/nanopb/internal/pbserver"
	"github.com/alvistar/nanopb/internal/usclient"
	pb "github.com/alvistar/nanopb/nanoproto"
	nested "github.com/antonfisher/nested-logrus-formatter"
	log "github.com/sirupsen/logrus"
	"github.com/zput/zxcTool/ztLog/zt_formatter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"net"
	"os"
	"path"
	"runtime"
)

func AppendCertsFromFile(pool *x509.CertPool, fileName string) error {
	cert, err := ioutil.ReadFile(fileName)


	if err != nil {
		return fmt.Errorf("could not read certificate: %s", err)
	}

	// Append the client certificates from the CA
	if ok := pool.AppendCertsFromPEM(cert); !ok {
		return errors.New("failed to append client certs")
	}

	return nil
}

func setupLog(debug bool) *log.Logger{
	l := log.New()

	if debug {

		l.SetFormatter(&zt_formatter.ZtFormatter{
			Formatter: nested.Formatter{
				HideKeys:    true,
				FieldsOrder: []string{"component"},
			},
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := path.Base(f.File)
				return fmt.Sprintf("%s()", f.Func.Name()), fmt.Sprintf("%s:%d", filename, f.Line)
			},
		})

		l.SetReportCaller(true)
		l.SetLevel(log.DebugLevel)

	} else {

		l.SetFormatter(&nested.Formatter{
				HideKeys:    true,
				FieldsOrder: []string{"component"},
			})

		l.SetLevel(log.InfoLevel)
	}

	return l
}

func main() {

	parser := argparse.NewParser("nanopb", "Nano Protobuf Gateway")

	address := parser.String("a", "address",
		&argparse.Options{Help: "Address to bind", Default:":50051"})

	poolSize := parser.Int("", "poolSize",
		&argparse.Options{Help: "Unix Socket Pool Size", Default:3})

	socket := parser.String("", "socket",
		&argparse.Options{Help: "Unix socket path", Default:"local:///tmp/nano"})

	ssl := parser.Flag("s", "ssl",
		&argparse.Options{Help: "Enable ssl", Default: false})

	certFile := parser.String("", "certfile",
		&argparse.Options{Help: "Server certification file"})

	keyFile := parser.String("", "keyfile",
		&argparse.Options{Help: "Server key file"})

	caCert := parser.String("", "cacert",
		&argparse.Options{Help: "CA cert file"})

	clientAuth := parser.Flag("", "clientauth",
		&argparse.Options{Help: "Request client certificate"})

	debug := parser.Flag("D", "debug",
		&argparse.Options{Help: "Enable detail debug log"})

	localAccounts := parser.Flag("", "localaccounts",
		&argparse.Options{Help: "Subscribe to confirmation of local accounts only"})

	err := parser.Parse(os.Args)

	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	// Other validation

	if *ssl {
		if *certFile == "" || *keyFile == "" {
			fmt.Print(parser.Usage("Need to specify both certfile and keyfile when using ssl"))
			os.Exit(1)
		}
	}

	logger := setupLog(*debug)


	lis, err := net.Listen("tcp", *address)
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}

	confnode := usclient.ConfNode{
		Connection: *socket,
		PoolSize:   *poolSize,
	}

	opts := make([]grpc.ServerOption, 0)


	//opts := []grpc.ServerOption{
	//	// The following grpc.ServerOption adds an interceptor for all unary
	//	// RPCs. To configure an interceptor for streaming RPCs, see:
	//	// https://godoc.org/google.golang.org/grpc#StreamInterceptor
	//	// grpc.UnaryInterceptor(pbserver.EnsureValidToken),
	//	// Enable TLS for all incoming connections.
	//	// grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
	//}

	// SSL Configuration

	if *ssl {
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			logger.Fatalf("Error loading certificates: %s", err)
		}

		tlsConfig := tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		if caCert != nil {
			certpool := x509.NewCertPool()

			if err:=AppendCertsFromFile(certpool, *caCert); err!=nil {
				logger.Fatal(err)
			}
			tlsConfig.ClientCAs = certpool
		}

		if *clientAuth {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}

		creds := credentials.NewTLS(&tlsConfig)

		opts = append(opts, grpc.Creds(creds))

	}


	s := grpc.NewServer(opts...)
	server := &pbserver.Server{
		USConfig: &confnode,
		LocalAccounts: *localAccounts,
	}

	server.PubKey = nil
	server.Init(logger)
	pb.RegisterNanoServer(s, server)
	if err := s.Serve(lis); err != nil {
		logger.Fatalf("failed to serve: %v", err)
	}
}

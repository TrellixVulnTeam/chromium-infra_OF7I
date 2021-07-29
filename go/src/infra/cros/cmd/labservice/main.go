// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command labservice implements the Chrome OS Lab Service.
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"

	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"google.golang.org/grpc"
)

var (
	addr = flag.String("addr", "0.0.0.0:1485", "Address to listen to")
)

func main() {
	// Configure the default Go logger only for handling fatal
	// errors in main and any libraries that are using it.
	// Otherwise, labservice code should use the internal log package.
	log.SetPrefix("labservice: ")
	if err := innerMain(); err != nil {
		log.Fatalf("Fatal error: %s", err)
	}
}

func innerMain() error {
	flag.Parse()
	l, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	gs := newServer()
	c := make(chan os.Signal, 1)
	signal.Notify(c, handledSignals...)
	ctx := context.Background()
	// This goroutine exits when the program exits.
	go func() {
		for sig := range c {
			// Handle asynchronously so we can handle
			// cases like getting a SIGINT (graceful stop)
			// followed by a SIGTERM (immediate stop).
			go handleSignal(ctx, gs, sig)
		}
	}()
	return gs.Serve(l)
}

// newServer creates a new gRPC server for labservice.
func newServer() *grpc.Server {
	ic := interceptor{}
	gs := grpc.NewServer(ic.unaryOption())
	s := &server{}
	labapi.RegisterInventoryServiceServer(gs, s)
	return gs
}

// interceptor has gRPC interceptor methods.
// This is the only way to modify the context passed to method handlers.
type interceptor struct{}

func (interceptor) unary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (interface{}, error) {
	return h(ctx, req)
}

func (ic interceptor) unaryOption() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(ic.unary)
}

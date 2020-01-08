// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command common-tls implements the shared high level test lab services (TLS) API.
// This depends on a separate implementation of the low level TLS wiring API.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

var (
	port       = flag.Int("port", 0, "Port to listen to")
	wiringPort = flag.Int("wiring-port", 0, "Port for the TLS wiring service")
)

func main() {
	if err := innerMain(); err != nil {
		log.Fatalf("common-tls: %s", err)
	}
}

func innerMain() error {
	flag.Parse()
	c, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", *wiringPort), grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer c.Close()
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		return err
	}
	s := server{
		conn: c,
	}
	if err := s.Serve(l); err != nil {
		return err
	}
	return nil
}

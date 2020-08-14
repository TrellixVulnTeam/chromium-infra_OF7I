// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command dev-tlw implements the TLS wiring API for development convenience.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

var (
	port = flag.Int("port", 0, "Port to listen to")
)

func main() {
	flag.Parse()
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatalf("dev-tlw: %s", err)
	}
	s := server{}
	if err := s.Serve(l); err != nil {
		log.Fatalf("dev-tlw: %s", err)
	}
}

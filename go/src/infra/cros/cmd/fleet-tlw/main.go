// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command fleet-tlw implements the TLS wiring API for Chrome OS fleet labs.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

var (
	port = flag.Int("port", 0, "Port to listen to")
)

func main() {
	flag.Parse()
	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		log.Fatalf("fleet-tlw: %s", err)
	}
	sigChan := make(chan os.Signal, 1)
	s := newServer()
	go func() {
		SetUpSignalHandler(sigChan)
		sig := <-sigChan
		log.Printf("Captured %v, stopping fleet-tlw service and cleaning up...", sig)
		s.Close()
	}()
	if err := s.Serve(l); err != nil {
		log.Fatalf("fleet-tlw: %s", err)
	}
}

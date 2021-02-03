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
	"sync"

	"infra/cros/cmd/fleet-tlw/internal/cache"

	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 0, "Port to listen to")
)

func main() {
	if err := innerMain(); err != nil {
		log.Fatalf("fleet-tlw: %s", err)
	}
}

func innerMain() error {
	flag.Parse()
	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		log.Fatalf("fleet-tlw: %s", err)
	}
	s := grpc.NewServer()

	// TODO (guocb) Fetch caching backends data from UFS after migration to
	// caching cluster.
	ce, err := cache.NewDevserverEnv(cache.AutotestConfig)
	if err != nil {
		return err
	}

	tlw := newTLWServer(ce)
	tlw.registerWith(s)
	defer tlw.Close()

	ss := newSessionServer(ce)
	ss.registerWith(s)
	defer ss.Close()

	c := setupSignalHandler()
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		sig := <-c
		log.Printf("Captured %v, stopping fleet-tlw service and cleaning up...", sig)
		s.GracefulStop()
	}()
	return s.Serve(l)
}

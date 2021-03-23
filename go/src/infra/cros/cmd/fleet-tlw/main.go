// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command fleet-tlw implements the TLS wiring API for Chrome OS fleet labs.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"

	"infra/cros/cmd/fleet-tlw/internal/cache"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

var (
	port            = flag.Int("port", 0, "Port to listen to")
	ufsService      = flag.String("ufs-service", "ufs.api.cr.dev", "Host of the UFS service")
	svcAcctJSONPath = flag.String("service-account-json", "", "Path to JSON file with service account credentials to use")
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

	ce, err := cacheEnv()
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

func cacheEnv() (cache.Environment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	uc, err := ufsapi.NewClient(ctx, ufsapi.ServiceName(*ufsService), ufsapi.ServiceAccountJSONPath(*svcAcctJSONPath), ufsapi.UserAgent("fleet-tlw/3.0.0"))
	if err != nil {
		return nil, err
	}
	ce, err := cache.NewUFSEnv(uc)
	if err != nil {
		return nil, err
	}

	return ce, nil
}

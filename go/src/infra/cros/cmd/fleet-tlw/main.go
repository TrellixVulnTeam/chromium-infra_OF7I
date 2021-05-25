// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command fleet-tlw implements the TLS wiring API for Chrome OS fleet labs.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
)

var (
	port            = flag.Int("port", 0, "Port to listen to")
	ufsService      = flag.String("ufs-service", "ufs.api.cr.dev", "Host of the UFS service")
	svcAcctJSONPath = flag.String("service-account-json", "", "Path to JSON file with service account credentials to use")
	proxySSHKey     = flag.String("proxy-ssh-key", "", "Path to SSH key for SSH proxy servers (no auth for ExposePortToDut Proxy Mode if unset)")
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

	proxySSHSigner, err := authMethodFromKey(*proxySSHKey)
	if err != nil {
		return err
	}

	b := fleetTLWBuilder{ufsService: *ufsService, proxySSHSigner: proxySSHSigner, serviceAcctJSON: *svcAcctJSONPath}
	tlw, err := b.build()
	if err != nil {
		return err
	}

	tlw.registerWith(s)
	defer tlw.Close()

	ss := newSessionServer(b)
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

func authMethodFromKey(keyfile string) (ssh.Signer, error) {
	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, fmt.Errorf("auth ssh from key: %s", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("auth ssh from key: %s", err)
	}
	return signer, nil
}

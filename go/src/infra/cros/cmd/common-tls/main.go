// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command common-tls implements the shared high level test lab services (TLS) API.
// This depends on a separate implementation of the low level TLS wiring API.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
)

var (
	port          = flag.Int("port", 0, "Port to listen to")
	wiringPort    = flag.Int("wiring-port", 0, "Port for the TLS wiring service")
	sshKey        = flag.String("ssh-key", "", "Path to SSH key for DUTs (no auth if unset)")
	serverTimeout = flag.Duration("server-timeout", 0, "Maximum duration for which to allow the server to run (<=0 to run indefinitely)")
)

func main() {
	if err := innerMain(); err != nil {
		log.Fatalf("common-tls: %s", err)
	}
}

func innerMain() error {
	flag.Parse()
	// We need to make sure that something eventually terminates this program,
	// since it cannot be guaranteed that the test_runner builder will send the
	// requisite process signal.
	go func() {
		if serverTimeout.Nanoseconds() <= 0 {
			return
		}
		programTimeout := *serverTimeout + time.Minute
		time.Sleep(programTimeout)
		log.Printf("Not-so-gracefully stopping the program due to its %v program timeout", programTimeout)
		os.Exit(1)
	}()

	sshConfig := &ssh.ClientConfig{
		User: "root",
		// We don't care about the host key for DUTs.
		// Attackers intercepting our connections to DUTs is not part
		// of our attack profile.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	if *sshKey != "" {
		m, err := authMethodWithKey(*sshKey)
		if err != nil {
			return err
		}
		sshConfig.Auth = []ssh.AuthMethod{m}
	}
	// TODO(ayatane): Handle if the wiring service connection drops.
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", *wiringPort), grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		return err
	}
	log.Printf("CommonServer listening at address %v", l.Addr())
	s := newServer(conn, sshConfig)
	go func() {
		if serverTimeout.Nanoseconds() <= 0 {
			return
		}
		time.Sleep(*serverTimeout)
		log.Printf("Gracefully stopping the server due to timeout being hit")
		s.GracefulStop()
	}()
	if err := s.Serve(l); err != nil {
		return err
	}
	return nil
}

func authMethodWithKey(keyfile string) (ssh.AuthMethod, error) {
	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, fmt.Errorf("read ssh key: %s", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("read ssh key %s: %s", keyfile, err)
	}
	return ssh.PublicKeys(signer), nil
}

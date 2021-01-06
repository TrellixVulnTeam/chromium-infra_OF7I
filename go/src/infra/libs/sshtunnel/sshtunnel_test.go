// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package sshtunnel

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"golang.org/x/crypto/ssh"
)

func ExampleNewTunnel() {
	const (
		remoteAddr     = "127.0.0.1:0"
		remoteHostAddr = "remoteHostname:22"
	)
	// In this example, we set up an HTTP server on the local machine and
	// expose it to a remote device via ssh tunnel, so on the remote device,
	// we can access the http service on "localhost:<OS assigned port>".
	s := http.Server{}
	http.HandleFunc("/foo", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Hello World!\n")
	})
	// Use port "0" to request the OS to assign an unused port number.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(fmt.Sprintf("Error starting the listener: %s", err))
	}
	defer ln.Close()
	go s.Serve(ln)
	defer s.Shutdown(context.Background())
	// Get the listener's address.
	localAddr := ln.Addr().String()
	// Create the SSH client.
	client, err := ssh.Dial("tcp", remoteHostAddr, &ssh.ClientConfig{})
	if err != nil {
		panic(fmt.Sprintf("Error connecting to %s: %s", client.RemoteAddr().String(), err))
	}
	defer client.Close()
	// Create the SSH tunnel.
	t, err := NewTunnel(localAddr, remoteAddr, client)
	if err != nil {
		panic(fmt.Sprintf("Error setting up SSH tunnel: %s", err))
	}
	defer t.Close()
	// Get address of the listener on the client.
	lnAddr := t.RemoteAddr().String()
	// Send commands over the created tunnel.
	cs, err := client.NewSession()
	if err != nil {
		panic(fmt.Sprintf("Error staring new session on %s: %s", remoteHostAddr, err))
	}
	defer cs.Close()
	err = cs.Run(fmt.Sprintf("curl %s/foo", lnAddr))
	if err != nil {
		panic(fmt.Sprintf("Error running 'curl %s/foo' on %s: %s", lnAddr, remoteHostAddr, err))
	}
}

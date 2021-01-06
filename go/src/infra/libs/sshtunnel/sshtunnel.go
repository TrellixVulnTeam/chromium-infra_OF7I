// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package sshtunnel helps create a SSH tunnels between clients.
package sshtunnel

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Tunnel to create SSH port forwarding between hosts.
type Tunnel struct {
	client   *ssh.Client
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewTunnel sets up SSH port forwarding so that commands sent to the remote
// address (remoteAddr) on client "c" are forwarded to the local address
// (localAddr).
//
// It returns a new Tunnel that can be closed after use.
func NewTunnel(localAddr string, remoteAddr string, c *ssh.Client) (*Tunnel, error) {
	// Listen on remote server port.
	listener, err := c.Listen("tcp", remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("Error listening on %s: %s", remoteAddr, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t := &Tunnel{
		client:   c,
		listener: listener,
		ctx:      ctx,
		cancel:   cancel,
	}
	t.logf("Starting SSH Tunnel.")
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.closeListenerWhenDone()
	}()
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.handleConn(localAddr)
	}()
	return t, nil
}

func (t *Tunnel) closeListenerWhenDone() {
	<-t.ctx.Done()
	if err := t.listener.Close(); err != nil {
		t.logf("%s", err)
	}
}

// handleConn copies the data between the remote and local service for as long
// as the tunnel is not interrupted.
func (t *Tunnel) handleConn(localAddr string) {
	for t.IsAlive() {
		remoteConn, err := t.listener.Accept()
		if err != nil {
			t.logf("%s", err)
			continue
		}
		ctx, cancel := context.WithCancel(t.ctx)
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			defer cancel()
			localConn, err := net.Dial("tcp", localAddr)
			if err != nil {
				t.logf("%s", err)
				return
			}
			t.registerConnToClose(ctx, localConn)
			t.registerConnToClose(ctx, remoteConn)
			t.mirrorConn(remoteConn, localConn)
		}()
	}
}

// mirrorConn is a helper function that mirrors input and output between two
// connections.
func (t *Tunnel) mirrorConn(lConn, rConn net.Conn) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(lConn, rConn); err != nil {
			t.logf("%s", err)
		}
	}()
	if _, err := io.Copy(rConn, lConn); err != nil {
		t.logf("%s", err)
	}
	wg.Wait()
}

func (t *Tunnel) registerConnToClose(ctx context.Context, conn net.Conn) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		<-ctx.Done()
		conn.Close()
	}()
}

// IsAlive checks if the Tunnel is alive. If the tunnel is in the process of
// shutting down but not fully shut down, this method will return false.
func (t *Tunnel) IsAlive() bool {
	return t.ctx.Err() == nil
}

// RemoteAddr returns the address and port on which the service is
// running on the remote device.
func (t *Tunnel) RemoteAddr() net.Addr {
	return t.listener.Addr()
}

// Close closes the tunnel including all resources, and ongoing connections.
func (t *Tunnel) Close() {
	t.logf("Tunnel stopping...")
	t.cancel()
	t.wg.Wait()
	t.logf("Tunnel stopped")
}

func (t *Tunnel) logf(msg string, args ...interface{}) {
	log.Printf("sshtunnel (remote addr %s): %s", t.client.RemoteAddr(), fmt.Sprintf(msg, args...))
}

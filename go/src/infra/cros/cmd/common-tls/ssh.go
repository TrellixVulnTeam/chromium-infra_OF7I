// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"sync"

	"golang.org/x/crypto/ssh"
)

// sshClientPool is a pool of SSH clients to reuse.
// Clients are pooled by the hostname they are connected to.
//
// Users should call Get, which returns a client from the pool if available,
// or creates and returns a new client.
// The returned client is not guaranteed to be good,
// e.g., the connection may have broken while the client was in the pool.
// The user should Put the client back into the pool after use.
//
// If the client appears to be bad, the user should not Put the
// client back into the pool.
// The user should Close the bad client to make sure all resources are
// freed.
//
// The user should Close the pool after use, to free any SSH clients
// in the pool.
type sshClientPool struct {
	sync.Mutex
	pool   map[string][]*ssh.Client
	config *ssh.ClientConfig
}

func newSSHClientPool(c *ssh.ClientConfig) *sshClientPool {
	return &sshClientPool{
		pool:   make(map[string][]*ssh.Client),
		config: c,
	}
}

func (p *sshClientPool) Get(host string) (*ssh.Client, error) {
	p.Lock()
	if n := len(p.pool[host]); n > 0 {
		c := p.pool[host][n-1]
		p.pool[host] = p.pool[host][:n-1]
		p.Unlock()
		return c, nil
	}
	p.Unlock()
	return ssh.Dial("tcp", host, p.config)
}

func (p *sshClientPool) Put(host string, c *ssh.Client) {
	p.Lock()
	p.pool[host] = append(p.pool[host], c)
	p.Unlock()
}

func (p *sshClientPool) Close() error {
	p.Lock()
	defer p.Unlock()
	for hostname, clients := range p.pool {
		for _, c := range clients {
			go c.Close()
		}
		delete(p.pool, hostname)
	}
	return nil
}

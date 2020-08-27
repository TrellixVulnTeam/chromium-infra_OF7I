// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"sync"

	"golang.org/x/crypto/ssh"
)

// client is used by sshClientPool to help users close connections.
type client struct {
	*ssh.Client
	// knownGood is used in deciding if the client can be Put back
	// into the pool.
	knownGood bool
}

func (c *client) close() {
	if c.Client == nil {
		return
	}
	c.Close()
	c.Client = nil
}

// sshClientPool is a pool of SSH clients to reuse.
// Clients are pooled by the hostname they are connected to.
//
// Users should call Get, which returns a client from the pool if available,
// or creates and returns a new client.
// The returned client is not guaranteed to be good,
// e.g., the connection may have broken while the client was in the pool.
// The user should Put the client back into the pool after use.
//
// The user should Put the client back into the pool after use.
// If the user knows the client is still usable, it should set client.knownGood
// to be true before the client is Put back.
// The user should not close the client as sshClientPool will close it.
//
// The user should Close the pool after use, to free any SSH clients
// in the pool.
type sshClientPool struct {
	mu     sync.Mutex
	pool   map[string][]*client
	config *ssh.ClientConfig
}

func newSSHClientPool(c *ssh.ClientConfig) *sshClientPool {
	return &sshClientPool{
		pool:   make(map[string][]*client),
		config: c,
	}
}

// Get returns a client with knownGood as false.
// The user should:
//  1) defer a Put back of the client.
//  2) set the client.knownGood to be true before the client is Put back if the
//     client is usable.
func (p *sshClientPool) Get(host string) (*client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for n := len(p.pool[host]) - 1; n >= 0; n-- {
		c := p.pool[host][n]
		p.pool[host] = p.pool[host][:n]
		if _, err := c.NewSession(); err != nil {
			// This client is probably bad, so close and stop using it.
			go c.close()
			continue
		}
		// knownGood is set to false as the user is responsible for
		// returning a good client into the pool.
		c.knownGood = false
		return c, nil
	}
	c, err := ssh.Dial("tcp", host, p.config)
	return &client{c, false}, err
}

// Put puts the client back into the pool if client.knownGood is true.
// Otherwise, the client is closed.
func (p *sshClientPool) Put(host string, c *client) {
	if c == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if c.knownGood {
		p.pool[host] = append(p.pool[host], c)
	} else {
		c.close()
	}
}

func (p *sshClientPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for hostname, cs := range p.pool {
		for _, c := range cs {
			go c.close()
		}
		delete(p.pool, hostname)
	}
	return nil
}

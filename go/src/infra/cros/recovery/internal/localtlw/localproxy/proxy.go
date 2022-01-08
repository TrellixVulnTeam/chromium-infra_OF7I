// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package proxy provide temp solution to run  existion from local environment
// to execute recovery flows.
package localproxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"
)

var (
	// Pool all proxies used in app.
	proxyPool = make(map[string]*proxy)
)

// Proxy holds info for active running ssh proxy for requested host.
type proxy struct {
	requestHost string
	port        int
	cmd         *exec.Cmd
}

// newProxy creates if not exist or returns existing proxy from pool.
func newProxy(ctx context.Context, host string, port int) *proxy {
	p, ok := proxyPool[host]
	if !ok {
		p = &proxy{
			requestHost: host,
			port:        port,
		}
		p.cmd = exec.CommandContext(ctx, "ssh", "-f", "-N", "-L", fmt.Sprintf("%d:127.0.0.1:22", p.port), fmt.Sprintf("root@%s", p.requestHost))
		initSystemProcAttr(p)
		stderr, err := p.cmd.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("---> local proxy for %q on port %v\n", p.requestHost, p.port)
		if err := p.cmd.Start(); err != nil {
			fmt.Printf("Fail to starte proxy: %s\n", err)
		}
		go func() {
			slurp, _ := io.ReadAll(stderr)
			fmt.Printf("Logs for %q proxy: %s\n", p.requestHost, slurp)
			err := p.cmd.Wait()
			if err != nil {
				fmt.Printf("Proxy %q for %q finished with error: %s\n", p.address(), p.requestHost, err)
			} else {
				fmt.Printf("Proxy %q for %q finished\n", p.address(), p.requestHost)
			}
		}()
		time.Sleep(time.Second)
		proxyPool[p.requestHost] = p
	}
	return p
}

// Close the proxyPool.
func ClosePool() {
	for _, p := range proxyPool {
		closeProxy(p)
	}
}

func (p *proxy) address() string {
	return fmt.Sprintf("root@127.0.0.1:%d", p.port)
}

// Port provides proxy port information.
func (p *proxy) Port() int {
	return p.port
}

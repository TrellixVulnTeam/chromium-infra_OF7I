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
	"net"
	"os/exec"
	"time"

	"go.chromium.org/luci/common/errors"
)

var (
	// Pool all proxies used in app.
	proxyPool = make(map[string]*proxy)
)

// Proxy holds info for active running ssh proxy for requested host.
type proxy struct {
	host         string
	hostIp       string
	hostPort     int
	jumpHost     string
	jumpHostPort int
	cmd          *exec.Cmd
}

// newProxy creates if not exist or returns existing proxy from pool.
func newProxy(ctx context.Context, host string, hostPort int, jumpHost string, jumpHostPort int) *proxy {
	p, ok := proxyPool[host]
	if !ok {
		ip, err := lookupHost(host)
		p = &proxy{
			host:         host,
			hostIp:       ip,
			hostPort:     hostPort,
			jumpHost:     jumpHost,
			jumpHostPort: jumpHostPort,
		}
		if err != nil {
			fmt.Printf("Fail loopup ip for %s: %s\n", host, err)
			proxyPool[p.host] = p
			return p
		}
		// Ex.: the proxy create command will look something like this:
		// "ssh -f -N -L jumpHostPort:127.0.0.1:22 -L hostPort:host:22 root@jumpHost"
		p.cmd = exec.CommandContext(ctx, "ssh", "-f", "-N",
			"-L", fmt.Sprintf("%d:127.0.0.1:22", p.jumpHostPort),
			"-L", fmt.Sprintf("%d:%s:22", p.hostPort, p.hostIp),
			fmt.Sprintf("root@%s", p.jumpHost))
		initSystemProcAttr(p)
		stderr, err := p.cmd.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("---> local proxy for %q on port %v (with jump host of %q on jump host port: %v)\n", p.host, p.hostPort, p.jumpHost, p.jumpHostPort)
		if err := p.cmd.Start(); err != nil {
			fmt.Printf("Fail to starte proxy: %s\n", err)
		}
		go func() {
			slurp, _ := io.ReadAll(stderr)
			fmt.Printf("Logs for %q proxy: %s\n", p.host, slurp)
			err := p.cmd.Wait()
			if err != nil {
				fmt.Printf("Proxy %q for %q finished with error: %s\n", p.address(), p.host, err)
			} else {
				fmt.Printf("Proxy %q for %q finished\n", p.address(), p.host)
			}
		}()
		time.Sleep(time.Second)
		proxyPool[p.host] = p
	}
	return p
}

// ClosePool closes the proxyPool.
func ClosePool() {
	for _, p := range proxyPool {
		closeProxy(p)
	}
}

func (p *proxy) address() string {
	return fmt.Sprintf("root@127.0.0.1:%d", p.hostPort)
}

// Port provides proxy port information.
func (p *proxy) Port() int {
	return p.hostPort
}

// lookupHost is a helper function that looks up the IP address of the provided
// host by using the local resolver.
func lookupHost(hostname string) (string, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return "", errors.Annotate(err, "lookup host %q", hostname).Err()
	}
	if len(addrs) == 0 {
		return "", errors.Reason("lookup host %q: no ip addresses found", hostname).Err()
	}
	return addrs[0], nil
}

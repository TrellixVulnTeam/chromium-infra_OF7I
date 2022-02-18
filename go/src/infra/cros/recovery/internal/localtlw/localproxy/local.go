// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package localproxy

import (
	"context"
	"fmt"

	"infra/cros/recovery/internal/localtlw/ssh"
)

var (
	// Mapping resource of proxy port to host and jumpHost
	// created for local testing.
	hostProxyPortMap = map[string]int{}
	// Incrementally port number used to track which port will be used next.
	lastUsedProxyPort = 2500
)

// RegHost registers the hostname for proxy connections map.
// If hostname i snot known then new proxy will be created and register to map.
func RegHost(ctx context.Context, hostname string, jumpHostname string) error {
	if _, ok := hostProxyPortMap[hostname]; !ok {
		p := newProxy(ctx, hostname, lastUsedProxyPort, jumpHostname, lastUsedProxyPort+1)
		if p.Port() == lastUsedProxyPort {
			lastUsedProxyPort += 2
		}
		hostProxyPortMap[hostname] = p.Port()
	}
	return nil
}

// BuildAddr creates address fro SSH access for execution.
// Ih host present present in the hostProxyPortMap then instead hostname will
// be used proxy address.
func BuildAddr(hostname string) string {
	p, ok := hostProxyPortMap[hostname]
	if ok {
		return fmt.Sprintf("127.0.0.1:%d", p)
	}
	return fmt.Sprintf("%s:%d", hostname, ssh.DefaultPort)
}

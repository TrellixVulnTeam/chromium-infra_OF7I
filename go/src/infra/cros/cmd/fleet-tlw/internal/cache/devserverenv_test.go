// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"net"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseDevserverConfig(t *testing.T) {
	t.Parallel()
	const mockCfg = `
foo = bar
dev_server = http://1.1.1.1:8082,http://1.1.2.1:8082,http://1.1.3.2:8082,http://1.1.3.1:8082
restricted_subnets = 1.1.1.0/24,1.1.2.0/24,1.1.3.0/24
`
	got, err := parseDevserverConfig(strings.NewReader(mockCfg))
	if err != nil {
		t.Errorf("Subnets() failed: %s", err)
	}

	m := net.CIDRMask(24, 32)
	want := []Subnet{
		{
			IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 1, 0), Mask: m},
			Backends: []string{"http://1.1.1.1:8888"},
		},
		{
			IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 2, 0), Mask: m},
			Backends: []string{"http://1.1.2.1:8888"},
		},
		{
			IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 3, 0), Mask: m},
			Backends: []string{"http://1.1.3.1:8888", "http://1.1.3.2:8888"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Subnets() returned unexpected diff (-want +got):\n%s", diff)
	}
}

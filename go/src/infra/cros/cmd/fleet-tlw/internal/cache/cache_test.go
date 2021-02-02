// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"fmt"
	"math"
	"net"
	"strings"
	"testing"
)

type mockEnv struct {
	healthChecker func(string) bool
	subnets       []Subnet
}

func (e mockEnv) Subnets() []Subnet {
	return e.subnets
}

func (e mockEnv) IsBackendHealthy(s string) bool {
	return e.healthChecker(s)
}

func TestAssignBackend_dutInASubnet(t *testing.T) {
	t.Parallel()
	m := net.CIDRMask(24, 32)
	env := mockEnv{
		healthChecker: func(string) bool { return true },
		subnets: []Subnet{
			{
				IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 1, 0), Mask: m},
				Backends: []string{"http://1.1.1.1:8082"},
			},
			{
				IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 2, 0), Mask: m},
				Backends: []string{"http://1.1.2.1:8082"},
			},
			{
				IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 3, 0), Mask: m},
				Backends: []string{"http://1.1.3.1:8082", "http://1.1.3.2:8082"},
			},
		},
	}

	fe := NewFrontend(env)
	const dutAddr, filename = "1.1.3.128", "path/to/file"
	r, err := fe.AssignBackend(dutAddr, filename)
	if err != nil {
		t.Errorf("AssignBackend(%s) failed: %s", dutAddr, err)
	}
	want := "http://1.1.3."
	if !strings.HasPrefix(r, want) {
		t.Errorf("AssignBackend(%s) = %s, not start with '%s'", dutAddr, r, want)
	}
	t.Run("backend failover when unhealthy", func(t *testing.T) {
		// Make the backend previously assigned unhealthy. So it won't be
		// assigned anymore.
		// Don't add 't.Parallel()' here since it depends on the upper level
		// test result.
		env.healthChecker = func(s string) bool { return s != r }
		fe := NewFrontend(env)
		r2, err := fe.AssignBackend(dutAddr, filename)
		if err != nil {
			t.Errorf("AssignBackend(%s) failed: %s", dutAddr, err)
		}
		if r2 == r {
			t.Errorf("AssignBackend(%s) returned unhealthy backend %s", dutAddr, r)
		}
	})
}

func TestAssignBackend_noHealthyBackends(t *testing.T) {
	t.Parallel()
	m := net.CIDRMask(24, 32)
	env := mockEnv{
		healthChecker: func(string) bool { return false },
		subnets: []Subnet{
			{
				IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 1, 0), Mask: m},
				Backends: []string{"http://1.1.1.1:8082"},
			},
			{
				IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 2, 0), Mask: m},
				Backends: []string{"http://1.1.2.1:8082"},
			},
			{
				IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 3, 0), Mask: m},
				Backends: []string{"http://1.1.3.1:8082", "http://1.1.3.2:8082"},
			},
		},
	}
	fe := NewFrontend(env)
	const dutAddr = "1.1.3.128"
	if r, err := fe.AssignBackend(dutAddr, "path/to/file"); err == nil {
		t.Errorf("AssignBackend(%s) succeeded with unhealthy backend %s", dutAddr, r)
	}
}

func TestAssignBackend_balancedLoad(t *testing.T) {
	t.Parallel()
	// Send 101 different request and ensure they are evenly distributed to
	// backends in the subnet.
	env := mockEnv{
		healthChecker: func(string) bool { return true },
		subnets: []Subnet{
			{
				IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 3, 0), Mask: net.CIDRMask(24, 32)},
				Backends: []string{"http://1.1.3.1:8082", "http://1.1.3.2:8082"},
			},
		},
	}
	fe := NewFrontend(env)
	m := make(map[string]int)
	const dutAddr, filename = "1.1.3.128", "path/to/file"
	for i := 0; i < 101; i++ {
		p := fmt.Sprintf("%s-%d", filename, i)
		r, err := fe.AssignBackend(dutAddr, p)
		if err != nil {
			t.Fail()
		}
		m[r]++
	}
	var c []int
	for _, v := range m {
		c = append(c, v)
	}
	if len(c) != 2 {
		t.Errorf("AssignBackend() failed to distribute to two backends; got %d", len(c))
	}
	const delta = 5
	if math.Abs(float64(c[0]-c[1])) > delta {
		t.Errorf("AssignBackend() failed to distribute workload evenly; got %d vs %d", c[0], c[1])
	}
}

func TestAssignBackend_dutNotInAnySubnets(t *testing.T) {
	t.Parallel()
	env := mockEnv{func(string) bool { return true }, nil}
	fe := NewFrontend(env)
	const dutAddr = "100.1.1.1"
	r, err := fe.AssignBackend(dutAddr, "path/to/file")
	if err == nil {
		t.Errorf("AssignBackend(%s) succeeded with DUT out of any subnet, got %s", dutAddr, r)
	}
}

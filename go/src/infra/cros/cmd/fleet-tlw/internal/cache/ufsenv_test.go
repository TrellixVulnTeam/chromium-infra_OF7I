// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/grpc"

	ufsmodels "infra/unifiedfleet/api/v1/models"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

type fakeUFSClient struct {
	ufsapi.FleetClient
	services []*ufsmodels.CachingService
}

func (c fakeUFSClient) ListCachingServices(context.Context, *ufsapi.ListCachingServicesRequest, ...grpc.CallOption) (*ufsapi.ListCachingServicesResponse, error) {
	return &ufsapi.ListCachingServicesResponse{
		CachingServices: c.services,
	}, nil
}

func TestSubnets_multipleSubnets(t *testing.T) {
	t.Parallel()
	c := &fakeUFSClient{services: []*ufsmodels.CachingService{
		{
			Name:           "cachingservice/1.1.1.1",
			Port:           8001,
			ServingSubnets: []string{"1.1.1.0/24", "1.1.2.0/24"},
			State:          ufsmodels.State_STATE_SERVING,
		},
	}}
	env, err := NewUFSEnv(c)
	if err != nil {
		t.Fatalf("NewUFSEnv(fakeClient) failed: %s", err)
	}
	want := []Subnet{
		{
			IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 1, 0), Mask: net.CIDRMask(24, 32)},
			Backends: []string{"http://1.1.1.1:8001"},
		},
		{
			IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 2, 0), Mask: net.CIDRMask(24, 32)},
			Backends: []string{"http://1.1.1.1:8001"},
		},
	}
	got := env.Subnets()
	less := func(a, b Subnet) bool { return a.IPNet.String() < b.IPNet.String() }
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("Subnets() returned unexpected diff (-want +got):\n%s", diff)
	}

}
func TestSubnets_refresh(t *testing.T) {
	t.Parallel()
	c := &fakeUFSClient{services: []*ufsmodels.CachingService{
		{
			Name:           "cachingservice/1.1.1.1",
			Port:           8001,
			ServingSubnets: []string{"1.1.1.1/24"},
			State:          ufsmodels.State_STATE_SERVING,
		},
	}}
	env, err := NewUFSEnv(c)
	if err != nil {
		t.Fatalf("NewUFSEnv(fakeClient) failed: %s", err)
	}
	want := []Subnet{{
		IPNet:    &net.IPNet{IP: net.IPv4(1, 1, 1, 0), Mask: net.CIDRMask(24, 32)},
		Backends: []string{"http://1.1.1.1:8001"},
	}}
	t.Run("add initial data", func(t *testing.T) {
		got := env.Subnets()
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Subnets() returned unexpected diff (-want +got):\n%s", diff)
		}
	})
	t.Run("expired data will be updated", func(t *testing.T) {
		c.services = []*ufsmodels.CachingService{{
			Name:           "cachingservice/2.2.2.2",
			Port:           8002,
			ServingSubnets: []string{"2.2.2.2/24"},
			State:          ufsmodels.State_STATE_SERVING,
		}}
		t.Run("Subnets won't change when not expired", func(t *testing.T) {
			got := env.Subnets()
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Subnets() returned unexpected diff (-want +got):\n%s", diff)
			}
		})
		t.Run("Subnets will change when expired", func(t *testing.T) {
			// Set the `expire` to an old time to ensure the cache is expired.
			env.(*ufsEnv).expire = time.Time{}
			gotNew := env.Subnets()
			wantNew := []Subnet{{
				IPNet:    &net.IPNet{IP: net.IPv4(2, 2, 2, 0), Mask: net.CIDRMask(24, 32)},
				Backends: []string{"http://2.2.2.2:8002"},
			}}
			if diff := cmp.Diff(wantNew, gotNew); diff != "" {
				t.Errorf("Subnets() returned unexpected diff (-want +got):\n%s", diff)
			}
		})
	})
}

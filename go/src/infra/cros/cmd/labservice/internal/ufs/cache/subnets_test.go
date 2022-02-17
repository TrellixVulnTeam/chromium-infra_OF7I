// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"bytes"
	"net"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

func TestGetSubnets_single(t *testing.T) {
	t.Parallel()

	c := &fakeClient{
		CachingServices: &ufsapi.ListCachingServicesResponse{
			CachingServices: []*ufspb.CachingService{
				{
					Name:           "cachingservice/200.200.200.208",
					Port:           55,
					ServingSubnets: []string{"200.200.200.200/24"},
					State:          ufspb.State_STATE_SERVING,
				},
				{
					Name:           "cachingservice/200.200.200.108",
					Port:           155,
					ServingSubnets: []string{"200.200.200.200/24"},
					State:          ufspb.State_STATE_SERVING,
				},
				{
					Name:           "cachingservice/200.200.100.208",
					Port:           255,
					ServingSubnets: []string{"200.200.100.200/24"},
					State:          ufspb.State_STATE_SERVING,
				},
			},
		},
	}
	subnets := newSubnetsFinder()
	got, err := subnets.getSubnets(c)
	if err != nil {
		t.Fatal(err)
	}
	m := net.CIDRMask(24, 32)
	want := []Subnet{
		{
			IPNet:    &net.IPNet{IP: net.IPv4(200, 200, 200, 0), Mask: m},
			Backends: []address{{Ip: "200.200.200.108", Port: 155}, {Ip: "200.200.200.208", Port: 55}},
		},
		{
			IPNet:    &net.IPNet{IP: net.IPv4(200, 200, 100, 0), Mask: m},
			Backends: []address{{Ip: "200.200.100.208", Port: 255}},
		},
	}
	sort.Slice(got, func(i, j int) bool {
		return bytes.Compare(got[i].IPNet.IP, got[j].IPNet.IP) > 0
	})
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("getSubnets() mismatch (-want +got):\n%s", diff)
	}
}

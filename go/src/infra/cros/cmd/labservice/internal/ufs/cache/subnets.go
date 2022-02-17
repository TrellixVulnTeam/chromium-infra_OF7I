// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"context"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	ufsmodels "infra/unifiedfleet/api/v1/models"
	ufsapi "infra/unifiedfleet/api/v1/rpc"

	"google.golang.org/grpc/metadata"
)

const refreshInterval = time.Hour

func newSubnetsFinder() *subnetsFinder {
	return &subnetsFinder{
		subnets: nil,
	}
}

type subnetsFinder struct {
	expireMu sync.Mutex
	expire   time.Time
	subnets  []Subnet
}

// Subnet is a network in labs (i.e. test VLAN).
// DUTs can only access caching backends in the same Subnet.
type Subnet struct {
	IPNet    *net.IPNet
	Backends []address
}

type address struct {
	Ip   string
	Port int32
}

// getSubnets returns the list of up-to-date subnets and cache servers.
func (e *subnetsFinder) getSubnets(client ufsapi.FleetClient) ([]Subnet, error) {
	if err := e.refreshSubnets(client); err != nil {
		return nil, fmt.Errorf("get subnets: %s", err)
	}
	return e.subnets, nil
}

// refreshSubnets makes sure the internal list of subnets is up-to-date.
func (e *subnetsFinder) refreshSubnets(client ufsapi.FleetClient) error {
	n := time.Now()
	e.expireMu.Lock()
	defer e.expireMu.Unlock()
	if e.subnets != nil && n.Before(e.expire) {
		return nil
	}
	e.expire = n.Add(refreshInterval)
	s, err := fetchCachingSubnets(client)
	if err != nil {
		return fmt.Errorf("refresh subnets: %s", err)
	}
	e.subnets = s
	return nil
}

// fetchCachingSubnets fetches caching services info from UFS and constructs
// caching subnets.
func fetchCachingSubnets(client ufsapi.FleetClient) ([]Subnet, error) {
	cachingServices, err := fetchCachingServicesFromUFS(client)
	if err != nil {
		return nil, fmt.Errorf("fetch caching subnets: %s", err)
	}

	var result []Subnet
	m := make(map[string][]address)
	for _, s := range cachingServices {
		if state := s.GetState(); state != ufsmodels.State_STATE_SERVING {
			continue
		}
		ip, port, subnets, err := extractBackendInfo(s)
		if err != nil {
			return nil, fmt.Errorf("fetch caching subnets: %s", err)
		}
		for _, s := range subnets {
			m[s] = append(m[s], address{
				Ip:   ip,
				Port: port,
			})
		}
	}
	for k, v := range m {
		_, ipNet, err := net.ParseCIDR(k)
		if err != nil {
			return nil, fmt.Errorf("fetch caching subnets: parse subnet %q: %s", k, err)
		}
		sort.Slice(v, func(i, j int) bool {
			return v[i].Ip < v[j].Ip || (v[i].Ip == v[j].Ip && v[i].Port < v[j].Port)
		})
		result = append(result, Subnet{IPNet: ipNet, Backends: v})
		log.Printf("Caching subnet: %q: %#v", k, v)
	}
	return result, nil
}

func fetchCachingServicesFromUFS(c ufsapi.FleetClient) ([]*ufsmodels.CachingService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	md := metadata.Pairs("namespace", "os")
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := c.ListCachingServices(ctx, &ufsapi.ListCachingServicesRequest{})
	if err != nil {
		return nil, fmt.Errorf("fetch caching service from UFS: %s", err)
	}
	return resp.GetCachingServices(), nil
}

// extractBackendInfo extracts the caching service name (http://host:port) and
// the serving subnets from the data structure returned by UFS.
func extractBackendInfo(s *ufsmodels.CachingService) (ip string, port int32, subnets []string, err error) {
	// The name returned has a prefix of "cachingservice/".
	nameParts := strings.Split(s.GetName(), "/")
	if len(nameParts) != 2 {
		return "", 0, nil, fmt.Errorf("extract backend info: wrong format service name: %q", s.GetName())
	}
	port = s.GetPort()
	ip, err = lookupHost(nameParts[1])
	if err != nil {
		return "", 0, nil, fmt.Errorf("extract backend info: %s", err)
	}
	subnets = s.GetServingSubnets()
	return ip, port, subnets, nil
}

// lookupHost looks up the IP address of the provided host by using the local
// resolver.
func lookupHost(hostname string) (string, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return "", fmt.Errorf("look up host: IP of %q: %s", hostname, err)
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("look up host: IP of %q: No addresses found", hostname)
	}
	return addrs[0], nil
}

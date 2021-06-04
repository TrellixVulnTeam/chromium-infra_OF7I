// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/metadata"

	ufsmodels "infra/unifiedfleet/api/v1/models"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

const refreshInterval = time.Hour

// NewUFSEnv creates an instance of Environment for caching services registered
// in UFS.
// It caches the result to prevent frequent access to UFS. It updates the cache
// regularly.
func NewUFSEnv(c ufsapi.FleetClient) (Environment, error) {
	e := &ufsEnv{client: c}
	if err := e.refreshSubnets(); err != nil {
		return nil, fmt.Errorf("NewUFSEnv: %s", err)
	}
	return e, nil
}

type ufsEnv struct {
	client   ufsapi.FleetClient
	expireMu sync.Mutex
	expire   time.Time
	subnets  []Subnet
}

func (e *ufsEnv) Subnets() []Subnet {
	if err := e.refreshSubnets(); err != nil {
		log.Printf("UFSEnv: fallback to cached data due to refresh failure: %s", err)
	}
	return e.subnets
}

func (e *ufsEnv) IsBackendHealthy(s string) bool {
	// We registered all devservers to UFS, so we still need to check the server
	// health.
	// TODO(guocb): We can remove this function after we remove all devservers
	// from UFS.

	// We think the backend is healthy as long as it responds.
	// Due to restricted subnets, TLW may not access caching server via HTTP.
	// Instead we use SSH and issue a `curl` command remotely.
	u, _ := url.Parse(s) // `s` is a verified URL string, so no worries about parsing error.
	host, _, _ := net.SplitHostPort(u.Host)
	err := exec.Command("ssh", host, "curl", s).Run()
	return err == nil
}

func (e *ufsEnv) refreshSubnets() error {
	n := time.Now()
	e.expireMu.Lock()
	defer e.expireMu.Unlock()
	if n.Before(e.expire) {
		return nil
	}
	e.expire = n.Add(refreshInterval)

	s, err := e.fetchCachingSubnets()
	if err != nil {
		return fmt.Errorf("refresh subnets: %s", err)
	}
	e.subnets = s
	return nil
}

// fetchCachingSubnets fetches caching services info from UFS and constructs
// caching subnets.
func (e *ufsEnv) fetchCachingSubnets() ([]Subnet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cachingServices, err := fetchCachingServicesFromUFS(ctx, e.client)
	if err != nil {
		return nil, fmt.Errorf("fetch caching subnets: %s", err)
	}
	var result []Subnet
	m := make(map[string][]string)
	for _, s := range cachingServices {
		if state := s.GetState(); state != ufsmodels.State_STATE_SERVING {
			continue
		}
		svc, subnet, err := extractBackendInfo(s)
		if err != nil {
			return nil, err
		}
		m[subnet] = append(m[subnet], svc)
	}
	for k, v := range m {
		_, ipNet, err := net.ParseCIDR(k)
		if err != nil {
			return nil, fmt.Errorf("fetch caching subnets: parse subnet %q: %s", k, err)
		}
		sort.Strings(v)
		result = append(result, Subnet{IPNet: ipNet, Backends: v})
		log.Printf("Caching subnet: %q: %#v", k, v)
	}
	return result, nil
}

// extractBackendInfo extracts the caching service name (http://host:port) and
// the serving subnet from the data structure returned by UFS.
func extractBackendInfo(s *ufsmodels.CachingService) (name, subnet string, err error) {
	// The name returned has a prefix of "cachingservice/".
	nameParts := strings.Split(s.GetName(), "/")
	if len(nameParts) != 2 {
		return "", "", fmt.Errorf("extract cache backend info: wrong format service name: %q", s.GetName())
	}
	port := strconv.Itoa(int(s.GetPort()))
	name = fmt.Sprintf("http://%s", net.JoinHostPort(nameParts[1], port))
	subnet = s.GetServingSubnet()
	return name, subnet, nil
}

func fetchCachingServicesFromUFS(ctx context.Context, c ufsapi.FleetClient) ([]*ufsmodels.CachingService, error) {
	md := metadata.Pairs("namespace", "os")
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := c.ListCachingServices(ctx, &ufsapi.ListCachingServicesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list caching service from UFS: %s", err)
	}
	return resp.GetCachingServices(), nil
}

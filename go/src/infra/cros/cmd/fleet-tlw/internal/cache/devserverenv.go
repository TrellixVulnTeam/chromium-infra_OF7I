// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
)

// AutotestConfig is the autotest config file name.
const AutotestConfig = "/usr/local/autotest/shadow_config.ini"

// NewDevserverEnv creates an instance of Environment for devserver based
// caching.
func NewDevserverEnv(configFilename string) (Environment, error) {
	f, err := os.Open(configFilename)
	if err != nil {
		return nil, fmt.Errorf("new devserver env: %s", err)
	}
	defer f.Close()

	s, err := parseDevserverConfig(f)
	if err != nil {
		return nil, fmt.Errorf("new devserver env: %s", err)
	}
	return devserverEnv{subnets: s}, nil
}

type devserverEnv struct {
	subnets []Subnet
}

func (e devserverEnv) Subnets() []Subnet {
	// Make a copy of 'e.subnets' to prevent being modifed by a caller.
	s := make([]Subnet, len(e.subnets))
	copy(s, e.subnets)
	return s
}

func parseDevserverConfig(r io.Reader) ([]Subnet, error) {
	const devserverCfg, subnetCfg = "dev_server = ", "restricted_subnets = "
	var devservers, subnets string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if len(devservers) != 0 && len(subnets) != 0 {
			break
		}
		l := scanner.Text()
		switch {
		case strings.HasPrefix(l, devserverCfg):
			devservers = strings.TrimSpace(strings.TrimPrefix(l, devserverCfg))
		case strings.HasPrefix(l, subnetCfg):
			subnets = strings.TrimSpace(strings.TrimPrefix(l, subnetCfg))
		}
	}
	if len(devservers) == 0 {
		return nil, fmt.Errorf("dev_server config in shadow_config.ini is empty")
	}
	if len(subnets) == 0 {
		return nil, fmt.Errorf("restricted_subnets config in shadow_config.ini is empty")
	}
	// Register all subnets.
	var ss []Subnet
	for _, s := range strings.Split(subnets, ",") {
		_, ipNet, err := net.ParseCIDR(s)
		if err != nil {
			return nil, fmt.Errorf("cannot parse subnet: %s", err)
		}
		ss = append(ss, Subnet{IPNet: ipNet})
	}

	// Group devservers by its subnet.
nextDevserver:
	for _, ds := range strings.Split(devservers, ",") {
		u, err := url.Parse(ds)
		if err != nil {
			return nil, err
		}
		// We actually use 'gs_cache', so replace the devserver port with
		// gs_cache port.
		host, _, _ := net.SplitHostPort(u.Host)
		const gsCachePort = "8888"
		u.Host = net.JoinHostPort(host, gsCachePort)

		ip := net.ParseIP(host)
		for i := range ss {
			if ss[i].IPNet.Contains(ip) {
				ss[i].Backends = append(ss[i].Backends, u.String())
				continue nextDevserver
			}
		}
	}
	for _, s := range ss {
		sort.Strings(s.Backends)
	}
	return ss, nil

}

func (e devserverEnv) IsBackendHealthy(s string) bool {
	// We think the backend is healthy as long as it responds.
	_, err := http.Get(s)
	return err == nil
}

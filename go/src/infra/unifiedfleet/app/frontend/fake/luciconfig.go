// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package fake

import (
	"context"
	"net/url"

	luciconfig "go.chromium.org/luci/config"
)

// LuciConfigClient mocks the luciconfig.Interface
type LuciConfigClient struct {
}

const dcsContent = `
datacenter: "dc1"
datacenter: "dc2"
datacenter: "dc3"
`

const dcContent = `
name: "ATL97"
description: "Chrome Golo in Atlanta"
kvm {
  name: "cr20-kvm1"
  platform: "Raritan DKX3"
  rack: "cr20"
  mac_address: "00:0d:5d:0d:ee:79"
  ipv4: "192.168.50.28"
  state: SERVING
}
kvm {
  name: "cr22-kvm1"
  platform: "Raritan DKX3"
  rack: "cr22"
  mac_address: "00:0d:5d:10:a6:d7"
  ipv4: "192.168.50.49"
  state: SERVING
}
kvm {
  name: "cr22-kvm2"
  platform: "Raritan DKX3"
  rack: "cr22"
  mac_address: "00:0d:5d:10:e9:5c"
  ipv4: "192.168.50.52"
  state: SERVING
}
rack {
  name: "cr20"
  kvm: "cr20-kvm1"
  state: SERVING
  switch {
    name: "eq017.atl97"
    description: "Arista Networks DCS-7050T-52"
    ports: 48
    state: SERVING
  }
}
rack {
  name: "cr22"
  kvm: "cr22-kvm1"
  state: SERVING
  switch {
    name: "eq041.atl97"
    description: "Arista Networks DCS-7050TX-64"
    ports: 48
    state: SERVING
  }
  switch {
    name: "eq050.atl97"
    description: "Arista Networks DCS-7050TX-64"
    ports: 48
    state: SERVING
  }
  switch {
    name: "eq113.atl97"
    description: "Arista Networks DCS-7050TX-64"
    ports: 48
    state: SERVING
  }
}
`

const vlanContent = `
vlan {
  id: 40
  alias: "vlan-esx"
  cidr_block: "192.168.40.0/22"
  state: SERVING
}
vlan {
  id: 20
  alias: "vlan-dmz"
  cidr_block: "192.168.20.0/24"
  state: SERVING
}
vlan {
  id: 144
  alias: "vlan-master4"
  cidr_block: "192.168.144.0/22"
  state: SERVING
}
`

// GetConfig returns a config at a path in a config set
func (c *LuciConfigClient) GetConfig(ctx context.Context, configSet luciconfig.Set, path string, metaOnly bool) (*luciconfig.Config, error) {
	switch path {
	case "fakeDatacenter.cfg":
		return &luciconfig.Config{
			Content: dcContent,
		}, nil
	case "datacenters.cfg":
		return &luciconfig.Config{
			Content: dcsContent,
		}, nil
	case "fakeVlans.cfg":
		return &luciconfig.Config{
			Content: vlanContent,
		}, nil
	}
	return nil, nil
}

// GetConfigByHash returns the contents of a config
func (c *LuciConfigClient) GetConfigByHash(ctx context.Context, contentHash string) (string, error) {
	return "", nil
}

// GetConfigSetLocation returns the URL location of a config set.
func (c *LuciConfigClient) GetConfigSetLocation(ctx context.Context, configSet luciconfig.Set) (*url.URL, error) {
	return &url.URL{}, nil
}

// GetProjectConfigs returns all the configs at the given path
func (c *LuciConfigClient) GetProjectConfigs(ctx context.Context, path string, metaOnly bool) ([]luciconfig.Config, error) {
	return nil, nil
}

// GetProjects returns all the registered projects in the configuration service.
func (c *LuciConfigClient) GetProjects(ctx context.Context) ([]luciconfig.Project, error) {
	return nil, nil
}

// GetRefConfigs returns the config at the given path
func (c *LuciConfigClient) GetRefConfigs(ctx context.Context, path string, metaOnly bool) ([]luciconfig.Config, error) {
	return nil, nil
}

// GetRefs returns the list of refs for a project.
func (c *LuciConfigClient) GetRefs(ctx context.Context, projectID string) ([]string, error) {
	return nil, nil
}

// ListFiles returns the list of files for a config set.
func (c *LuciConfigClient) ListFiles(ctx context.Context, configSet luciconfig.Set) ([]string, error) {
	return nil, nil
}

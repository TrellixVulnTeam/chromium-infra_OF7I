// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hostinfo

import (
	"context"
	"fmt"

	grpc "google.golang.org/grpc"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"

	"infra/libs/skylab/inventory"
)

// InventoryClient is a client that knows how to resolve a DUT hostname to information about the DUT.
// Its prototypical implementation is inventoryclient.InventoryClientV2.
type InventoryClient interface {
	GetDutInfo(ctx context.Context, id string, byHostname bool) (*inventory.DeviceUnderTest, error)
}

// AdminClient is a client that knows how to respond to the GetStableVersion RPC call.
// Its prototypical implementation is fleet.InventoryClient.
type AdminClient interface {
	GetStableVersion(ctx context.Context, in *fleet.GetStableVersionRequest, opts ...grpc.CallOption) (*fleet.GetStableVersionResponse, error)
}

// Getter is a container for the clients needed to construct the host_info_store contents for a given hostname.
type Getter struct {
	ic InventoryClient
	ac AdminClient
}

// NewGetter constructs a getter in the default configuration.
func NewGetter(
	ic InventoryClient,
	ac AdminClient,
) *Getter {
	g := &Getter{}
	g.ic = ic
	g.ac = ac
	return g
}

// GetContentsForHostname gets the entire hostinfostore file contents for a given hostname
// as a string.
func (g *Getter) GetContentsForHostname(ctx context.Context, hostname string) (string, error) {
	if hostname == "" {
		return "", fmt.Errorf("hostname cannot be empty")
	}
	if g.ic == nil {
		return "", fmt.Errorf("no Inventory client for dut-info")
	}
	if g.ac == nil {
		return "", fmt.Errorf("no Inventory client for stable version")
	}

	di, err := g.ic.GetDutInfo(ctx, hostname, true)
	if err != nil {
		return "", err
	}

	hi := ConvertDut(di)

	hi.StableVersions, err = g.GetStableVersionForHostname(ctx, hostname)
	if err != nil {
		return "", err
	}

	bytes, err := MarshalIndent(hi)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// GetStableVersionForHostname gets the stable version info for a given hostname.
func (g *Getter) GetStableVersionForHostname(ctx context.Context, hostname string) (map[string]string, error) {
	if g.ac == nil {
		return nil, fmt.Errorf("no Inventory client for stable version")
	}
	if hostname == "" {
		return nil, fmt.Errorf("hostname cannot be empty")
	}

	res, err := g.ac.GetStableVersion(ctx, &fleet.GetStableVersionRequest{
		Hostname: hostname,
	})
	if err != nil {
		return nil, err
	}

	return extractStableVersionFromResponse(res), nil
}

func extractStableVersionFromResponse(res *fleet.GetStableVersionResponse) map[string]string {
	return map[string]string{
		"cros":       res.GetCrosVersion(),
		"faft":       res.GetFaftVersion(),
		"firmware":   res.GetFirmwareVersion(),
		"servo-cros": res.GetServoCrosVersion(),
	}
}

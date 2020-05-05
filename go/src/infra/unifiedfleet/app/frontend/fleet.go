// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"golang.org/x/net/context"

	luciconfig "go.chromium.org/luci/config"
)

const (
	machineCollection string = "machines"
)

// CfgInterfaceFactory is a contsructor for a luciconfig.Interface
// For potential unittest usage
type CfgInterfaceFactory func(ctx context.Context) luciconfig.Interface

// FleetServerImpl implements the configuration server interfaces.
type FleetServerImpl struct {
	cfgInterfaceFactory CfgInterfaceFactory
}

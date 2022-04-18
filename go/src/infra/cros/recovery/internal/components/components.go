// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package components

import (
	"context"
	"time"

	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
)

// Runner defines the type for a function that will execute a command
// on a host, and returns the result as a single line.
type Runner func(context.Context, time.Duration, string, ...string) (string, error)

// Pinger defines the type for a function that will execute a ping command
// on a host, and returns error if something went wrong.
type Pinger func(ctx context.Context, count int) error

// Servod defines the interface to communicate with servod daemon.
type Servod interface {
	// Call calls servod method with params.
	Call(ctx context.Context, method string, args ...interface{}) (*xmlrpc.Value, error)
	// Get read value by requested command.
	Get(ctx context.Context, cmd string) (*xmlrpc.Value, error)
	// Set sets value to provided command.
	Set(ctx context.Context, cmd string, val interface{}) error
	// Has verifies that command is known.
	// Error is returned if the control is not listed in the doc.
	Has(ctx context.Context, command string) error
	// Port provides port used for running servod daemon.
	Port() int
}

// CrosVersionInfo holds information for ChromeOS devices.
type CrosVersionInfo struct {
	OSImage   string
	FwImage   string
	FwVersion string
}

// Versioner defines the interface to receive versions information per request.
type Versioner interface {
	// Cros return version info for request Chrome OS device.
	Cros(ctx context.Context, resource string) (*CrosVersionInfo, error)
}

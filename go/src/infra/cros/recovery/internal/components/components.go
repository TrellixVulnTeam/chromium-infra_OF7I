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
type Runner func(ctx context.Context, cmd string, timeout time.Duration) (string, error)

// Servod defines the interface to communicate with servod daemon.
type Servod interface {
	// Get read value by requested command.
	Get(ctx context.Context, cmd string) (*xmlrpc.Value, error)
	// Set sets value to provided command.
	Set(ctx context.Context, cmd string, val *xmlrpc.Value) error
	// Port provides port used for running servod daemon.
	Port() int
}

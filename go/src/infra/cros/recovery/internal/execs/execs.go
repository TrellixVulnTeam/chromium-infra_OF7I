// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package execs provides collection of execution functions for actions and ability to execute them.
package execs

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/tlw"
)

// exec represents an execution function of the action.
// The single exec can be associated with one or more actions.
type exec func(ctx context.Context, args *RunArgs) error

var (
	execMap = make(map[string]exec)
)

// RunArgs holds input arguments for an exec function.
type RunArgs struct {
	DUT    *tlw.Dut
	Access tlw.Access
}

// Run runs exec function provided by this package by name.
func Run(ctx context.Context, name string, args *RunArgs) error {
	e, ok := execMap[name]
	if !ok {
		return errors.Reason("exec %q: not found", name).Err()
	}
	return e(ctx, args)
}

// Exist check if exec function with name is present.
func Exist(name string) bool {
	_, ok := execMap[name]
	return ok
}

// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build windows

package main

import (
	"context"
	"os"

	"google.golang.org/grpc"
)

var handledSignals = []os.Signal{}

func handleSignal(ctx context.Context, gs *grpc.Server, sig os.Signal) {
	panic("not implemented for windows")
}

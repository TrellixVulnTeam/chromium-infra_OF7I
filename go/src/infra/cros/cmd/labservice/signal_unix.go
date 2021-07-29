// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package main

import (
	"context"
	"os"

	"golang.org/x/sys/unix"
	"google.golang.org/grpc"

	"infra/cros/cmd/labservice/internal/log"
)

var handledSignals = []os.Signal{unix.SIGINT, unix.SIGHUP, unix.SIGTERM, unix.SIGQUIT}

func handleSignal(ctx context.Context, gs *grpc.Server, sig os.Signal) {
	log.Infof(ctx, "Got signal %s", sig)
	switch sig {
	case unix.SIGINT, unix.SIGHUP:
		gs.GracefulStop()
	default:
		gs.Stop()
	}
}

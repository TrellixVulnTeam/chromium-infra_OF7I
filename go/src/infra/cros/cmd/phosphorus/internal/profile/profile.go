// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build linux

package profile

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/profile"
)

type sighandler struct {
	sigs     chan os.Signal
	profiler interface{ Stop() }
}

var _sighandler *sighandler

// Register creates a handler to catch SIGUSR1 and SIGUSR2 signals to start and
// stop profiling, respectively.
func Register(options ...func(*profile.Profile)) {
	if _sighandler != nil {
		return
	}

	_sighandler = &sighandler{
		sigs: make(chan os.Signal, 1),
	}

	signal.Notify(_sighandler.sigs, syscall.SIGUSR1, syscall.SIGUSR2)

	go func() {
		// Create closure to shutdown the current profiler.  We have to call it
		// before the program exits as well, so defer execution until this
		// goroutine quits
		stopProfiler := func() {
			if _sighandler.profiler != nil {
				_sighandler.profiler.Stop()
				_sighandler.profiler = nil
			}
		}
		defer stopProfiler()

		for {
			switch sig := <-_sighandler.sigs; sig {
			case syscall.SIGUSR1:
				stopProfiler()
				_sighandler.profiler = profile.Start(options...)

			case syscall.SIGUSR2:
				stopProfiler()
			}
		}
	}()
}

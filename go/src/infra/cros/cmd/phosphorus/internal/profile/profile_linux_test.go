// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package profile

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/pkg/profile"
)

func raise(sig os.Signal) error {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return p.Signal(sig)
}

func TestRegister(t *testing.T) {
	tmpPath := "./tmp"

	Register(profile.ProfilePath(tmpPath))
	if _sighandler == nil {
		t.Fatalf("Failed to register profile handler")
	}

	curHandler := _sighandler
	Register()
	if curHandler != _sighandler {
		t.Fatalf("Register isn't idempotent")
	}

	if _sighandler.profiler != nil {
		t.Fatalf("Profiler was not nil to start")
	}

	// The signal handler is async so we have to loop a bit to wait for
	// the profiler to be created. If it hasn't been created after one second,
	// then we'll assume it failed.
	raise(syscall.SIGUSR1)
	for start := time.Now(); time.Since(start) < time.Second; {
		if _sighandler.profiler != nil {
			break
		}
	}

	if _sighandler.profiler == nil {
		t.Fatalf("Profiler wasn't started")
	}

	// Now check that we can stop the profiler as well
	raise(syscall.SIGUSR2)
	for start := time.Now(); time.Since(start) < time.Second; {
		if _sighandler.profiler == nil {
			break
		}
	}

	if _sighandler.profiler != nil {
		t.Fatalf("Profiler wasn't stopped")
	}

	// Cleanup
	os.RemoveAll(tmpPath)
}

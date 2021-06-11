// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package abortsock

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Example for using abortsock package.
func Example() {
	ctx := context.Background()
	s, err := Open("/some/path")
	if err != nil {
		log.Fatalf("Error opening abort socket: %s", err)
	}
	defer s.Close()
	ctx = s.AttachContext(ctx)
}

// Test the full lifecycle of a socket that is aborted.
func TestAbortingSocket(t *testing.T) {
	t.Parallel()
	d, err := ioutil.TempDir("", "abortsock_test")
	if err != nil {
		t.Fatalf("Error creating test directory: %s", err)
	}
	defer os.RemoveAll(d)
	p := filepath.Join(d, "job.sock")
	s, err := Open(p)
	if err != nil {
		t.Fatalf("Error opening socket: %s", err)
	}
	defer s.Close()

	ctx := context.Background()
	ctx, f := context.WithTimeout(ctx, time.Duration(10)*time.Second)
	defer f()
	ctx = s.AttachContext(ctx)
	Abort(s.Path)
	select {
	case <-ctx.Done():
	}
	if ctx.Err() != context.Canceled {
		t.Errorf("Not canceled after we sent abort")
	}
}

// Test the full lifecycle of a socket that is aborted.
func TestClosingSocket(t *testing.T) {
	t.Parallel()
	d, err := ioutil.TempDir("", "abortsock_test")
	if err != nil {
		t.Fatalf("Error creating test directory: %s", err)
	}
	defer os.RemoveAll(d)
	p := filepath.Join(d, "job.sock")
	s, err := Open(p)
	if err != nil {
		t.Fatalf("Error opening socket: %s", err)
	}

	ctx := context.Background()
	ctx, f := context.WithTimeout(ctx, time.Duration(10)*time.Second)
	defer f()
	ctx = s.AttachContext(ctx)
	s.Close()
	select {
	case <-ctx.Done():
	}
	if ctx.Err() != context.Canceled {
		t.Errorf("Not canceled after we closed socket")
	}
}

// Test that opening fails if the path is too long.
func TestOpenSocketPathTooLong(t *testing.T) {
	t.Parallel()
	path := strings.Repeat("a", 1+socketPathLimit)
	abortSock, err := Open(path)
	if abortSock != nil {
		t.Errorf("unexpected non-nil abortSock %#v", abortSock)
	}
	if err == nil {
		t.Errorf("expected err to not be nil")
	}
}

// Test that aborting a socket fails if the path is too long.
func TestAbortSocketPathTooLong(t *testing.T) {
	t.Parallel()
	path := strings.Repeat("a", 1+socketPathLimit)
	err := Abort(path)
	if err == nil {
		t.Errorf("expected err to not be nil")
	}
}

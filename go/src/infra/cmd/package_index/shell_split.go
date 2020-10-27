// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package main

import (
	"github.com/google/shlex"
)

// shellSplit splits cmd into a parsed list of args.
// This method should only be run on unix.
func shellSplit(cmd string) ([]string, error) {
	r, err := shlex.Split(cmd)
	if err != nil {
		return nil, err
	}
	return r, nil
}

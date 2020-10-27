// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build windows

package main

import (
	"runtime"
	"syscall"

	"golang.org/x/sys/windows"
)

// shellSplit splits cmd into a parsed list of args.
// This method should only be run on windows.
func shellSplit(cmd string) ([]string, error) {
	if runtime.GOOS != "windows" {
		panic("should only be run on windows")
	}

	u, err := windows.UTF16PtrFromString(cmd)
	if err != nil {
		return nil, err
	}

	var argc int32
	argv, err := syscall.CommandLineToArgv(u, &argc)
	if err != nil {
		return nil, err
	}

	args := make([]string, argc)
	for i, v := range (*argv)[:argc] {
		args[i] = windows.UTF16ToString((*v)[:])
	}
	return args, nil
}

// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build windows

package abortsock

// Open opens and returns an AbortSock.
// Make sure to defer Close on it.
func Open(path string) (*AbortSock, error) {
	panic("not supported on windows")
}

// Abort sends an abort datagram to the socket at the given path.
func Abort(path string) error {
	panic("not supported on windows")
}

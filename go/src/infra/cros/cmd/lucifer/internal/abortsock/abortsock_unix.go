// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package abortsock

import (
	"fmt"
	"net"
)

// The limit for a socket path on Linux is 108 characters including
// a null terminator.
const socketPathLimit = 107

// ValidateSocketPath takes a path and checks whether it is valid.
func validateSocketPath(path string) error {
	// On Linux, a socket path length that's too long produces the following
	// error ".../abort_sock: bind: invalid argument". This error is not very
	// informative, so we check up front in order to provide a better error message.
	if length := len(path); length > socketPathLimit {
		return fmt.Errorf("path exceeds maximum length for Linux (%d > %d)", length, socketPathLimit)
	}
	return nil
}

// Open opens and returns an AbortSock.
// Make sure to defer Close on it.
func Open(path string) (*AbortSock, error) {
	if err := validateSocketPath(path); err != nil {
		return nil, err
	}
	c, err := net.ListenPacket("unixgram", path)
	if err != nil {
		return nil, err
	}
	return &AbortSock{Path: path, PacketConn: c}, nil
}

// Abort sends an abort datagram to the socket at the given path.
func Abort(path string) error {
	if err := validateSocketPath(path); err != nil {
		return err
	}
	c, err := net.Dial("unixgram", path)
	if err != nil {
		return err
	}
	// The value sent does not matter.
	b := []byte("abort")
	_, err = c.Write(b)
	return err
}

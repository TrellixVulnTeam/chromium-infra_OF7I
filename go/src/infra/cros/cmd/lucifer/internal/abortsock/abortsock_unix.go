// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package abortsock

import (
	"net"
)

// Open opens and returns an AbortSock.
// Make sure to defer Close on it.
func Open(path string) (*AbortSock, error) {
	c, err := net.ListenPacket("unixgram", path)
	if err != nil {
		return nil, err
	}
	return &AbortSock{Path: path, PacketConn: c}, nil
}

// Abort sends an abort datagram to the socket at the given path.
func Abort(path string) error {
	c, err := net.Dial("unixgram", path)
	if err != nil {
		return err
	}
	// The value sent does not matter.
	b := []byte("abort")
	_, err = c.Write(b)
	return err
}

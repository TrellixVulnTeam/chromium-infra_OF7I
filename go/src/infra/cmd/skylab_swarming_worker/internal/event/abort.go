// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package event

import (
	"context"
	"log"
	"net"
)

// AbortWhenDone forwards an abort message to an abort socket when provided
// context is Done().
//
// This function spawns a goroutine that is cleaned up when the returned
// CancelFunc is called.
func AbortWhenDone(ctx context.Context, path string) context.CancelFunc {
	cancelCtx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ctx.Done():
			if err := abort(path); err != nil {
				log.Printf("Error sending abort for signal: %s", err)
			}
		case <-cancelCtx.Done():
			return
		}
	}()
	return cancel
}

// abort sends an abort datagram to the socket at the given path.
func abort(path string) error {
	c, err := net.Dial("unixgram", path)
	if err != nil {
		return err
	}
	// The value sent does not matter.
	b := []byte("abort")
	_, err = c.Write(b)
	return err
}

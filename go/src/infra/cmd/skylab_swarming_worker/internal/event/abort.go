// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package event

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
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

// ForwardAbortSignal catches termination signals and forwards them as
// abort messages to an abort socket.  This function spawns a
// goroutine and modifies the process signal handlers.  Both of these
// are cleaned up when the returned Forwarder is closed.
func ForwardAbortSignal(path string) *Forwarder {
	c := make(chan os.Signal, 1)
	notifyOnAbort(c)
	go listenAndAbort(c, path)
	return &Forwarder{c: c}
}

// Forwarder encapsulates cleanup for ForwardAbortSignal.
type Forwarder struct {
	c chan<- os.Signal
}

// Close cleans up signal forwarding stuff.  Subsequent calls do
// nothing.
func (f *Forwarder) Close() {
	if f.c == nil {
		return
	}
	signal.Stop(f.c)
	close(f.c)
	f.c = nil
}

// listenAndAbort sends an abort to an abort socket when signals are
// received.  This function is intended to be used as a goroutine for
// handling signals.  This function returns when the channel is
// closed.
func listenAndAbort(c <-chan os.Signal, path string) {
	for range c {
		if err := abort(path); err != nil {
			log.Printf("Error sending abort for signal: %s", err)
		}
	}
}

// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"time"
)

// ctxFunc is a function that takes no arguments and returns an error indicating
// whether it was successful or not.
type ctxFunc = func(context.Context) error

const (
	// completed means that the task ran to completion.
	completed = "completed"
	// interrupted means that the task did not run to completion.
	interrupted = "interrupted"
)

// callNullaryFuncWithTimeout synchronously calls a unary function with a timeout.
// It returns whatever error was produced by the unary function in question, or a dedicated error
// if the deadline was exceeded.
// The status unambiguously indicates whether the function ran to completion or not.
func callFuncWithTimeout(ctx context.Context, timeout time.Duration, cb ctxFunc) (status string, err error) {
	ctx, cancelHandle := context.WithCancel(ctx)
	defer cancelHandle()
	ch := make(chan error, 1)
	go func() {
		ch <- cb(ctx)
	}()
	select {
	case e := <-ch:
		return completed, e
	case <-time.After(timeout):
		return interrupted, errors.New("deadline exceeded")
	}
}

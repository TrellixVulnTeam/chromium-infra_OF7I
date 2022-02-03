// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"bytes"
	"context"
	"log"
	"os/exec"
	"time"

	"go.chromium.org/luci/common/errors"
)

// RunWithTimeout runs command with timeout limit.
func RunWithTimeout(ctx context.Context, cmd *exec.Cmd, timeout time.Duration) (stdout string, stderr string, err error) {
	newCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cw := make(chan error, 1)
	var se, so bytes.Buffer
	cmd.Stderr = &se
	cmd.Stdout = &so
	defer func() {
		stdout = so.String()
		stderr = se.String()
	}()
	go func() {
		log.Printf("Run cmd: %s", cmd)
		cw <- cmd.Run()
	}()
	select {
	case e := <-cw:
		err = errors.Annotate(e, "run with timeout %s", timeout).Err()
		return
	case <-newCtx.Done():
		err = errors.Reason("run with timeout %s: excited timeout", timeout).Err()
		return
	}
}

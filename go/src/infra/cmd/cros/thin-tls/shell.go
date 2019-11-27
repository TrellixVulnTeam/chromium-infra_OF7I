// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"os/exec"

	"infra/cmd/cros/thin-tls/api"
)

func (s *server) DutShell(req *api.DutShellRequest, stream api.Tls_DutShellServer) error {
	cmd := exec.Command("ssh", s.config.DutHostname, req.GetCommand())
	outw := bufio.NewWriter(dutShellStdoutWriter{stream})
	errw := bufio.NewWriter(dutShellStderrWriter{stream})
	cmd.Stdout = outw
	cmd.Stderr = errw
	status := int32(0)
	if err := cmd.Run(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			status = int32(err.ExitCode())
		} else {
			_ = outw.Flush()
			_ = errw.Flush()
			return err
		}
	}
	if err := outw.Flush(); err != nil {
		return err
	}
	if err := errw.Flush(); err != nil {
		return err
	}
	return stream.Send(&api.DutShellResponse{
		Status: status,
		Exited: true,
	})
}

// dutShellStdoutWriter wraps a DutShell stream as an io.Writer.
type dutShellStdoutWriter struct {
	stream api.Tls_DutShellServer
}

func (w dutShellStdoutWriter) Write(p []byte) (n int, err error) {
	resp := api.DutShellResponse{
		Stdout: p,
	}
	if err := w.stream.Send(&resp); err != nil {
		return 0, fmt.Errorf("dut shell writer: %v", err)
	}
	return len(p), nil
}

// dutShellStderrWriter wraps a DutShell stream as an io.Writer.
type dutShellStderrWriter struct {
	stream api.Tls_DutShellServer
}

func (w dutShellStderrWriter) Write(p []byte) (n int, err error) {
	resp := api.DutShellResponse{
		Stderr: p,
	}
	if err := w.stream.Send(&resp); err != nil {
		return 0, fmt.Errorf("dut shell writer: %v", err)
	}
	return len(p), nil
}

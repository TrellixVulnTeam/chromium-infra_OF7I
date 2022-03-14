// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

//go:build !windows
// +build !windows

package localproxy

import (
	"log"
	"syscall"
)

func closeProxy(p *proxy) {
	if p.cmd == nil {
		log.Printf("Proxy %q is empty\n", p.host)
	} else if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL); err != nil {
		log.Printf("Closing proxy for %q finished with error: %s\n", p.address(), err)
	} else {
		log.Printf("Closing proxy for %q:\n", p.address())
	}
}

func initSystemProcAttr(p *proxy) {
	p.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

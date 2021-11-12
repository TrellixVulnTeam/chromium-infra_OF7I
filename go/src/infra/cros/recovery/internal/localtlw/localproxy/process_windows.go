// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package localproxy

import (
	"log"
	"syscall"
)

func closeProxy(p *proxy) {
	log.Printf("Closing proxy for %q is not implemented yet:\n", p.address())
}

func initSystemProcAttr(p *proxy) {
	p.cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

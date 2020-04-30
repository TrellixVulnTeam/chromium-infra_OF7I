// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logdog

import "log"

// printer implements the printing part of the Logger interface.
type printer struct {
	logger *log.Logger
}

// Print implements the Logger interface.
func (p printer) Print(v ...interface{}) {
	p.logger.Print(v...)
}

// Printf implements the Logger interface.
func (p printer) Printf(format string, v ...interface{}) {
	p.logger.Printf(format, v...)
}

// Println implements the Logger interface.
func (p printer) Println(v ...interface{}) {
	p.logger.Println(v...)
}

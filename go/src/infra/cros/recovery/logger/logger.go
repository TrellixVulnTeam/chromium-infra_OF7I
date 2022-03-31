// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package logger provides an abstract representation of logging interfaces used by recovery lib.
package logger

import (
	"log"
)

// NewLogger creates default logger.
func NewLogger() Logger {
	return &logger{}
}

// logger provides default implementation of Logger interface.
type logger struct{}

// Debugf log message at Debug level.
func (l *logger) Debugf(format string, args ...interface{}) {
	l.print(format, args...)
}

// Infof is like Debug, but logs at Info level.
func (l *logger) Infof(format string, args ...interface{}) {
	l.print(format, args...)
}

// Warningf is like Debug, but logs at Warning level.
func (l *logger) Warningf(format string, args ...interface{}) {
	l.print(format, args...)
}

// Errorf is like Debug, but logs at Error level.
func (l *logger) Errorf(format string, args ...interface{}) {
	l.print(format, args...)
}

// print currently implements logging for all log levels. Filtering
// on log levels has not yet been implemented (b/222863687).
func (l *logger) print(format string, args ...interface{}) {
	log.Printf(format, args...)
}

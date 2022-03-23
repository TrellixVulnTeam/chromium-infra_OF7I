// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package logger provides an abstract representation of logging interfaces used by recovery lib.
package logger

import (
	"log"
	"strings"
	"sync/atomic"
)

// NewLogger creates default logger.
func NewLogger() Logger {
	return &logger{
		indentation: 0,
	}
}

// logger provides default implementation of Logger interface.
type logger struct {
	// if negative then treated as 0.
	indentation int32
}

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

// IndentLogging increment indentation for logger.
func (l *logger) IndentLogging() {
	atomic.AddInt32(&l.indentation, 1)
}

// DedentLogging decrement indentation for logger.
func (l *logger) DedentLogging() {
	atomic.AddInt32(&l.indentation, -1)
}

// print currently implements logging for all log levels. Filtering
// on log levels has not yet been implemented (b/222863687).
func (l *logger) print(format string, args ...interface{}) {
	prefix := l.getIndentationPrefix("  ")
	log.Printf(prefix+format, args...)
}

// getIndentationPrefix returns the prefix to use in logging based on current
// indentation level. If indentStr is empty tab is used.
func (l *logger) getIndentationPrefix(indentStr string) string {
	indentation := atomic.LoadInt32(&l.indentation)
	if indentation <= 0 {
		return ""
	}
	if indentStr == "" {
		indentStr = "\t"
	}
	return strings.Repeat(indentStr, int(indentation))
}

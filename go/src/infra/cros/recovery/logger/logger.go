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

// Logger represents a simple interface for logging data.
type Logger interface {
	// Debug log message at Debug level.
	Debug(format string, args ...interface{})
	// Info is like Debug, but logs at Info level.
	Info(format string, args ...interface{})
	// Warning is like Debug, but logs at Warning level.
	Warning(format string, args ...interface{})
	// Error is like Debug, but logs at Error level.
	Error(format string, args ...interface{})
	// IndentLogging increment indentation for logger.
	IndentLogging()
	// DedentLogging decrement indentation for logger.
	DedentLogging()
}

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

// Debug log message at Debug level.
func (l *logger) Debug(format string, args ...interface{}) {
	l.print(format, args...)
}

// Info is like Debug, but logs at Info level.
func (l *logger) Info(format string, args ...interface{}) {
	l.print(format, args...)
}

// Warning is like Debug, but logs at Warning level.
func (l *logger) Warning(format string, args ...interface{}) {
	l.print(format, args...)
}

// Error is like Debug, but logs at Error level.
func (l *logger) Error(format string, args ...interface{}) {
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

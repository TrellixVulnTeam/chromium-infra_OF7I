// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package logger provides an abstract representation of logging interfaces used by recovery lib.
package logger

import (
	"log"
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
	indentation int
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
	l.indentation += 1
}

// DedentLogging decrement indentation for logger.
func (l *logger) DedentLogging() {
	if l.indentation > 0 {
		l.indentation -= 1
	}
}

// Default logging logic for all levels.
func (l *logger) print(format string, args ...interface{}) {
	i := GetIndent(l.indentation, "\t")
	log.Printf(i+format, args...)
}

// Generate an indent string that will be placed before messages
// Default indent is tab (`\t`).
func GetIndent(i int, indentStr string) string {
	if i == 0 {
		return ""
	}
	if indentStr == "" {
		indentStr = "\t"
	}
	is := []byte(indentStr)
	b := make([]byte, i*len(is))
	for v := 0; v < i; v++ {
		b = append(b, is...)
	}
	return string(b)
}

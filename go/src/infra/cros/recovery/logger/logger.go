// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package logger provides an abstract representation of logging interfaces used by recovery lib.
package logger

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
}

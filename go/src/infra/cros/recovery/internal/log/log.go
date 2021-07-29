// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package log provides a wrapper over logging interfaces to simplify usage in recovery lib.
//
// If Logger not set then default golang log package is using.
package log

import (
	"context"

	"infra/cros/recovery/logger"
)

const (
	// loggerKey is key to access to logger from context.
	loggerKey = "recovery_logger"
)

// WithLogger sets logger to the context.
// If Logger is not provided process will be finished with panic.
func WithLogger(ctx context.Context, logger logger.Logger) context.Context {
	if logger == nil {
		panic("logger is not provided")
	}
	return context.WithValue(ctx, loggerKey, logger)
}

// Get logger from context.
func get(ctx context.Context) logger.Logger {
	if log, ok := ctx.Value(loggerKey).(logger.Logger); ok {
		return log
	}
	return nil
}

// Debug log message at Debug level.
func Debug(ctx context.Context, format string, args ...interface{}) {
	if l := get(ctx); l != nil {
		l.Debug(format, args...)
	}
}

// Info is like Debug, but logs at Info level.
func Info(ctx context.Context, format string, args ...interface{}) {
	if l := get(ctx); l != nil {
		l.Info(format, args...)
	}
}

// Warning is like Debug, but logs at Warning level.
func Warning(ctx context.Context, format string, args ...interface{}) {
	if l := get(ctx); l != nil {
		l.Warning(format, args...)
	}
}

// Error is like Debug, but logs at Error level.
func Error(ctx context.Context, format string, args ...interface{}) {
	if l := get(ctx); l != nil {
		l.Error(format, args...)
	}
}

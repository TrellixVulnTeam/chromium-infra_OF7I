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

// loggerKeyType is a unique type for a context key.
type loggerKeyType string

const (
	// loggerKey is key to access to logger from context.
	loggerKey loggerKeyType = "recovery_logger"
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

// Debugf log message at Debugf level.
func Debugf(ctx context.Context, format string, args ...interface{}) {
	if l := get(ctx); l != nil {
		l.Debugf(format, args...)
	}
}

// Infof is like Debug, but logs at Infof level.
func Infof(ctx context.Context, format string, args ...interface{}) {
	if l := get(ctx); l != nil {
		l.Infof(format, args...)
	}
}

// Warningf is like Debug, but logs at Warningf level.
func Warningf(ctx context.Context, format string, args ...interface{}) {
	if l := get(ctx); l != nil {
		l.Warningf(format, args...)
	}
}

// Errorf is like Debug, but logs at Errorf level.
func Errorf(ctx context.Context, format string, args ...interface{}) {
	if l := get(ctx); l != nil {
		l.Errorf(format, args...)
	}
}

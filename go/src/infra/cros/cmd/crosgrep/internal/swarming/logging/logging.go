// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// LoggingDestKey is the key for the destination for all logging messages.
const loggingDestKey = "loggingDestKey"

// LoggingLevelKey is the key for the destination for all logging messages.
const loggingLevelKey = "loggingLevelKey"

const (
	debugLevel = iota
	infoLevel
	warningLevel
	errorLevel
)

// GetDest returns the logging destination out of the current context.
// The default destination is os.Stderr.
func getDest(ctx context.Context) io.Writer {
	dest, ok := ctx.Value(loggingDestKey).(io.Writer)
	if !ok {
		// TODO(gregorynisbet): Replace os.Stderr with logging sync that understands newlines.
		return os.Stderr
	}
	return dest
}

// GetLevel returns the logging level out of the current context.
// The default level is info (1).
func getLevel(ctx context.Context) int {
	level, ok := ctx.Value(loggingLevelKey).(int)
	if !ok {
		return infoLevel
	}
	return level
}

// NewContextWithLevel changes the log level that is currently in effect program-wide.
func newContextWithLevel(ctx context.Context, level string) (context.Context, error) {
	switch strings.ToLower(level) {
	case "debug":
		return context.WithValue(ctx, loggingLevelKey, debugLevel), nil
	case "", "info":
		return context.WithValue(ctx, loggingLevelKey, infoLevel), nil
	case "warning":
		return context.WithValue(ctx, loggingLevelKey, warningLevel), nil
	case "error":
		return context.WithValue(ctx, loggingLevelKey, errorLevel), nil
	}
	return nil, fmt.Errorf("log level %q is not valid", level)
}

// SetContextVerbosity produces a new context with the given verbosity level.
func SetContextVerbosity(ctx context.Context, verbose bool) context.Context {
	if verbose {
		newCtx, _ := newContextWithLevel(ctx, "debug")
		return newCtx
	}
	newCtx, _ := newContextWithLevel(ctx, "error")
	return newCtx
}

// Writef writes a formatted message and a given level.
func writef(ctx context.Context, level int, fmtString string, args ...interface{}) (int, error) {
	// We only write the message if the message's level is high enough.
	if level >= getLevel(ctx) {
		return fmt.Fprintf(getDest(ctx), fmtString, args...)
	}
	return 0, nil
}

// Debugf writes a message at the debug level.
func Debugf(ctx context.Context, fmtString string, args ...interface{}) (int, error) {
	return writef(ctx, debugLevel, fmtString, args...)
}

// Errorf writes a message at the error level.
func Errorf(ctx context.Context, fmtString string, args ...interface{}) (int, error) {
	return writef(ctx, errorLevel, fmtString, args...)
}

// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logger

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/common/logging/teelogger"

	"infra/cros/recovery/logger"
)

const (
	// DefaultFormat is optional formatting which can be used for logger.
	DefaultFormat = `[%{level:.1s}%{time:2006-01-02T15:04:05:00} %{shortfile:20s}] %{message}`
)

// Logger is interface for logger with closing.
type Logger interface {
	logger.Logger
	Close()
}

// loggerImpl represents local recovery logger implementation.
type loggerImpl struct {
	log logging.Logger
	// callDepth sets desired stack depth (code line at which logging message is reported).
	callDepth int
	// logDir path to directory used to created log files.
	logDir string
	// Format used for logging.
	format string
	// closers to manage resource closing when logging is closing.
	closers []closer
}

// Create custom logger config with custom formatter.
func NewLogger(ctx context.Context, callDepth int, logDir string, stdLevel logging.Level, format string, createFiles bool) (_ context.Context, _ Logger, rErr error) {
	l := &loggerImpl{
		logDir:    logDir,
		callDepth: callDepth,
		format:    format,
	}
	defer func() {
		if rErr != nil && l != nil {
			l.Close()
		}
	}()
	// List of file loggers used for loggings.
	var filtereds []teelogger.Filtered
	if createFiles {
		var err error
		// For demo purposes, create two backend for os.Stderr.
		if ctx, filtereds, err = l.createFileLogger(ctx, createFile, filtereds, logging.Info); err != nil {
			return ctx, l, errors.Annotate(err, "create logger").Err()
		}
		if ctx, filtereds, err = l.createFileLogger(ctx, createFile, filtereds, logging.Debug); err != nil {
			return ctx, l, errors.Annotate(err, "create logger").Err()
		}
	}
	ctx = gologger.StdConfig.Use(ctx)
	ctx = logging.SetLevel(ctx, stdLevel)
	ctx = teelogger.UseFiltered(ctx, filtereds...)
	l.log = logging.Get(ctx)
	return ctx, l, nil
}

// createFileLogger creates file logger based on gologger.
func (l *loggerImpl) createFileLogger(ctx context.Context, fc fileCreator, filtereds []teelogger.Filtered, level logging.Level) (context.Context, []teelogger.Filtered, error) {
	fn := fmt.Sprintf("log.%s", level)
	w, c, err := fc(ctx, l.logDir, fn)
	if err != nil {
		return ctx, filtereds, errors.Annotate(err, "logger for %s", fn).Err()
	}
	// Always register closer first!
	l.closers = append(l.closers, c)
	// gologger reads level from context when it is creating and then limit
	// all messages by that level. Work around is set required level is
	// to set level before create it and cache for future usage.
	ctx = logging.SetLevel(ctx, level)
	lc := &gologger.LoggerConfig{
		Out:    w,
		Format: l.format,
	}
	logger := lc.NewLogger(ctx)
	filtereds = append(filtereds, teelogger.Filtered{
		Factory: func(_ context.Context) logging.Logger { return logger },
		Level:   level,
	})
	return ctx, filtereds, nil
}

// Close log resources.
func (l *loggerImpl) Close() {
	for i := len(l.closers) - 1; i >= 0; i-- {
		l.closers[i]()
	}
	l.closers = nil
}

// Debugf log message at Debug level.
func (l *loggerImpl) Debugf(format string, args ...interface{}) {
	l.log.LogCall(logging.Debug, l.callDepth, format, args)
}

// Infof is like Debugf, but logs at Info level.
func (l *loggerImpl) Infof(format string, args ...interface{}) {
	l.log.LogCall(logging.Info, l.callDepth, format, args)
}

// Warningf is like Debugf, but logs at Warning level.
func (l *loggerImpl) Warningf(format string, args ...interface{}) {
	l.log.LogCall(logging.Warning, l.callDepth, format, args)
}

// Errorf is like Debug, but logs at Error level.
func (l *loggerImpl) Errorf(format string, args ...interface{}) {
	l.log.LogCall(logging.Error, l.callDepth, format, args)
}

// closer is function to close some resource.
type closer func()

type fileCreator func(ctx context.Context, dir, name string) (io.Writer, closer, error)

// createFile creates the file and provide closer to close the file.
func createFile(ctx context.Context, dir, name string) (io.Writer, closer, error) {
	n := filepath.Join(dir, name)
	var closers []closer
	c := func() {
		for i := len(closers) - 1; i >= 0; i-- {
			closers[i]()
		}
		closers = nil
	}
	f, err := os.Create(n)
	if err != nil {
		return nil, nil, err
	}
	closers = append(closers, func() {
		f.Close()
	})
	w := bufio.NewWriter(f)
	closers = append(closers, func() {
		w.Flush()
	})
	return w, c, nil
}

// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package logdog provide an interface for writing LogDog logs.
//
// Since LogDog output is a single stream, this package is not
// goroutine safe.
//
// Also, because it is a single stream, printing from
// any Logger or Step that is not the most recently created, or
// creating a second step at the same nesting level without closing
// the previous one is undefined behavior.
//
// As a rule of thumb, do not call any methods on a Logger or Step
// after calling the Step method, until the returned Step is Closed.
// This is not enforced because it would require adding errors
// everywhere and complicating the API, or adding panics which is
// undesirable (logging errors are not severe enough to be fatal).
package logdog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"infra/cros/cmd/lucifer/internal/annotations"
)

// Logger defines the methods for writing LogDog logs.  The
// implementation may format the output differently if the output
// stream is not being processed for LogDog (e.g., if the output is
// just a text stream).
//
// The provided interface is consistent with top level LogDog output
// not associated with a step.  Since it is not associated with a
// step, you cannot attach step links or indicate step failure.
type Logger interface {
	// RawWriter returns the raw backing Writer.  This can be used
	// for dumping copious log data from sources like a
	// subprocess.
	RawWriter() io.Writer
	// Basic log printing methods.
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	// LogDog specific methods.
	Step(string) Step
	LabeledLog(string, string)
}

// Step defines the methods for writing LogDog logs while a step is
// active.  This provides a superset of the methods available globally
// via Logger.
//
// Step also implements Logger, which means you could pass a Step to a
// hypothetical function that takes a root Logger, and the function
// which would normally log steps under the root would instead log
// substeps under the given Step.  However, you would generally not
// assign a Step to a Logger variable.
type Step interface {
	Logger
	// Close writes out the end of the LogDog step.  Calling any
	// method on a step after it is closed is undefined, including
	// calling Close again.  This method always returns nil (but
	// implements io.Closer).
	Close() error
	Failure()
	Exception()
	AddLink(label, url string)
}

var programName = filepath.Base(os.Args[0])

// realLogger is the real LogDog implementation of Logger.
type realLogger struct {
	w io.Writer
	printer
}

// NewLogger returns a Logger that writes to the given writer, which
// should process LogDog annotations.
func NewLogger(w io.Writer) Logger {
	rl := realLogger{
		w: w,
	}
	prefix := fmt.Sprintf("%s: ", programName)
	rl.logger = log.New(w, prefix, log.LstdFlags)
	return &rl
}

// RawWriter implements the Logger interface.
func (rl *realLogger) RawWriter() io.Writer {
	return rl.w
}

// Step implements the Logger interface.
func (rl *realLogger) Step(name string) Step {
	annotations.SeedStep(rl.w, name)
	annotations.StepCursor(rl.w, name)
	annotations.StepStarted(rl.w)
	return &realStep{
		realLogger: rl,
		name:       name,
	}
}

// LabeledLog implements the Logger interface.
func (rl *realLogger) LabeledLog(label, line string) {
	annotations.StepLogLine(rl.w, label, line)
}

type realStep struct {
	*realLogger
	name       string
	level      int
	parentName string
}

// Close implements the Step interface.
func (rs *realStep) Close() error {
	annotations.StepClosed(rs.w)
	if rs.level > 0 {
		annotations.StepCursor(rs.w, rs.parentName)
	}
	return nil
}

// Failure implements the Step interface.
func (rs *realStep) Failure() {
	annotations.StepFailure(rs.w)
}

// Exception implements the Step interface.
func (rs *realStep) Exception() {
	annotations.StepException(rs.w)
}

// AddLink implements the Step interface.
func (rs *realStep) AddLink(label, url string) {
	annotations.StepLink(rs.w, label, url)
}

// Step implements the Step interface.
func (rs *realStep) Step(name string) Step {
	annotations.SeedStep(rs.w, name)
	annotations.StepCursor(rs.w, name)
	annotations.StepNestLevel(rs.w, rs.level+1)
	annotations.StepStarted(rs.w)
	return &realStep{
		realLogger: rs.realLogger,
		name:       name,
		level:      rs.level + 1,
		parentName: rs.name,
	}
}

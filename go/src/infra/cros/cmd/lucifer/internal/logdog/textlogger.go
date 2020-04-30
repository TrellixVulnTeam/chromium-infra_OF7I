// Copyright 2018 The Chromium OS Authots. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logdog

import (
	"fmt"
	"io"
	"log"
)

// textLogger is the text LogDog implementation of Logger.
type textLogger struct {
	w io.Writer
	printer
}

// NewTextLogger returns a text Logger that writes to the given
// writer.  The output will be formatted for plain text and will not
// contain LogDog annotations.
func NewTextLogger(w io.Writer) Logger {
	tl := textLogger{
		w: w,
	}
	prefix := fmt.Sprintf("%s: ", programName)
	tl.logger = log.New(w, prefix, log.LstdFlags)
	return &tl
}

// RawWriter implements the Logger interface.
func (tl *textLogger) RawWriter() io.Writer {
	return tl.w
}

// LabeledLog implements the Logger interface.
func (tl *textLogger) LabeledLog(label, line string) {
	tl.Printf("LOG %s %s", label, line)
}

// Step implements the Logger interface.
func (tl *textLogger) Step(name string) Step {
	tl.Printf("STEP %s", name)
	return &textStep{
		textLogger: tl,
		name:       name,
	}
}

type textStep struct {
	*textLogger
	name   string
	status string
	// prefix is used for substeps.
	prefix string
}

// Close implements the Step interface.
func (ts *textStep) Close() error {
	if ts.status != "" {
		ts.Printf("STEP %s%s %s", ts.prefix, ts.name, ts.status)
	} else {
		ts.Printf("STEP %s%s OK", ts.prefix, ts.name)
	}
	return nil
}

// Failure implements the Step interface.
func (ts *textStep) Failure() {
	ts.status = "FAIL"
}

// Exception implements the Step interface.
func (ts *textStep) Exception() {
	ts.status = "ERROR"
}

// AddLink implements the Step interface.
func (ts *textStep) AddLink(label, url string) {
	ts.Printf("LINK %s %s", label, url)
}

// Step implements the Step interface.  Substep support is not
// implemented yet, so this method panics.
func (ts *textStep) Step(name string) Step {
	p := fmt.Sprintf("%s%s::", ts.prefix, ts.name)
	ts.Printf("STEP %s%s", p, name)
	return &textStep{
		textLogger: ts.textLogger,
		name:       name,
		prefix:     p,
	}
}

// ConfigForTest configures the text logger for deterministic output
// for tests.
func ConfigForTest(lg Logger) {
	switch lg := lg.(type) {
	case *textLogger:
		lg.logger.SetFlags(0)
		lg.logger.SetPrefix("example: ")
	}
}

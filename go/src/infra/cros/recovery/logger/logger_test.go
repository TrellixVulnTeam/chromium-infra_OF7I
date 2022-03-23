// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package logger provides an abstract representation of logging interfaces used by recovery lib.
package logger

import (
	"bytes"
	"infra/cros/internal/assert"
	"log"
	"os"
	"testing"
)

var want = `
Line 1
  Line 2
      Line 3
    Line 4
Line 5
`

func TestBasicLogging(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	flags := log.Flags()
	log.SetFlags(0)
	defer log.SetFlags(flags)

	l := NewLogger()

	// No indent
	// Empty first line to keep want string neat.
	l.Infof("")
	l.Infof("Line 1")

	// Single indent
	l.IndentLogging()
	l.Infof("Line 2")

	// Multi-indent
	l.IndentLogging()
	l.IndentLogging()
	l.Infof("Line 3")

	// Dedent
	l.DedentLogging()
	l.Infof("Line 4")

	// Negative indent safety check
	l.DedentLogging()
	l.DedentLogging()
	l.DedentLogging()
	l.Infof("Line 5")

	assert.IntsEqual(t, int(l.(*logger).indentation), -1)
	assert.StringsEqual(t, buf.String(), want)
}

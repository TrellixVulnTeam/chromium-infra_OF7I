// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"github.com/maruel/subcommands"
)

// Flags contains flags common to all crosfleet commands.
type Flags struct {
	subcommands.CommandRunBase
}

// Init initializes the flags.
func (c *Flags) Init() {
}

// Parse parses the flags.
func (c *Flags) Parse() error {
	return nil
}

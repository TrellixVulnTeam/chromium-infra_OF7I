// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugs

import (
	"errors"
	"fmt"
	"regexp"
)

// MonorailSystem is the name of the monorail bug tracker system.
const MonorailSystem = "monorail"

// BuganizerSystem is the name of the buganizer bug tracker system.
const BuganizerSystem = "buganizer"

// MonorailBugIDRe matches identifiers of monorail bugs, like
// "{monorail_project}/{numeric_id}".
var MonorailBugIDRe = regexp.MustCompile(`^([a-z0-9\-_]+)/([1-9][0-9]*)$`)

// BuganizerBugIDRe matches identifiers of buganizer bugs (excluding
// the b/), like 1234567890.
var BuganizerBugIDRe = regexp.MustCompile(`^([1-9][0-9]*)$`)

// BugID represents the identity of a bug managed by Weetbix.
type BugID struct {
	// System is the bug tracking system of the bug. This is either
	// "monorail" or "buganizer".
	System string `json:"system"`
	// ID is the bug tracking system-specific identity of the bug.
	// For monorail, the scheme is {project}/{numeric_id}, for
	// buganizer the scheme is {numeric_id}.
	ID string `json:"id"`
}

// Validate checks if BugID is a valid bug reference. If not, it
// returns an error.
func (b *BugID) Validate() error {
	switch b.System {
	case MonorailSystem:
		if !MonorailBugIDRe.MatchString(b.ID) {
			return fmt.Errorf("invalid monorail bug ID %q", b.ID)
		}
	case BuganizerSystem:
		if !BuganizerBugIDRe.MatchString(b.ID) {
			return fmt.Errorf("invalid buganizer bug ID %q", b.ID)
		}
	default:
		return fmt.Errorf("invalid bug tracking system %q", b.System)
	}
	return nil
}

// MonorailID returns the monorail project and ID of the given bug.
// If the bug is not a monorail bug or is invalid, an error is returned.
func (b *BugID) MonorailProjectAndID() (project, id string, err error) {
	if b.System != MonorailSystem {
		return "", "", errors.New("not a monorail bug")
	}
	m := MonorailBugIDRe.FindStringSubmatch(b.ID)
	if m == nil {
		return "", "", errors.New("not a valid monorail bug ID")
	}
	return m[1], m[2], nil
}

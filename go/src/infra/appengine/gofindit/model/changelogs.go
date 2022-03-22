// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package model

import (
	"fmt"
	"regexp"
)

// ChangeLog represents the changes of a revision
type ChangeLog struct {
	Commit         string          `json:"commit"`
	Tree           string          `json:"tree"`
	Parents        []string        `json:"parents"`
	Author         ChangeLogActor  `json:"author"`
	Committer      ChangeLogActor  `json:"committer"`
	Message        string          `json:"message"`
	ChangeLogDiffs []ChangeLogDiff `json:"tree_diff"`
}

type ChangeLogActor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Time  string `json:"time"`
}

type ChangeLogDiff struct {
	Type    ChangeType `json:"type"`
	OldID   string     `json:"old_id"`
	OldMode int        `json:"old_mode"`
	OldPath string     `json:"old_path"`
	NewID   string     `json:"new_id"`
	NewMode int        `json:"new_mode"`
	NewPath string     `json:"new_path"`
}

type ChangeType string

const (
	ChangeType_ADD    = "add"
	ChangeType_MODIFY = "modify"
	ChangeType_COPY   = "copy"
	ChangeType_RENAME = "rename"
	ChangeType_DELETE = "delete"
)

// ChangeLogResponse represents the response from gitiles for changelog.
type ChangeLogResponse struct {
	Log  []*ChangeLog `json:"log"`
	Next string       `json:"next"` // From next revision
}

// GetReviewUrl returns the review URL of the changelog.
func (cl *ChangeLog) GetReviewUrl() (string, error) {
	pattern := regexp.MustCompile("\\nReviewed-on: (https://.+)\\n")
	matches := pattern.FindStringSubmatch(cl.Message)
	if matches == nil {
		return "", fmt.Errorf("Could not find review CL. Message: %s", cl.Message)
	}
	return matches[1], nil
}

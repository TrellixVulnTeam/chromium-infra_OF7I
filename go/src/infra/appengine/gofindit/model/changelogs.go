// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package model

// ChangeLog represents the changes of a revision
type ChangeLog struct {
	Commit  string   `json:"commit"`
	Tree    string   `json:"tree"`
	Parents []string `json:"parents"`
	Author  struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Time  string `json:"time"`
	} `json:"author"`
	Committer struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Time  string `json:"time"`
	} `json:"committer"`
	Message  string `json:"message"`
	TreeDiff []struct {
		Type    string `json:"type"`
		OldID   string `json:"old_id"`
		OldMode int    `json:"old_mode"`
		OldPath string `json:"old_path"`
		NewID   string `json:"new_id"`
		NewMode int    `json:"new_mode"`
		NewPath string `json:"new_path"`
	} `json:"tree_diff"`
}

// ChangeLogResponse represents the response from gitiles for changelog
type ChangeLogResponse struct {
	Log  []*ChangeLog `json:"log"`
	Next string       `json:"next"` // From next revision
}

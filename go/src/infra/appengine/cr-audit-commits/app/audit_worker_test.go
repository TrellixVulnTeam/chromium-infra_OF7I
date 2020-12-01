// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"testing"

	"infra/appengine/cr-audit-commits/app/rules"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/impl/memory"
)

type failRule struct{}

func (r failRule) GetName() string {
	return "Fail Rule"
}

func (r failRule) Run(c context.Context, ap *rules.AuditParams, rc *rules.RelevantCommit, cs *rules.Clients) (*rules.RuleResult, error) {
	return nil, errors.New("life can be challenging")
}

func TestRunAccountRules(t *testing.T) {
	Convey("RunAccountRules", t, func() {
		ctx := memory.Use(context.Background())
		rs := rules.AccountRules{
			Account: "*",
			Rules:   []rules.Rule{failRule{}},
		}
		rc := &rules.RelevantCommit{
			CommitHash:       "600dc0de",
			CommitterAccount: "committer@example.com",
			AuthorAccount:    "author@example.com",
			CommitMessage:    "This commit is awesome",
		}
		ap := rules.AuditParams{
			RepoCfg: &rules.RefConfig{
				BaseRepoURL: "https://dummy.googlesource.com/dummy.git",
				GerritURL:   "https://dummy-review.googlesource.com",
				BranchName:  "refs/heads/master",
			},
			RepoState: &rules.RepoState{},
		}
		wp := &workerParams{
			workerFinished: make(chan bool, 1),
		}
		_, err := runAccountRules(ctx, rs, rc, ap, wp)
		So(err, ShouldErrLike, "Fail Rule: life can be challenging\nCommit: https://dummy.googlesource.com/dummy.git/+/600dc0de\nBranch: refs/heads/master\nAuthor: author@example.com\nSubject: \"This commit is awesome\"")
	})
}

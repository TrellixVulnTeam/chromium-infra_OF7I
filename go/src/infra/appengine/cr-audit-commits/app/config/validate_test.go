// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/config/validation"
	cpb "infra/appengine/cr-audit-commits/app/proto"
)

func TestConfigValidator(t *testing.T) {
	validate := func(cfg *cpb.Config) error {
		c := validation.Context{Context: context.Background()}
		validateConfig(&c, cfg)
		return c.Finalize()
	}
	Convey("sampleValidRefConfig is valid", t, func() {
		cfg := createConfig(sampleValidRefConfig)
		So(validate(cfg), ShouldBeNil)
	})

	Convey("validateRefConfig catches errors", t, func() {
		Convey("empty gerrit_host", func() {
			cfg := createConfig(sampleValidRefConfig)
			cfg.RefConfigs["fakeproject"].GerritHost = ""
			So(validate(cfg), ShouldErrLike, "missing gerrit_host")
		})

		Convey("empty gerrit_repo", func() {
			cfg := createConfig(sampleValidRefConfig)
			cfg.RefConfigs["fakeproject"].GerritRepo = ""
			So(validate(cfg), ShouldErrLike, "missing gerrit_repo")
		})

		Convey("empty ref", func() {
			cfg := createConfig(sampleValidRefConfig)
			cfg.RefConfigs["fakeproject"].Ref = ""
			So(validate(cfg), ShouldErrLike, "missing ref")
		})

		Convey("empty starting_commit", func() {
			cfg := createConfig(sampleValidRefConfig)
			cfg.RefConfigs["fakeproject"].StartingCommit = ""
			So(validate(cfg), ShouldErrLike, "missing starting_commit")
		})

		Convey("invalid starting_commit", func() {
			cfg := createConfig(sampleValidRefConfig)
			cfg.RefConfigs["fakeproject"].StartingCommit = "12345"
			So(validate(cfg), ShouldErrLike, "invalid starting_commit")
			cfg.RefConfigs["fakeproject"].StartingCommit = "230942q22f1p8516ab098764c3d8e7e40972c60a"
		})

		Convey("empty monorail_project", func() {
			cfg := createConfig(sampleValidRefConfig)
			cfg.RefConfigs["fakeproject"].MonorailProject = ""
			So(validate(cfg), ShouldErrLike, "missing monorail_project")
		})

		Convey("invalid overwrite_last_known_commit", func() {
			cfg := createConfig(sampleValidRefConfig)
			cfg.RefConfigs["fakeproject"].OverwriteLastKnownCommit = "12345"
			So(validate(cfg), ShouldErrLike, "invalid overwrite_last_known_commit")
		})

		Convey("validateAccountRules catches errors", func() {
			Convey("empty account", func() {
				cfg := createConfig(sampleValidRefConfig)
				cfg.RefConfigs["fakeproject"].Rules["manual-changes"].Account = ""
				So(validate(cfg), ShouldErrLike, "missing account")
			})

			Convey("invalid account", func() {
				cfg := createConfig(sampleValidRefConfig)
				cfg.RefConfigs["fakeproject"].Rules["manual-changes"].Account = "example.com"
				So(validate(cfg), ShouldErrLike, "invalid account")
			})

			Convey("validateRule catches errors", func() {
				Convey("validateChangeReviewed catches errors", func() {
					cfg := createConfig(sampleValidRefConfig)
					rules := cfg.RefConfigs["fakeproject"].Rules["manual-changes"].GetRules()
					rules[3].GetChangeReviewed().Robots[0] = "example.com"
					So(validate(cfg), ShouldErrLike, "invalid robot")
				})

				Convey("validateOnlyMergeApprovedChange catches errors", func() {
					Convey("invalid allowed_user", func() {
						cfg := createConfig(sampleValidRefConfig)
						rules := cfg.RefConfigs["fakeproject"].Rules["manual-changes"].GetRules()
						rules[8].GetOnlyMergeApprovedChange().AllowedUsers[0] = "example.com"
						So(validate(cfg), ShouldErrLike, "invalid allowed_user")
					})
					Convey("invalid allowed_robot", func() {
						cfg := createConfig(sampleValidRefConfig)
						rules := cfg.RefConfigs["fakeproject"].Rules["manual-changes"].GetRules()
						rules[8].GetOnlyMergeApprovedChange().AllowedRobots[0] = "example.com"
						So(validate(cfg), ShouldErrLike, "invalid allowed_robot")
					})
				})

				Convey("validateOnlyModifiesFilesAndDirsRule catches errors", func() {
					Convey("empty name", func() {
						cfg := createConfig(sampleValidRefConfig)
						rules := cfg.RefConfigs["fakeproject"].Rules["manual-changes"].GetRules()
						rules[9].GetOnlyModifiesFilesAndDirsRule().Name = ""
						So(validate(cfg), ShouldErrLike, "missing name")
					})
				})
			})

			Convey("validateNotification catches errors", func() {
				Convey("validateCommentOrFileMonorailIssue catches errors", func() {
					Convey("empty component", func() {
						cfg := createConfig(sampleValidRefConfig)
						notifications := cfg.RefConfigs["fakeproject"].Rules["manual-changes"].GetNotifications()
						notifications[1].GetCommentOrFileMonorailIssue().Components[0] = ""
						So(validate(cfg), ShouldErrLike, "empty component")
					})
					Convey("empty label", func() {
						cfg := createConfig(sampleValidRefConfig)
						notifications := cfg.RefConfigs["fakeproject"].Rules["manual-changes"].GetNotifications()
						notifications[1].GetCommentOrFileMonorailIssue().Labels[0] = ""
						So(validate(cfg), ShouldErrLike, "empty label")
					})
				})
				Convey("validateFileBugForMergeApprovalViolation catches errors", func() {
					Convey("empty component", func() {
						cfg := createConfig(sampleValidRefConfig)
						notifications := cfg.RefConfigs["fakeproject"].Rules["manual-changes"].GetNotifications()
						notifications[2].GetFileBugForMergeApprovalViolation().Components[0] = ""
						So(validate(cfg), ShouldErrLike, "empty component")
					})
					Convey("empty label", func() {
						cfg := createConfig(sampleValidRefConfig)
						notifications := cfg.RefConfigs["fakeproject"].Rules["manual-changes"].GetNotifications()
						notifications[2].GetFileBugForMergeApprovalViolation().Labels[0] = ""
						So(validate(cfg), ShouldErrLike, "empty label")
					})
				})
			})
		})
	})
}

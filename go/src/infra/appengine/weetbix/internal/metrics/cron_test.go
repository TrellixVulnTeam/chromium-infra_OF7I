// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package metrics

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/gae/impl/memory"

	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/config"
	configpb "infra/appengine/weetbix/internal/config/proto"
	"infra/appengine/weetbix/internal/ingestion/control"
	"infra/appengine/weetbix/internal/testutil"
)

func TestGlobalMetrics(t *testing.T) {
	Convey(`With Spanner Test Database`, t, func() {
		ctx := testutil.SpannerTestContext(t)

		ctx = memory.Use(ctx) // For project config in datastore.

		// Setup project configuration.
		projectCfgs := map[string]*configpb.ProjectConfig{
			"project-a": {},
			"project-b": {},
		}
		So(config.SetTestProjectConfig(ctx, projectCfgs), ShouldBeNil)

		// Create some active rules.
		rulesToCreate := []*rules.FailureAssociationRule{
			rules.NewRule(0).WithProject("project-a").WithActive(true).Build(),
			rules.NewRule(1).WithProject("project-a").WithActive(true).Build(),
		}
		err := rules.SetRulesForTesting(ctx, rulesToCreate)
		So(err, ShouldBeNil)

		// Create some ingestion control records.
		reference := time.Now().Add(-1 * time.Minute)
		entriesToCreate := []*control.Entry{
			control.NewEntry(0).
				WithBuildProject("project-a").
				WithPresubmitProject("project-b").
				WithBuildJoinedTime(reference).
				WithPresubmitJoinedTime(reference).
				Build(),
			control.NewEntry(1).
				WithBuildProject("project-b").
				WithBuildJoinedTime(reference).
				WithPresubmitResult(nil).Build(),
			control.NewEntry(2).
				WithPresubmitProject("project-a").
				WithPresubmitJoinedTime(reference).
				WithBuildResult(nil).Build(),
		}
		_, err = control.SetEntriesForTesting(ctx, entriesToCreate)
		So(err, ShouldBeNil)

		err = GlobalMetrics(ctx)
		So(err, ShouldBeNil)
	})
}

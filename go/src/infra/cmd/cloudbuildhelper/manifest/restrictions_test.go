// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package manifest

import (
	"testing"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/flag/stringsetflag"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRestrictions(t *testing.T) {
	t.Parallel()

	call := func(infra Infra, storage, registry, build, notifications []string) []string {
		r := Restrictions{
			storage:       stringsetflag.Flag{Data: stringset.NewFromSlice(storage...)},
			registry:      stringsetflag.Flag{Data: stringset.NewFromSlice(registry...)},
			build:         stringsetflag.Flag{Data: stringset.NewFromSlice(build...)},
			notifications: stringsetflag.Flag{Data: stringset.NewFromSlice(notifications...)},
		}
		return r.CheckInfra(&infra)
	}

	infra := Infra{
		Storage:  "gs://something/a/b/c",
		Registry: "gcr.io/something",
		CloudBuild: CloudBuildConfig{
			Project: "some-project",
		},
		Notify: []NotifyConfig{
			{
				Kind:   "git",
				Repo:   "https://repo.example.com/something",
				Script: "some/script.py",
			},
		},
	}

	Convey("Unrestricted", t, func() {
		violations := call(infra, nil, nil, nil, nil)
		So(violations, ShouldBeEmpty)
	})

	Convey("Passing via direct hits", t, func() {
		violations := call(infra,
			[]string{"gs://something/a/b/c", "gs://something/else"},
			[]string{"gcr.io/something", "gcr.io/else"},
			[]string{"another-project", "some-project"},
			[]string{"git:https://repo.example.com/something/some/script.py"},
		)
		So(violations, ShouldBeEmpty)
	})

	Convey("Passing via prefix hits", t, func() {
		violations := call(infra,
			[]string{"gs://something/", "gs://something/else"},
			[]string{"gcr.io/something"},
			[]string{"some-project"},
			[]string{"git:https://repo.example.com/something/"},
		)
		So(violations, ShouldBeEmpty)
	})

	Convey("Bad storage", t, func() {
		violations := call(infra, []string{"gs://allowed"}, nil, nil, nil)
		So(violations, ShouldResemble, []string{
			`forbidden Google Storage destination "gs://something/a/b/c" (allowed prefixes are ["gs://allowed"])`,
		})
	})

	Convey("Bad registry", t, func() {
		violations := call(infra, nil, []string{"gcr.io/some"}, nil, nil)
		So(violations, ShouldResemble, []string{
			`forbidden Container Registry destination "gcr.io/something" (allowed values are ["gcr.io/some"])`,
		})
	})

	Convey("Bad cloud build", t, func() {
		violations := call(infra, nil, nil, []string{"some"}, nil)
		So(violations, ShouldResemble, []string{
			`forbidden Cloud Build project "some-project" (allowed values are ["some"])`,
		})
	})

	Convey("Bad notify config", t, func() {
		violations := call(infra, nil, nil, nil, []string{"git:https://another"})
		So(violations, ShouldResemble, []string{
			`forbidden notification destination "git:https://repo.example.com/something/some/script.py" (allowed prefixes are ["git:https://another"])`,
		})
	})
}

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

func TestCheckTargetName(t *testing.T) {
	t.Parallel()

	Convey("Unrestricted", t, func() {
		r := Restrictions{}
		So(r.CheckTargetName("blah"), ShouldBeNil)
	})

	Convey("Restricted", t, func() {
		r := Restrictions{
			targets: stringsetflag.Flag{Data: stringset.NewFromSlice("some-prefix/")},
		}
		So(r.CheckTargetName("some-prefix/zzz"), ShouldBeNil)
		So(r.CheckTargetName("another"), ShouldResemble, []string{
			`forbidden target name "another" (allowed prefixes are ["some-prefix/"])`,
		})
	})
}

func TestCheckBuildSteps(t *testing.T) {
	t.Parallel()

	Convey("Unrestricted", t, func() {
		r := Restrictions{}
		So(r.CheckBuildSteps([]*BuildStep{
			{concrete: &CopyBuildStep{}},
			{concrete: &GoBuildStep{}},
			{concrete: &RunBuildStep{}},
			{concrete: &GoGAEBundleBuildStep{}},
		}), ShouldBeNil)
	})

	Convey("Restricted", t, func() {
		r := Restrictions{
			steps: stringsetflag.Flag{Data: stringset.NewFromSlice("copy", "go_gae_bundle")},
		}
		So(r.CheckBuildSteps([]*BuildStep{
			{concrete: &CopyBuildStep{}},
			{concrete: &GoGAEBundleBuildStep{}},
		}), ShouldBeNil)
		So(r.CheckBuildSteps([]*BuildStep{
			{concrete: &CopyBuildStep{}},
			{concrete: &GoBuildStep{}},
			{concrete: &RunBuildStep{}},
			{concrete: &GoGAEBundleBuildStep{}},
		}), ShouldResemble, []string{
			`forbidden build step kind "go_binary" (allowed values are ["copy" "go_gae_bundle"])`,
			`forbidden build step kind "run" (allowed values are ["copy" "go_gae_bundle"])`,
		})
	})
}

func TestCheckInfra(t *testing.T) {
	t.Parallel()

	call := func(infra Infra, storage, registry, notifications []string) []string {
		r := Restrictions{
			storage:       stringsetflag.Flag{Data: stringset.NewFromSlice(storage...)},
			registry:      stringsetflag.Flag{Data: stringset.NewFromSlice(registry...)},
			notifications: stringsetflag.Flag{Data: stringset.NewFromSlice(notifications...)},
		}
		return r.CheckInfra(&infra)
	}

	infra := Infra{
		Storage:  "gs://something/a/b/c",
		Registry: "gcr.io/something",
		Notify: []NotifyConfig{
			{
				Kind:   "git",
				Repo:   "https://repo.example.com/something",
				Script: "some/script.py",
			},
		},
	}

	Convey("Unrestricted", t, func() {
		violations := call(infra, nil, nil, nil)
		So(violations, ShouldBeEmpty)
	})

	Convey("Passing via direct hits", t, func() {
		violations := call(infra,
			[]string{"gs://something/a/b/c", "gs://something/else"},
			[]string{"gcr.io/something", "gcr.io/else"},
			[]string{"git:https://repo.example.com/something/some/script.py"},
		)
		So(violations, ShouldBeEmpty)
	})

	Convey("Passing via prefix hits", t, func() {
		violations := call(infra,
			[]string{"gs://something/", "gs://something/else"},
			[]string{"gcr.io/something"},
			[]string{"git:https://repo.example.com/something/"},
		)
		So(violations, ShouldBeEmpty)
	})

	Convey("Bad storage", t, func() {
		violations := call(infra, []string{"gs://allowed"}, nil, nil)
		So(violations, ShouldResemble, []string{
			`forbidden Google Storage destination "gs://something/a/b/c" (allowed prefixes are ["gs://allowed"])`,
		})
	})

	Convey("Bad registry", t, func() {
		violations := call(infra, nil, []string{"gcr.io/some"}, nil)
		So(violations, ShouldResemble, []string{
			`forbidden Container Registry destination "gcr.io/something" (allowed values are ["gcr.io/some"])`,
		})
	})

	Convey("Bad notify config", t, func() {
		violations := call(infra, nil, nil, []string{"git:https://another"})
		So(violations, ShouldResemble, []string{
			`forbidden notification destination "git:https://repo.example.com/something/some/script.py" (allowed prefixes are ["git:https://another"])`,
		})
	})
}

func TestCheckCloudBuild(t *testing.T) {
	t.Parallel()

	call := func(cb CloudBuildBuilder, build []string) []string {
		r := Restrictions{
			build: stringsetflag.Flag{Data: stringset.NewFromSlice(build...)},
		}
		return r.CheckCloudBuild(&cb)
	}

	cfg := CloudBuildBuilder{
		Project: "some-project",
	}

	Convey("Unrestricted", t, func() {
		violations := call(cfg, nil)
		So(violations, ShouldBeEmpty)
	})

	Convey("Passing via direct hits", t, func() {
		violations := call(cfg, []string{"another-project", "some-project"})
		So(violations, ShouldBeEmpty)
	})

	Convey("Passing via prefix hits", t, func() {
		violations := call(cfg, []string{"some-project"})
		So(violations, ShouldBeEmpty)
	})

	Convey("Bad project", t, func() {
		violations := call(cfg, []string{"some"})
		So(violations, ShouldResemble, []string{
			`forbidden Cloud Build project "some-project" (allowed values are ["some"])`,
		})
	})
}

// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"infra/appengine/cr-rev/config"
	"infra/appengine/cr-rev/models"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"
)

func TestRedirects(t *testing.T) {
	ctx := gaetesting.TestingContext()
	ds := datastore.GetTestable(ctx)
	ds.Consistent(true)
	ds.AutoIndex(true)

	r := newRedirectRules()
	Convey("number redirect", t, func() {
		_, err := r.findRedirectURL(ctx, "/42")
		So(err, ShouldResemble, errors.New("number redirect not implemented"))
	})

	Convey("full hash redirect", t, func() {
		commits := []*models.Commit{
			{
				ID:         "foo-bar-0000000000000000000000000000000000000000",
				CommitHash: "0000000000000000000000000000000000000000",
				Host:       "foo",
				Repository: "bar",
			},
			{
				ID:         "foo-bar/mirror-0000000000000000000000000000000000000000",
				CommitHash: "0000000000000000000000000000000000000000",
				Host:       "foo",
				Repository: "bar/mirror",
			},
			{
				ID:         "foo-baz-0000000000000000000000000000000000000001",
				CommitHash: "0000000000000000000000000000000000000001",
				Host:       "foo",
				Repository: "baz",
			},
			{
				ID:         "foo-baz/mirror-0000000000000000000000000000000000000001",
				CommitHash: "0000000000000000000000000000000000000001",
				Host:       "foo",
				Repository: "baz/not/indexed",
			},
			{
				ID:         "foo-baz/mirror-0000000000000000000000000000000000000002",
				CommitHash: "0000000000000000000000000000000000000002",
				Host:       "foo",
				Repository: "baz/not/indexed",
			},
		}
		datastore.Put(ctx, commits)

		err := config.Override(ctx, &config.Config{
			Hosts: []*config.Host{
				{
					Name: "foo",
					Repos: []*config.Repository{
						{
							Name:     "bar",
							Indexing: &config.Repository_Priority{Priority: true},
						},
						{
							Name:     "baz/not/indexed",
							Indexing: &config.Repository_DoNotIndex{DoNotIndex: true},
						},
					},
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Could not set context: %s", err.Error()))
		}

		Convey("respect priority", func() {
			url, err := r.findRedirectURL(
				ctx, "/0000000000000000000000000000000000000000")
			So(err, ShouldBeNil)
			So(url, ShouldEqual, "https://foo.googlesource.com/bar/+/0000000000000000000000000000000000000000")
		})

		Convey("avoid not indexed", func() {
			url, err := r.findRedirectURL(
				ctx, "/0000000000000000000000000000000000000001")
			So(err, ShouldBeNil)
			So(url, ShouldEqual, "https://foo.googlesource.com/baz/+/0000000000000000000000000000000000000001")
		})

		Convey("redirect to not indexed if only one", func() {
			url, err := r.findRedirectURL(
				ctx, "/0000000000000000000000000000000000000002")
			So(err, ShouldBeNil)
			So(url, ShouldEqual, "https://foo.googlesource.com/baz/not/indexed/+/0000000000000000000000000000000000000002")
		})
	})

	Convey("default not found", t, func() {
		_, err := r.findRedirectURL(
			ctx, "/foo")
		So(err, ShouldEqual, errNoMatch)
	})
}

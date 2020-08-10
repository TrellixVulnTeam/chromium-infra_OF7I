// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRedirects(t *testing.T) {
	r := newRedirectRules()
	Convey("number redirect", t, func() {
		_, err := r.findRedirectURL("/42")
		So(err, ShouldResemble, errors.New("number redirect not implemented"))
	})

	Convey("full hash redirect", t, func() {
		_, err := r.findRedirectURL(
			"/0000000000000000000000000000000000000000")
		So(err, ShouldResemble, errors.New("full commit hash redirect not implemented"))
	})

	Convey("default not found", t, func() {
		_, err := r.findRedirectURL(
			"/foo")
		So(err, ShouldEqual, errNoMatch)
	})
}

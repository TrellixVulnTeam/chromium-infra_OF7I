// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"infra/tricium/api/v1"
	"testing"
)

// These tests read from files on the filesystem, so modifying the tests may
// require modifying the example test files.
const (
	gLinks           string = "test/src/g_links.md"
	goLinks          string = "test/src/go_links.md"
	httpsURLs        string = "test/src/https_urls.md"
	httpURL          string = "test/src/http_single_url.md"
	multipleHTTPURLs string = "test/src/http_multiple_urls.md"
)

func TestHTTPSChecker(t *testing.T) {

	Convey("Produces no comment for g/ link", t, func() {
		results := &tricium.Data_Results{}
		checkHTTPS(gLinks, results)
		So(results.Comments, ShouldBeNil)
	})

	Convey("Produces no comment for file with go/ link", t, func() {
		results := &tricium.Data_Results{}
		checkHTTPS(goLinks, results)
		So(results.Comments, ShouldBeNil)
	})

	Convey("Produces no comment for file with httpsURLs", t, func() {
		results := &tricium.Data_Results{}
		checkHTTPS(httpsURLs, results)
		So(results.Comments, ShouldBeNil)
	})

	Convey("Flags a single http URL", t, func() {
		results := &tricium.Data_Results{}
		checkHTTPS(httpURL, results)
		So(results.Comments, ShouldNotBeNil)
		So(results.Comments[0], ShouldResemble, &tricium.Data_Comment{
			Category:  "HttpsCheck/Warning",
			Message:   ("Nit: Replace http:// URLs with https://"),
			Path:      httpURL,
			StartLine: 5,
			EndLine:   5,
			StartChar: 7,
			EndChar:   24,
		})
	})

	Convey("Flags multiple http URLs", t, func() {
		results := &tricium.Data_Results{}
		checkHTTPS(multipleHTTPURLs, results)
		So(len(results.Comments), ShouldEqual, 2)
		So(results.Comments[1], ShouldResemble, &tricium.Data_Comment{
			Category:  "HttpsCheck/Warning",
			Message:   ("Nit: Replace http:// URLs with https://"),
			Path:      multipleHTTPURLs,
			StartLine: 9,
			EndLine:   9,
			StartChar: 7,
			EndChar:   26,
		})
	})
}

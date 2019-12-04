// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package track

import (
	"context"
	tricium "infra/tricium/api/v1"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	. "github.com/smartystreets/goconvey/convey"
)

func TestComment_UnpackComment(t *testing.T) {
	Convey("Test", t, func() {
		ctx := context.Background()
		out := tricium.Data_Comment{}
		Convey("success", func() {
			c := Comment{}
			data := tricium.Data_Comment{}
			s, err := (&jsonpb.Marshaler{}).MarshalToString(&data)
			So(err, ShouldBeNil)
			c.Comment = []byte(s)
			So(c.UnpackComment(ctx, &out), ShouldBeNil)
		})
		Convey("failures", func() {
			c := Comment{}
			So(c.UnpackComment(ctx, &out), ShouldNotBeNil)
			c.Comment = []byte{0}
			So(c.UnpackComment(ctx, &out), ShouldNotBeNil)
		})
	})
}

func TestExtractFunctionPlatform(t *testing.T) {
	Convey("Test Environment", t, func() {
		functionName := "Lint"
		platform := "UBUNTU"
		f, p, err := ExtractFunctionPlatform(functionName + workerSeparator + platform)
		So(err, ShouldBeNil)
		So(f, ShouldEqual, functionName)
		So(p, ShouldEqual, platform)
		_, _, err = ExtractFunctionPlatform(functionName)
		So(err, ShouldNotBeNil)
	})
}

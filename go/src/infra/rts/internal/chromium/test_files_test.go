// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package chromium

import (
	"bytes"
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"google.golang.org/api/iterator"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestTestFileSet(t *testing.T) {
	t.Parallel()

	Convey("TestFileSet", t, func() {
		ctx := context.Background()

		buf := bytes.NewBuffer(nil)

		expected := []*TestFile{
			{Path: "a.cc"},
			{Path: "b.cc"},
			{Path: "c.cc"},
		}

		// Write expected protos.
		remaining := expected
		err := writeTestFilesFrom(ctx, buf, func(dest interface{}) error {
			if len(remaining) == 0 {
				return iterator.Done
			}
			proto.Merge(dest.(proto.Message), remaining[0])
			remaining = remaining[1:]
			return nil
		})
		So(err, ShouldBeNil)

		// Read protos.
		var actual []*TestFile
		err = ReadTestFiles(buf, func(f *TestFile) error {
			actual = append(actual, f)
			return nil
		})

		So(actual, ShouldResembleProto, expected)
	})
}

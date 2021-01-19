// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package history

import (
	"bytes"
	"io"
	"testing"

	"google.golang.org/protobuf/encoding/prototext"

	evalpb "infra/rts/presubmit/eval/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestReaderWriter(t *testing.T) {
	t.Parallel()
	Convey(`ReaderWriter`, t, func() {
		buf := bytes.NewBuffer(nil)

		records := []*evalpb.Record{
			parseRecord(`test_duration { test_variant { id: "1" } }`),
			parseRecord(`test_duration { test_variant { id: "2" } }`),
			parseRecord(`test_duration { test_variant { id: "3" } }`),
		}

		// Write the records.
		w := NewWriter(buf)
		for _, r := range records {
			So(w.Write(r), ShouldBeNil)
		}
		So(w.Close(), ShouldBeNil)

		// Read the records back.
		r := NewReader(buf)
		defer r.Close()
		for _, expected := range records {
			actual, err := r.Read()
			So(err, ShouldBeNil)
			So(actual, ShouldResembleProto, expected)
		}
		_, err := r.Read()
		So(err, ShouldEqual, io.EOF)
	})
}

func parseRecord(text string) *evalpb.Record {
	rec := &evalpb.Record{}
	err := prototext.Unmarshal([]byte(text), rec)
	So(err, ShouldBeNil)
	return rec
}

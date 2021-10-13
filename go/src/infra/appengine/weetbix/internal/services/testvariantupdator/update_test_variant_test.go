// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantupdator

import (
	"testing"

	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSchedule(t *testing.T) {
	Convey(`TestSchedule`, t, func() {
		ctx, skdr := tq.TestingContext(testutil.TestingContext(), nil)
		RegisterTaskClass()

		realm := "chromium:ci"
		testID := "ninja://test"
		variantHash := "deadbeef"
		task := &taskspb.UpdateTestVariant{
			TestVariantKey: &taskspb.TestVariantKey{
				Realm:       realm,
				TestId:      testID,
				VariantHash: variantHash,
			},
		}
		So(Schedule(ctx, realm, testID, variantHash), ShouldBeNil)
		So(skdr.Tasks().Payloads()[0], ShouldResembleProto, task)
	})
}

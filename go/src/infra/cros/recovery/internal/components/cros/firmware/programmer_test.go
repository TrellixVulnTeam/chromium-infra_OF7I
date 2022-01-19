// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package firmware

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/components/mocks"
	"infra/cros/recovery/logger"
)

func TestNewProgrammer(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	logger := logger.NewLogger()
	Convey("Fail if servod fail to respond to servod", t, func() {
		servod := mocks.NewMockServod(ctrl)
		servod.EXPECT().Get(ctx, "servo_type").Return(nil, errors.Reason("fail to get servo_type!").Err()).Times(1)

		p, err := NewProgrammer(ctx, mockRunner(nil), servod, logger)
		So(p, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
	Convey("Fail as servo_v2 is not supported", t, func() {
		servod := mocks.NewMockServod(ctrl)
		servod.EXPECT().Get(ctx, "servo_type").Return(stringValue("servo_v2"), nil).Times(1)

		p, err := NewProgrammer(ctx, mockRunner(nil), servod, logger)
		So(p, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
	Convey("Creates programmer for servo_v3", t, func() {
		servod := mocks.NewMockServod(ctrl)
		servod.EXPECT().Get(ctx, "servo_type").Return(stringValue("servo_v3"), nil).Times(1)

		p, err := NewProgrammer(ctx, mockRunner(nil), servod, logger)
		So(p, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
	Convey("Creates programmer for servo_v4", t, func() {
		servod := mocks.NewMockServod(ctrl)
		servod.EXPECT().Get(ctx, "servo_type").Return(stringValue("servo_v4"), nil).Times(1)

		p, err := NewProgrammer(ctx, mockRunner(nil), servod, logger)
		So(p, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}

func stringValue(v string) *xmlrpc.Value {
	return &xmlrpc.Value{
		ScalarOneof: &xmlrpc.Value_String_{
			String_: v,
		},
	}
}

func mockRunner(runResponse map[string]string) components.Runner {
	return func(ctx context.Context, cmd string, timeout time.Duration) (string, error) {
		if v, ok := runResponse[cmd]; ok {
			return v, nil
		}
		return "", errors.Reason("Did not found response for %q!", cmd).Err()
	}
}

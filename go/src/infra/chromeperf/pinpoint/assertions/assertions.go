// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package assertions contains GoConvey assertions used by pinpoint.
// Like the GoConvey package, it should be dot-imported into tests.
package assertions

import (
	"fmt"

	"github.com/smartystreets/goconvey/convey"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ShouldBeStatusError(got interface{}, want ...interface{}) string {
	err, ok := got.(error)
	if !ok {
		return fmt.Sprintf("actual value was not of type error: got type %T (value=%v)", got, got)
	}
	wantCode := want[0].(codes.Code)
	s, ok := status.FromError(err)
	if !ok {
		return fmt.Sprintf("error was not a Status error, found %T", err)
	}
	return convey.ShouldEqual(s.Code(), wantCode)
}

// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
//go:generate mockgen -source=service.pb.go -package api -destination service.mock.go CrrevServer
package api

import (
	"net/http"
	"net/url"
	"testing"

	"go.chromium.org/luci/server/router"

	gomock "github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mock := NewMockCrrevServer(mockCtrl)
	s := &restAPIServer{
		grpcServer: mock,
	}

	Convey("Numbering request", t, func() {
		Convey("non-chromim/src", func() {
			expectedReq := &NumberingRequest{
				Host:           "chromium",
				Repository:     "foo",
				PositionRef:    "refs/heads/main",
				PositionNumber: int64(1),
			}
			mock.EXPECT().Numbering(gomock.Any(), gomock.Eq(expectedReq)).Times(1)

			url, _ := url.Parse("/?project=chromium&repo=foo&numbering_identifier=refs/heads/main&number=1")
			c := &router.Context{
				Request: &http.Request{
					URL: url,
				},
			}
			s.handleNumbering(c)
		})

		Convey("chromim/src", func() {
			Convey("before migration", func() {
				Convey("using old ref", func() {
					expectedReq := &NumberingRequest{
						Host:           "chromium",
						Repository:     "chromium/src",
						PositionRef:    "refs/heads/master",
						PositionNumber: int64(1),
					}
					mock.EXPECT().Numbering(gomock.Any(), gomock.Eq(expectedReq)).Times(1)

					url, _ := url.Parse("/?project=chromium&repo=chromium/src&numbering_identifier=refs/heads/master&number=1")
					c := &router.Context{
						Request: &http.Request{
							URL: url,
						},
					}
					s.handleNumbering(c)
				})
				Convey("using new ref", func() {
					expectedReq := &NumberingRequest{
						Host:           "chromium",
						Repository:     "chromium/src",
						PositionRef:    "refs/heads/master",
						PositionNumber: int64(1),
					}
					mock.EXPECT().Numbering(gomock.Any(), gomock.Eq(expectedReq)).Times(1)

					url, _ := url.Parse("/?project=chromium&repo=chromium/src&numbering_identifier=refs/heads/main&number=1")
					c := &router.Context{
						Request: &http.Request{
							URL: url,
						},
					}
					s.handleNumbering(c)
				})
			})
			Convey("chromim/src after migration", func() {
				Convey("using old ref", func() {
					expectedReq := &NumberingRequest{
						Host:           "chromium",
						Repository:     "chromium/src",
						PositionRef:    "refs/heads/main",
						PositionNumber: int64(1000000),
					}
					mock.EXPECT().Numbering(gomock.Any(), gomock.Eq(expectedReq)).Times(1)

					url, _ := url.Parse("/?project=chromium&repo=chromium/src&numbering_identifier=refs/heads/master&number=1000000")
					c := &router.Context{
						Request: &http.Request{
							URL: url,
						},
					}
					s.handleNumbering(c)
				})
				Convey("using new ref", func() {
					expectedReq := &NumberingRequest{
						Host:           "chromium",
						Repository:     "chromium/src",
						PositionRef:    "refs/heads/main",
						PositionNumber: int64(1000000),
					}
					mock.EXPECT().Numbering(gomock.Any(), gomock.Eq(expectedReq)).Times(1)

					url, _ := url.Parse("/?project=chromium&repo=chromium/src&numbering_identifier=refs/heads/main&number=1000000")
					c := &router.Context{
						Request: &http.Request{
							URL: url,
						},
					}
					s.handleNumbering(c)
				})
			})
		})
	})
}

// Copyright 2020 The Chromium Authors.
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
package main

import (
	"context"
	"flag"
	"fmt"
	"infra/chromeperf/pinpoint"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func TestMisconfiguredServer(t *testing.T) {

	Convey("Given a grpc server without a client", t, func() {
		ctx := context.Background()
		l := bufconn.Listen(bufSize)
		s := grpc.NewServer()
		dialer := func(context.Context, string) (net.Conn, error) {
			return l.Dial()
		}
		pinpoint.RegisterPinpointServer(s, &pinpointServer{})
		go func() {
			if err := s.Serve(l); err != nil {
				log.Fatalf("Server startup failed.")
			}
		}()

		Convey("When we connect to the Pinpoint service", func() {
			conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
			So(err, ShouldBeNil)
			defer conn.Close()
			client := pinpoint.NewPinpointClient(conn)

			Convey("Then requests to ScheduleJob will fail with 'misconfigured service'", func() {
				_, err := client.ScheduleJob(ctx, &pinpoint.ScheduleJobRequest{})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "misconfigured service")
			})

		})
	})

	Convey("Given a grpc server with a legacy client not behind the ESP", t, func() {
		ctx := context.Background()
		l := bufconn.Listen(bufSize)
		s := grpc.NewServer()
		dialer := func(context.Context, string) (net.Conn, error) {
			return l.Dial()
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "{}")
		}))
		defer ts.Close()
		httpClient, err := google.DefaultClient(oauth2.NoContext, scopesForLegacy...)
		flag.Set("legacy_pinpoint_service", ts.URL)
		So(err, ShouldBeNil)

		pinpoint.RegisterPinpointServer(s, &pinpointServer{LegacyClient: httpClient})
		go func() {
			if err := s.Serve(l); err != nil {
				log.Fatalf("Server startup failed.")
			}
		}()

		Convey("When we connect to the Pinpoint service", func() {
			conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
			So(err, ShouldBeNil)
			defer conn.Close()
			client := pinpoint.NewPinpointClient(conn)
			Convey("Then requests to ScheduleJob will fail with 'missing required auth header'", func() {
				_, err := client.ScheduleJob(ctx, &pinpoint.ScheduleJobRequest{})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "missing required auth header")
			})
		})
	})

}

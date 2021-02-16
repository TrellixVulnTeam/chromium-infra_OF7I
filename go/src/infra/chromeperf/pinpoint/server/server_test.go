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

package server

import (
	"context"
	"flag"
	"fmt"
	"infra/chromeperf/pinpoint"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"

	. "github.com/smartystreets/goconvey/convey"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func TestServerService(t *testing.T) {
	t.Parallel()
	Convey("Given a grpc server without a client", t, func() {
		ctx := context.Background()
		l := bufconn.Listen(bufSize)
		defer l.Close()
		s := grpc.NewServer()
		defer s.Stop()
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
		defer l.Close()
		s := grpc.NewServer()
		defer s.Stop()
		dialer := func(context.Context, string) (net.Conn, error) {
			return l.Dial()
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "{}")
		}))
		defer ts.Close()
		flag.Set("legacy_pinpoint_service", ts.URL)
		log.Printf("legacy service = %s", ts.URL)
		pinpoint.RegisterPinpointServer(s, &pinpointServer{LegacyClient: &http.Client{}})
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

func TestGetJob(t *testing.T) {
	t.Parallel()
	Convey("Given a grpc server with a client", t, func() {
		ctx := context.Background()
		l := bufconn.Listen(bufSize)
		defer l.Close()
		s := grpc.NewServer()
		defer s.Stop()
		dialer := func(context.Context, string) (net.Conn, error) {
			return l.Dial()
		}
		// Check the path target before we give a response.
		httpResponses := make(map[string]string)

		mockResponse := func(path, response string) {
			httpResponses[path] = response
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("TEST: %s", r.URL.String())
			resp, found := httpResponses[r.URL.Path]
			if !found {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintln(w, "Not found")
				return
			}
			fmt.Fprintln(w, resp)
		}))
		defer ts.Close()
		flag.Set("legacy_pinpoint_service", ts.URL)
		log.Printf("legacy service = %s", ts.URL)
		pinpoint.RegisterPinpointServer(s, &pinpointServer{LegacyClient: &http.Client{}})
		go func() {
			if err := s.Serve(l); err != nil {
				log.Fatalf("Server startup failed.")
			}
		}()

		conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
		So(err, ShouldBeNil)
		defer conn.Close()
		client := pinpoint.NewPinpointClient(conn)

		Convey("When we attempt to get a defined job", func() {
			c, err := ioutil.ReadFile("testdata/defined-job-experiment.json")
			So(err, ShouldBeNil)
			mockResponse("/api/job/11423cdd520000", string(c))
			j, err := client.GetJob(ctx, &pinpoint.GetJobRequest{
				Name: "jobs/legacy-11423cdd520000",
			})

			Convey("Then we find details in the response proto", func() {
				So(err, ShouldBeNil)
				So(j.Name, ShouldEqual, "jobs/legacy-11423cdd520000")
			})
		})

		Convey("When we attempt to get an undefined job", func() {
			_, err := client.GetJob(ctx, &pinpoint.GetJobRequest{
				Name: "jobs/legacy-02",
			})
			Convey("Then we get an error in the gRPC request", func() {
				So(err, ShouldNotBeNil)
				So(grpc.Code(err), ShouldEqual, codes.NotFound)
			})

		})

		Convey("When we attempt to provide an ill-defined legacy id", func() {
			_, err := client.GetJob(ctx, &pinpoint.GetJobRequest{
				Name: "jobs/legacy-",
			})
			Convey("Then we get an error in the gRPC request", func() {
				So(err, ShouldNotBeNil)
				So(grpc.Code(err), ShouldEqual, codes.InvalidArgument)
			})
		})

		Convey("When we attempt go get an experiment job with results", func() {
			c, err := ioutil.ReadFile("testdata/defined-job-experiment.json")
			So(err, ShouldBeNil)
			mockResponse("/api/job/11423cdd520000", string(c))
			j, err := client.GetJob(ctx, &pinpoint.GetJobRequest{
				Name: "jobs/legacy-11423cdd520000",
			})
			Convey("Then we find the results in the response", func() {
				So(err, ShouldBeNil)
				exp := j.JobSpec.GetExperiment()
				So(exp, ShouldNotBeNil)
				So(exp.BaseCommit.GitHash, ShouldEqual, "0d8952cfc50b039bf50320c9d3db82b164f3e549")
				So(exp.ExperimentPatch.Change, ShouldEqual, 2560197)
				So(exp.ExperimentPatch.Patchset, ShouldEqual, 12)
			})
		})
	})
}

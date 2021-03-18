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
	"encoding/base64"
	"fmt"
	"infra/chromeperf/pinpoint"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	. "github.com/smartystreets/goconvey/convey"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func registerPinpointServer(t *testing.T, srv *pinpointServer) func(context.Context, string) (net.Conn, error) {
	t.Helper()

	l := bufconn.Listen(bufSize)
	t.Cleanup(func() { l.Close() })
	s := grpc.NewServer()
	t.Cleanup(func() { s.Stop() })
	dialer := func(context.Context, string) (net.Conn, error) {
		return l.Dial()
	}
	pinpoint.RegisterPinpointServer(s, srv)
	go func() {
		if err := s.Serve(l); err != nil {
			log.Fatalf("Server startup failed.")
		}
	}()
	return dialer
}

type requestRecorder struct {
	*httptest.Server

	// List of URLs of received requests during recording.
	urls      []*url.URL
	recording bool
}

// startRecord begins the storage of URLs executed on the test server. It
// returns a function which can be used to retrieve results. Note that the
// returned function must be called in order to clean up.
func (rr *requestRecorder) startRecord() (done func() []*url.URL) {
	if rr.recording {
		panic(fmt.Sprintf("Invalid state: startRecorder called before prior recording was finished"))
	}
	rr.recording = true
	rr.urls = nil
	return func() []*url.URL {
		rr.recording = false
		return rr.urls
	}
}

func startFakeLegacyServer(t *testing.T, httpResponses map[string]string) *requestRecorder {
	ret := new(requestRecorder)
	ret.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ret.recording {
			ret.urls = append(ret.urls, r.URL)
		}

		resp, found := httpResponses[r.URL.Path]
		if !found {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, "Not found")
			return
		}
		fmt.Fprintln(w, resp)
	}))
	t.Cleanup(ret.Server.Close)
	return ret
}

func TestServerService(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	Convey("Given a grpc server without a client", t, func() {
		dialer := registerPinpointServer(t, &pinpointServer{})

		Convey("When we connect to the Pinpoint service", func() {
			conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
			So(err, ShouldBeNil)
			t.Cleanup(func() { conn.Close() })
			client := pinpoint.NewPinpointClient(conn)

			Convey("Then requests to ScheduleJob will fail with 'misconfigured service'", func() {
				_, err := client.ScheduleJob(ctx, &pinpoint.ScheduleJobRequest{})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "misconfigured service")
			})

		})
	})

	Convey("Given a grpc server with a legacy client not behind the ESP", t, func() {
		ts := startFakeLegacyServer(t, nil)
		log.Printf("legacy service = %s", ts.URL)
		dialer := registerPinpointServer(t, &pinpointServer{legacyPinpointService: ts.URL, LegacyClient: &http.Client{}})

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
	definedJobExperimentJSON, err := ioutil.ReadFile("testdata/defined-job-experiment.json")
	if err != nil {
		t.Fatal(err)
	}

	const (
		definedJobID   = "11423cdd520000"
		definedJobName = "jobs/legacy-" + definedJobID
	)
	ts := startFakeLegacyServer(t, map[string]string{
		"/api/job/" + definedJobID: string(definedJobExperimentJSON),
	})
	defer ts.Close()
	log.Printf("legacy service = %s", ts.URL)

	ctx := context.Background()
	Convey("Given a grpc server with a client", t, func() {
		dialer := registerPinpointServer(t, &pinpointServer{legacyPinpointService: ts.URL, LegacyClient: &http.Client{}})

		conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
		So(err, ShouldBeNil)
		defer conn.Close()
		client := pinpoint.NewPinpointClient(conn)

		Convey("When we attempt to get a defined job", func() {
			j, err := client.GetJob(ctx, &pinpoint.GetJobRequest{
				Name: definedJobName,
			})

			Convey("Then we find details in the response proto", func() {
				So(err, ShouldBeNil)
				So(j.Name, ShouldEqual, definedJobName)
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
			j, err := client.GetJob(ctx, &pinpoint.GetJobRequest{
				Name: definedJobName,
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

func TestCancelJob(t *testing.T) {
	t.Parallel()
	definedJobExperimentJSON, err := ioutil.ReadFile("testdata/defined-job-experiment.json")
	if err != nil {
		t.Fatal(err)
	}

	const (
		definedJobID   = "11423cdd520000"
		definedJobName = "jobs/legacy-" + definedJobID
	)
	ts := startFakeLegacyServer(t, map[string]string{
		"/api/job/" + definedJobID: string(definedJobExperimentJSON),
		"/api/job/cancel":          "",
	})
	defer ts.Close()
	log.Printf("legacy service = %s", ts.URL)

	ctx := context.Background()
	authorizedCtx := metadata.NewOutgoingContext(ctx, metadata.MD{
		endpointsHeader: []string{
			base64.RawURLEncoding.EncodeToString([]byte(`{"email": "anonymous-user@example.com"}`)),
		},
	})
	Convey("Given a grpc server with a client", t, func() {
		dialer := registerPinpointServer(t, &pinpointServer{legacyPinpointService: ts.URL, LegacyClient: &http.Client{}})

		conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
		So(err, ShouldBeNil)
		defer conn.Close()
		client := pinpoint.NewPinpointClient(conn)

		Convey("with an un-authenticated connection", func() {
			Convey("We fail to cancel the job", func() {
				_, err := client.CancelJob(ctx, &pinpoint.CancelJobRequest{Name: definedJobName, Reason: "because"})
				So(err, ShouldBeError)
			})
		})

		Convey("with an authenticated connection", func() {
			Convey("with an authorized connection", func() {
				ctx := authorizedCtx
				Convey("We fail to cancel the Job without a reason", func() {
					_, err := client.CancelJob(ctx, &pinpoint.CancelJobRequest{Name: definedJobName})
					So(err, ShouldBeError)
				})
				Convey("We fail to cancel the Job without a name", func() {
					_, err := client.CancelJob(ctx, &pinpoint.CancelJobRequest{Reason: "because"})
					So(err, ShouldBeError)
				})
				Convey("We can cancel a job with name and reason", func() {
					j, err := client.CancelJob(ctx, &pinpoint.CancelJobRequest{Name: definedJobName, Reason: "because"})
					So(err, ShouldBeNil)
					So(j.Name, ShouldEqual, definedJobName)
				})
				Convey("We fail to cancel a missing job", func() {
					_, err := client.CancelJob(ctx, &pinpoint.CancelJobRequest{Name: "doesnt-exist", Reason: "because"})
					So(err, ShouldBeError)
				})
			})
		})
	})
}

func TestListJob(t *testing.T) {
	t.Parallel()
	listResponseJSON, err := ioutil.ReadFile("testdata/example-list-response.json")
	if err != nil {
		t.Fatal(err)
	}

	ts := startFakeLegacyServer(t, map[string]string{
		"/api/jobs": string(listResponseJSON),
	})
	defer ts.Close()

	ctx := context.Background()
	Convey("Given a grpc server with a client", t, func() {
		dialer := registerPinpointServer(t, &pinpointServer{legacyPinpointService: ts.URL, LegacyClient: &http.Client{}})

		conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
		So(err, ShouldBeNil)
		defer conn.Close()
		client := pinpoint.NewPinpointClient(conn)

		Convey("listing results is successful", func() {
			_, err := client.ListJobs(ctx, &pinpoint.ListJobsRequest{})
			So(err, ShouldBeNil)
		})
		Convey("filters in the RPC request make it to the legacy service", func() {
			const filter = "THE FILTER"

			getURLs := ts.startRecord()
			_, err := client.ListJobs(ctx, &pinpoint.ListJobsRequest{Filter: filter})
			urls := getURLs()
			So(err, ShouldBeNil)
			So(len(urls), ShouldEqual, 1)
			So(urls[0].String(), ShouldContainSubstring, url.QueryEscape(filter))
		})
	})
}

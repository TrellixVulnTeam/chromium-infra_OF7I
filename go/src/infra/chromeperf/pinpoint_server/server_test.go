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
	"strings"
	"testing"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func TestMisconfiguredServer(t *testing.T) {
	ctx := context.Background()
	l := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pinpoint.RegisterPinpointServer(s, &pinpointServer{})
	go func() {
		if err := s.Serve(l); err != nil {
			log.Fatalf("Server startup failed.")
		}
	}()
	dialer := func(context.Context, string) (net.Conn, error) {
		return l.Dial()
	}
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	defer conn.Close()
	client := pinpoint.NewPinpointClient(conn)
	resp, err := client.ScheduleJob(ctx, &pinpoint.ScheduleJobRequest{})
	if err == nil {
		t.Fatalf("Expected err is not nil, got: nil")
	}
	if !strings.Contains(err.Error(), "misconfigured service") {
		t.Fatalf("Missing substring 'misconfigured service' in error; got: %v", err)
	}
	log.Printf("Response: %v", resp)
}

func TestServerNotBehindESP(t *testing.T) {
	ctx := context.Background()
	l := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "{}")
	}))
	defer ts.Close()

	client, err := google.DefaultClient(oauth2.NoContext, scopesForLegacy...)
	flag.Set("legacy_pinpoint_service", ts.URL)
	if err != nil {
		t.Fatalf("Failed getting credentials: %v", err)
	}
	pinpoint.RegisterPinpointServer(s, &pinpointServer{LegacyClient: client})
	go func() {
		if err := s.Serve(l); err != nil {
			log.Fatalf("Server startup failed.")
		}
	}()
	dialer := func(context.Context, string) (net.Conn, error) {
		return l.Dial()
	}
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	defer conn.Close()
	gclient := pinpoint.NewPinpointClient(conn)
	resp, err := gclient.ScheduleJob(ctx, &pinpoint.ScheduleJobRequest{})
	if err == nil {
		t.Fatalf("Expected err is not nil, got: nil")
	}
	if !strings.Contains(err.Error(), "missing required auth header") {
		t.Fatalf("Missing substring 'missing required auth header' in error; got: %v", err)
	}
	log.Printf("Response: %v", resp)
}

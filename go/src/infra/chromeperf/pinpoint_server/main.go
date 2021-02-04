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
	"flag"
	"log"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"infra/chromeperf/pinpoint"
)

const (
	port = ":60800"
)

// Scopes to use for OAuth2.0 credentials.
var scopesForLegacy = []string{
	// Provide access to the email address of the user.
	"https://www.googleapis.com/auth/userinfo.email",
}

func main() {
	// TODO(crbug/1059667): Wire up a cloud logging implementation (Stackdriver).
	flag.Parse()
	if _, err := url.Parse(*legacyPinpointService); err != nil {
		log.Fatalf(
			"Invalid URL for -legacy_pinpoint_service: %s",
			*legacyPinpointService)
	}
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	h := health.NewServer()

	// Set up a client to be used by the Pinpoint server with OAuth credentials for the service account.
	var client *http.Client

	// Check if we've been provided explicit credentials.
	if serviceAccountEmail != nil && *serviceAccountEmail != "" {
		conf := &jwt.Config{
			Email:      *serviceAccountEmail,
			PrivateKey: []byte(*privateKey),
			TokenURL:   google.JWTTokenURL,
		}
		client = conf.Client(oauth2.NoContext)
	} else {
		client, err = google.DefaultClient(oauth2.NoContext, scopesForLegacy...)
		if err != nil {
			log.Fatalf("Failed to get default credentials: %v", err)
		}
	}

	pinpoint.RegisterPinpointServer(s, &pinpointServer{LegacyClient: client})
	h.SetServingStatus("pinpoint", grpc_health_v1.HealthCheckResponse_SERVING)
	h.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(s, h)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

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
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"infra/chromeperf/pinpoint"
	"infra/chromeperf/pinpoint_server/conversion"
)

type pinpointServer struct {
	pinpoint.UnimplementedPinpointServer
	// Provide an HTTP Client to be used by the server, to the Pinpoint Legacy API.
	LegacyClient *http.Client
}

const (
	port                  = ":60800"
	legacyPinpointService = "https://pinpoint-dot-chromeperf.appspot.com"
)

// Scopes to use for OAuth2.0 credentials.
var scopesForLegacy = []string{
	// Provide access to the email address of the user.
	"https://www.googleapis.com/auth/userinfo.email",
}

func getRequestingUserEmail(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.InvalidArgument, "missing metadata from request context")
	}
	var userInfo struct {
		Email string
	}
	auth, ok := md["x-endpoints-api-userinfo"]
	if !ok {
		return "", status.Error(codes.PermissionDenied, "missing required auth header 'x-endpoints-api-userinfo'")
	}
	// Decode the auto header from base64encoded json, into a map we can inspect.
	decoded, err := base64.URLEncoding.DecodeString(auth[0])
	if err != nil {
		return "", status.Errorf(codes.InvalidArgument, "malformed x-endpoints-api-userinfo: %v", err)
	}
	if json.Unmarshal(decoded, &userInfo) != nil {
		return "", status.Errorf(codes.InvalidArgument, "malformed x-endpoints-api-userinfo: %v", err)
	}
	return userInfo.Email, nil
}

func (s *pinpointServer) ScheduleJob(ctx context.Context, r *pinpoint.ScheduleJobRequest) (*pinpoint.Job, error) {
	// First, ensure we can set the user from the incoming request, based on their identity provided in the OAuth2
	// headers, that make it into the context of this request. Because we intend this service to be hosted behind an
	// Endpoint Service Proxy (ESP), we're going to look for the authentication details in the
	// X-Endpoints-API-Userinfo header, as part of the context. We'll fail if we aren't being served behind an ESP.
	//
	// See
	// https://cloud.google.com/endpoints/docs/grpc/authenticating-users#receiving_authentication_results_in_your_api
	// for details on the format and specifications for the contents of this header.
	if s.LegacyClient == nil {
		return nil, status.Error(codes.Internal, "misconfigured service, please try again later")
	}
	userEmail, err := getRequestingUserEmail(ctx)
	if err != nil {
		return nil, err
	}

	// Before we make this service the source of truth for the Pinpoint service, we first proxy requests to the
	// actual Pinpoint legacy API from the provided request.
	values, err := conversion.ConvertToValues(r.Job, userEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("%v", err))
	}

	// Then we make the request to the Pinpoint service.
	res, err := s.LegacyClient.PostForm(legacyPinpointService+"/api/new", values)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed request to legqacy API: %v", err)
	}

	// The response of the legacy service has the following format:
	//
	// {
	//    'jobId': <string>,
	//    'jobUrl': <string>
	// }
	//
	newBody, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed reading response from legacy API: %v", err)
	}

	var newResponse struct {
		JobID string
	}
	if err := json.Unmarshal(newBody, &newResponse); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse response from legacy API: %v", err)
	}

	// Return with a minimal Job response.
	// TODO(dberris): Write this data out to Spanner when we're ready to replace the legacy API.
	return &pinpoint.Job{
		Id:        newResponse.JobID,
		CreatedBy: userEmail,
		JobSpec:   r.Job,
	}, nil
}

func (s *pinpointServer) GetJob(ctx context.Context, r *pinpoint.GetJobRequest) (*pinpoint.Job, error) {
	// TODO(dberris): Implement this!
	return nil, nil
}

func (s *pinpointServer) ListJobs(ctx context.Context, r *pinpoint.ListJobsRequest) (*pinpoint.ListJobsResponse, error) {
	// TODO(dberris): Implement this!
	return nil, nil
}

func (s *pinpointServer) CancelJob(ctx context.Context, r *pinpoint.CancelJobRequest) (*pinpoint.Job, error) {
	// TODO(dberris): Implement this!
	return nil, nil
}

// Email address for the service account to use.
var serviceAccountEmail = flag.String("service_account", "", "service account email")

// Contents of the service account credentials PEM file.
var privateKey = flag.String("private_key", "", "service account PEM file contents")

func main() {
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

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
	"encoding/json"
	"flag"
	"fmt"
	"infra/chromeperf/pinpoint"
	"infra/chromeperf/pinpoint/server/convert"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type pinpointServer struct {
	pinpoint.UnimplementedPinpointServer

	// URL for requests to the legacy service
	legacyPinpointService string

	// Provide an HTTP Client to be used by the server, to the Pinpoint Legacy API.
	LegacyClient *http.Client
}

const (
	port            = ":60800"
	endpointsHeader = "x-endpoint-api-userinfo"
)

// Scopes to use for OAuth2.0 credentials.
var (
	scopesForLegacy = []string{
		// Provide access to the email address of the user.
		"https://www.googleapis.com/auth/userinfo.email",
	}
	jobNameRe = regexp.MustCompile(`^jobs/legacy-(?P<id>[a-f0-9]+)$`)
)

func getRequestingUserEmail(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.InvalidArgument, "missing metadata from request context")
	}
	auth, ok := md[endpointsHeader]
	if !ok || len(auth) == 0 {
		return "", status.Errorf(codes.PermissionDenied, "missing required auth header '%s'", endpointsHeader)
	}
	// Decode the auto header from base64encoded json, into a map we can inspect.
	decoded, err := base64.RawURLEncoding.DecodeString(auth[0])
	if err != nil {
		grpclog.Errorf("Failed decoding auth = '%v'; error = %s", auth, err)
		return "", status.Errorf(codes.InvalidArgument, "malformed %s: %v", endpointsHeader, err)
	}
	userInfo := make(map[string]interface{})
	if json.Unmarshal(decoded, &userInfo) != nil {
		return "", status.Errorf(codes.InvalidArgument, "malformed %s: %v", endpointsHeader, err)
	}
	email, ok := userInfo["email"].(string)
	if !ok || len(email) == 0 {
		return "", status.Errorf(codes.PermissionDenied, "missing 'email' field from token")
	}
	return email, nil
}

func (s *pinpointServer) ScheduleJob(ctx context.Context, r *pinpoint.ScheduleJobRequest) (*pinpoint.Job, error) {
	// First, ensure we can set the user from the incoming request, based on their identity provided in the OAuth2
	// headers, that make it into the context of this request. Because we intend this service to be hosted behind an
	// Endpoint Service Proxy (ESP), we're going to look for the authentication details in the
	// X-Endpoint-API-Userinfo header, as part of the context. We'll fail if we aren't being served behind an ESP.
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
	values, err := convert.JobToValues(r.Job, userEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("%v", err))
	}

	// Then we make the request to the Pinpoint service.
	res, err := s.LegacyClient.PostForm(s.legacyPinpointService+"/api/new", values)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed request to legacy API: %v", err)
	}
	switch res.StatusCode {
	case http.StatusUnauthorized:
		return nil, status.Errorf(codes.Internal, "Internal error, service not authorized")
	case http.StatusBadRequest:
		return nil, status.Errorf(codes.Internal, "Internal error, service sent an invalid request")
	case http.StatusOK:
		break
	default:
		return nil, status.Errorf(codes.Internal, "Internal error")
	}

	// The response of the legacy service has the following format:
	//
	// {
	//    'jobId': <string>,
	//    'jobUrl': <string>
	// }
	//
	// We ignore the 'jobUrl' field for now.
	var newResponse struct {
		JobID string
	}
	if err := json.NewDecoder(res.Body).Decode(&newResponse); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse response from legacy API: %v", err)
	}

	// Return with a minimal Job response.
	// TODO(dberris): Write this data out to Spanner when we're ready to replace the legacy API.
	return &pinpoint.Job{
		Name:      fmt.Sprintf("jobs/legacy-%s", newResponse.JobID),
		CreatedBy: userEmail,
		JobSpec:   r.Job,
	}, nil
}

func (s *pinpointServer) GetJob(ctx context.Context, r *pinpoint.GetJobRequest) (*pinpoint.Job, error) {
	// This API does not require that the user be signed in, so we'll not need to check the credentials.
	// TODO(dberris): In the future, support ACL-limiting Pinpoint job results.
	if s.LegacyClient == nil {
		return nil, status.Error(
			codes.Internal,
			"misconfigured service, please try again later")
	}

	// Make a request to the legacy API.
	// Ensure that r.Id is a hex number.
	if !jobNameRe.MatchString(r.Name) {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid id format, must match %s", jobNameRe.String())
	}
	matches := jobNameRe.FindStringSubmatch(r.Name)
	legacyID := string(matches[jobNameRe.SubexpIndex("id")])
	if len(legacyID) == 0 {
		return nil, status.Error(codes.Unimplemented, "future ids not supported yet")
	}

	u := fmt.Sprintf("%s/api/job/%s?o=STATE", s.legacyPinpointService, legacyID)
	if _, err := url.Parse(u); err != nil {
		grpclog.Errorf("Invalid URL: %s", err)
		return nil, status.Errorf(codes.Internal, "failed to form a valid legacy request")
	}
	grpclog.Infof("GET %s", u)
	res, err := s.LegacyClient.Get(u)
	if err != nil {
		grpclog.Errorf("HTTP Request Error: %s", err)
		return nil, status.Errorf(codes.Internal, "failed retrieving job data from legacy service")
	}
	switch res.StatusCode {
	case http.StatusNotFound:
		return nil, status.Errorf(codes.NotFound, "job not found")
	case http.StatusOK:
		break
	default:
		return nil, status.Errorf(codes.Internal, "failed request: %s", res.Status)
	}

	defer res.Body.Close()
	job, err := convert.JobToProto(res.Body)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return job, nil
}

func (s *pinpointServer) ListJobs(ctx context.Context, r *pinpoint.ListJobsRequest) (*pinpoint.ListJobsResponse, error) {
	// TODO(dberris): Implement this!
	return nil, status.Error(codes.Unimplemented, "TODO")
}

func (s *pinpointServer) CancelJob(ctx context.Context, r *pinpoint.CancelJobRequest) (*pinpoint.Job, error) {
	// TODO(dberris): Implement this!
	return nil, status.Error(codes.Unimplemented, "TODO")
}

// Email address for the service account to use.
var serviceAccountEmail = flag.String("service_account", "", "service account email")

// Contents of the service account credentials PEM file.
var privateKey = flag.String("private_key", "", "service account PEM file contents")

// Flag to configure the legacy Pinpoint service URL base.
var legacyPinpointServiceFlag = flag.String(
	"legacy_pinpoint_service",
	"https://pinpoint-dot-chromeperf.appspot.com",
	"base URL for the legacy Pinpoint service",
)

// Main is the actual main body function.
func Main() {
	// TODO(crbug/1059667): Wire up a cloud logging implementation (Stackdriver).
	flag.Parse()
	if _, err := url.Parse(*legacyPinpointServiceFlag); err != nil {
		log.Fatalf(
			"Invalid URL for -legacy_pinpoint_service: %s",
			*legacyPinpointServiceFlag)
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

	pinpoint.RegisterPinpointServer(s, &pinpointServer{legacyPinpointService: *legacyPinpointServiceFlag, LegacyClient: client})
	h.SetServingStatus("pinpoint", grpc_health_v1.HealthCheckResponse_SERVING)
	h.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(s, h)
	log.Println("Listening on ", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

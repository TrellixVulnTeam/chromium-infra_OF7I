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
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"

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

	// If set, this will be returned by getRequestingUserEmail for all requests.
	hardcodedUserEmail string
}

const (
	// EndpointsHeader is the metadata header used to obtain user information.
	// https://cloud.google.com/endpoints/docs/openapi/authenticating-users-custom#receiving_authenticated_results_in_your_api
	EndpointsHeader = "x-endpoint-api-userinfo"
)

// Scopes to use for OAuth2.0 credentials.
var (
	scopesForLegacy = []string{
		// Provide access to the email address of the user.
		"https://www.googleapis.com/auth/userinfo.email",
	}
)

func (s *pinpointServer) getRequestingUserEmail(ctx context.Context) (string, error) {
	if email := s.hardcodedUserEmail; email != "" {
		return email, nil
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.InvalidArgument, "missing metadata from request context")
	}
	auth, ok := md[EndpointsHeader]
	if !ok || len(auth) == 0 {
		return "", status.Errorf(codes.PermissionDenied, "missing required auth header '%s'", EndpointsHeader)
	}
	// Decode the auto header from base64encoded json, into a map we can inspect.
	decoded, err := base64.RawURLEncoding.DecodeString(auth[0])
	if err != nil {
		grpclog.Errorf("Failed decoding auth = '%v'; error = %s", auth, err)
		return "", status.Errorf(codes.InvalidArgument, "malformed %s: %v", EndpointsHeader, err)
	}
	userInfo := make(map[string]interface{})
	if json.Unmarshal(decoded, &userInfo) != nil {
		return "", status.Errorf(codes.InvalidArgument, "malformed %s: %v", EndpointsHeader, err)
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
	userEmail, err := s.getRequestingUserEmail(ctx)
	if err != nil {
		return nil, err
	}

	if r.Job == nil {
		return nil, status.Error(codes.InvalidArgument, "must set Job in request")
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
		grpclog.Errorf("Got unexpected Status from /api/new: %v", res.Status)
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

	legacyID, err := pinpoint.LegacyJobID(r.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "bad name %q: %v", r.Name, err)
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
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNotFound:
		return nil, status.Errorf(codes.NotFound, "job not found")
	case http.StatusOK:
		break
	default:
		bs, _ := ioutil.ReadAll(res.Body)
		grpclog.Errorf("HTTP status %s: %s", res.Status, bs)
		return nil, status.Errorf(codes.Internal, "failed request to legacy service: %s", res.Status)
	}

	job, err := convert.JobToProto(res.Body)
	if err != nil {
		grpclog.Errorf("failed to convert results for GetJob: %s", err)

		// Only return an error to the user if no info is available at all.
		if job == nil {
			return nil, status.Error(codes.Internal, "failed to retrieve job from legacy service")
		}
	}
	return job, nil
}

func (s *pinpointServer) ListJobs(ctx context.Context, r *pinpoint.ListJobsRequest) (*pinpoint.ListJobsResponse, error) {
	if r.PageSize != 0 || r.PageToken != "" {
		// TODO(chowski): Implement this!
		return nil, status.Error(codes.Unimplemented, "TODO: implement pagination/page size")
	}

	if s.LegacyClient == nil {
		return nil, status.Error(
			codes.Internal,
			"misconfigured service, please try again later")
	}

	query := url.Values{
		"o":      {"INPUTS"},
		"filter": {r.Filter},
	}.Encode()
	u := fmt.Sprintf("%s/api/jobs?%s", s.legacyPinpointService, query)
	grpclog.Infof("GET %s", u)
	res, err := s.LegacyClient.Get(u)
	if err != nil {
		grpclog.Errorf("HTTP Request Error: %s", err)
		return nil, status.Errorf(codes.Internal, "failed retrieving job data from legacy service")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.Internal, "failed request to legacy service: %s", res.Status)
	}

	jobs, err := convert.JobListToProto(res.Body)
	if err != nil {
		grpclog.Errorf("failed to convert results for ListJobs: %s", err)

		// Only return an error back to the user if there were no successfully
		// parsed jobs.
		if len(jobs) == 0 {
			return nil, status.Errorf(codes.Internal, "failed to list results from legacy service")
		}
	}
	return &pinpoint.ListJobsResponse{
		Jobs: jobs,
	}, nil
}

func (s *pinpointServer) CancelJob(ctx context.Context, r *pinpoint.CancelJobRequest) (*pinpoint.Job, error) {
	userEmail, err := s.getRequestingUserEmail(ctx)
	if err != nil {
		return nil, err
	}

	if r.Reason == "" || r.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "must set the 'reason' and 'name' fields")
	}
	legacyID, err := pinpoint.LegacyJobID(r.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	query := url.Values{
		"job_id": {legacyID},
		"reason": {r.Reason},
		"user":   {userEmail},
	}.Encode()
	u := fmt.Sprintf("%s/api/job/cancel?%s", s.legacyPinpointService, query)
	if _, err := url.Parse(u); err != nil {
		grpclog.Errorf("Invalid URL: %s", err)
		return nil, status.Errorf(codes.Internal, "failed to form a valid legacy request")
	}
	grpclog.Infof("POST %s", u)
	res, err := s.LegacyClient.Post(u, "", nil)
	if err != nil {
		grpclog.Errorf("HTTP Request Error: %s", err)
		return nil, status.Errorf(codes.Internal, "failed retrieving job data from legacy service")
	}
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.Internal, "failed request: %s", res.Status)
	}
	return s.GetJob(ctx, &pinpoint.GetJobRequest{Name: r.Name})
}

// New returns a pinpointServer configured to contact the legacy API with the
// provided base URL and HTTP client.
func New(legacyPinpointBaseURL string, client *http.Client) *pinpointServer {
	return &pinpointServer{
		legacyPinpointService: legacyPinpointBaseURL,
		LegacyClient:          client,
	}
}

// Main is the actual main body function. It registers flags, parses them, and
// kicks off a gRPC service to host
func Main() {
	legacyPinpointServiceFlag := flag.String(
		"legacy_pinpoint_service",
		"https://pinpoint-dot-chromeperf.appspot.com",
		"base URL for the legacy Pinpoint service",
	)
	serviceAccountEmail := flag.String("service_account", "", "If specified, this email address is used as the identity of the service")
	privateKey := flag.String("private_key", "", "Required if -service_account is set; contents of the service account credentials PEM file.")
	hardcodedUserEmailFlag := flag.String("hardcoded_user_email", "", "TESTING ONLY; if set, request auth headers will be ignored and this email address will be used for all incoming requests. It also causes all outgoing RPCs to not use any authentication.")
	port := flag.Int("port", 60800, "Tcp port that the service should listen.")

	// TODO(crbug/1059667): Wire up a cloud logging implementation (Stackdriver).
	flag.Parse()
	if _, err := url.Parse(*legacyPinpointServiceFlag); err != nil {
		log.Fatalf(
			"Invalid URL for -legacy_pinpoint_service: %s",
			*legacyPinpointServiceFlag)
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
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
	} else if *hardcodedUserEmailFlag != "" {
		client = http.DefaultClient
	} else {
		client, err = google.DefaultClient(oauth2.NoContext, scopesForLegacy...)
		if err != nil {
			log.Fatalf("Failed to get default credentials: %v", err)
		}
	}

	server := New(*legacyPinpointServiceFlag, client)
	server.hardcodedUserEmail = *hardcodedUserEmailFlag
	pinpoint.RegisterPinpointServer(s, server)
	h.SetServingStatus("pinpoint", grpc_health_v1.HealthCheckResponse_SERVING)
	h.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(s, h)
	log.Println("Listening on ", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

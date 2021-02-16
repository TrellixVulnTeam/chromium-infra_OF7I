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
	"infra/chromeperf/pinpoint/server/conversion"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	"google.golang.org/protobuf/types/known/timestamppb"
)

type pinpointServer struct {
	pinpoint.UnimplementedPinpointServer

	// Provide an HTTP Client to be used by the server, to the Pinpoint Legacy API.
	LegacyClient *http.Client
}

// MicroTime is an alias to time.Time which allows us to parse microsecond-precision time.
type MicroTime time.Time

// UnmarshalJSON supports parsing nanosecond timestamps.
func (t *MicroTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `\"`)
	p, err := time.Parse("2006-01-02T15:04:05.999999", s)
	if err != nil {
		return err
	}
	*t = MicroTime(p)
	return nil
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
	gerritRe  = regexp.MustCompile(
		`^/c/(?P<project>[^/]+)/(?P<repo>[^+]+)/\+/(?P<cl>[1-9]\d*)(/(?P<patchset>[1-9]\d*))?$`)
	gerritProjectIdx  = gerritRe.SubexpIndex("project")
	gerritRepoIdx     = gerritRe.SubexpIndex("repo")
	gerritClIdx       = gerritRe.SubexpIndex("cl")
	gerritPatchSetIdx = gerritRe.SubexpIndex("patchset")
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
	values, err := conversion.ConvertToValues(r.Job, userEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("%v", err))
	}

	// Then we make the request to the Pinpoint service.
	res, err := s.LegacyClient.PostForm(*legacyPinpointService+"/api/new", values)
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

type gerritParts struct {
	project, repo string
	cl, patchSet  int64
}

func parseGerritURL(s string) (*gerritParts, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid patch url gotten from legacy service")
	}
	m := gerritRe.FindStringSubmatch(u.Path)
	if m == nil {
		return nil, status.Errorf(codes.Internal, "invalid CL path in URL gotten from legacy service: %s", u.Path)
	}
	project := m[gerritProjectIdx]
	clStr := m[gerritClIdx]
	patchSetStr := m[gerritPatchSetIdx]
	repo := m[gerritRepoIdx]

	cl, err := strconv.ParseInt(clStr, 10, 64)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid CL number in URL gotten from legacy service")
	}
	patchSet, err := strconv.ParseInt(patchSetStr, 10, 32)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid patchset number in URL gotten from legacy service")
	}
	return &gerritParts{
		project: project, repo: repo, cl: cl, patchSet: patchSet,
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

	u := fmt.Sprintf("%s/api/job/%s?o=STATE", *legacyPinpointService, legacyID)
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

	var l struct {
		Arguments           map[string]string       `json:"arguments"`
		BugID               int64                   `json:"bug_id"`
		ComparisonMode      string                  `json:"comparison_mode,omitempty"`
		ComparisonMagnitude float64                 `json:"comparison_magnitude,omitempty"`
		Cfg                 string                  `json:"configuration,omitempty"`
		Created             MicroTime               `json:"created,omitempty"`
		Exception           *map[string]interface{} `json:"exception,omitempty"`
		JobID               string                  `json:"job_id,omitempty"`
		Metric              string                  `json:"metric,omitempty"`
		Name                string                  `json:"name,omitempty"`
		Project             *string                 `json:"project,omitempty"`
		StepLabels          []string                `json:"quests,omitempty"`
		ResultsURL          string                  `json:"results_url,omitempty"`
		StartedTime         MicroTime               `json:"started_time,omitempty"`
		State               []struct {
			Attempts []struct {
				Executions []struct {
					Completed bool `json:"completed"`
					Details   []struct {
						Value string `json:"value,omitempty"`
						Key   string `json:"key,omitempty"`
						URL   string `json:"url,omitempty"`
					} `json:"details"`
				} `json:"executions"`
			} `json:"attempts"`
			Change struct {
				Commits []struct {
					Author         string    `json:"author,omitempty"`
					ChangeID       string    `json:"change_id,omitempty"`
					CommitPosition int64     `json:"commit_position,omitempty"`
					Created        MicroTime `json:"created,omitempty"`
					GitHash        string    `json:"git_hash,omitempty"`
					Message        string    `json:"message,omitempty"`
					Repo           string    `json:"repository,omitempty"`
					ReviewURL      string    `json:"review_url,omitempty"`
					Subject        string    `json:"subject,omitempty"`
					URL            string    `json:"url,omitempty"`
				} `json:"commits"`
				Patch struct {
					Created  MicroTime `json:"created,omitempty"`
					URL      string    `json:"url,omitempty"`
					Author   string    `json:"author,omitempty"`
					Server   string    `json:"server,omitempty"`
					Message  string    `json:"message,omitempty"`
					Subject  string    `json:"subject,omitempty"`
					ChangeID string    `json:"change,omitempty"`
					Revision string    `json:"revision,omitempty"`
				}
			} `json:"change"`
			Comparisons struct {
				Prev string `json:"prev,omitempty"`
				Next string `json:"next,omitempty"`
			} `json:"comparisons"`
		} `json:"state,omitempty"`
		Status  string    `json:"status,omitempty"`
		Updated MicroTime `json:"updated,omitempty"`
		User    string    `json:"user,omitempty"`
	}
	if err := json.NewDecoder(res.Body).Decode(&l); err != nil {
		grpclog.Errorf("failed parsing json: %q", err)
		return nil, status.Errorf(
			codes.Internal,
			"received ill-formed response from legacy service",
		)
	}

	transformState := func() pinpoint.Job_State {
		switch l.Status {
		case "Running":
			return pinpoint.Job_RUNNING
		case "Queued":
			return pinpoint.Job_PENDING
		case "Cancelled":
			return pinpoint.Job_CANCELLED
		case "Failed":
			return pinpoint.Job_FAILED
		case "Completed":
			return pinpoint.Job_SUCCEEDED
		}
		return pinpoint.Job_STATE_UNSPECIFIED
	}

	transformMode := func() pinpoint.JobSpec_ComparisonMode {
		switch l.ComparisonMode {
		case "functional":
			return pinpoint.JobSpec_FUNCTIONAL
		case "try", "performance":
			return pinpoint.JobSpec_PERFORMANCE
		}
		return pinpoint.JobSpec_COMPARISON_MODE_UNSPECIFIED
	}

	// Now attempt to translate the parsed JSON structure into a protobuf.
	// FIXME(dberris): Interpret the results better, differentiating experiments from bisections, etc.
	cMode := transformMode()
	j := &pinpoint.Job{
		Name:           fmt.Sprintf("jobs/legacy-%s", l.JobID),
		State:          transformState(),
		CreatedBy:      l.User,
		CreateTime:     timestamppb.New(time.Time(l.Created)),
		LastUpdateTime: timestamppb.New(time.Time(l.Updated)),
		JobSpec: &pinpoint.JobSpec{
			ComparisonMode:      cMode,
			ComparisonMagnitude: l.ComparisonMagnitude,
			Config:              l.Cfg,
			Target:              l.Arguments["target"],
			MonorailIssue: func() *pinpoint.MonorailIssue {
				if l.Project == nil || l.BugID == 0 {
					return nil
				}
				return &pinpoint.MonorailIssue{
					Project: *l.Project,
					IssueId: l.BugID,
				}
			}(),
		},
	}

	// We set the oneof field after initialising the proto because the
	// comparison_mode field in the JSON response is overloaded. The
	// proto doesn't have that problem because we're differentiating
	// between a bisection job and an experiment. This code performs
	// the disambiguation when we mean "try" to be a performance experiment
	// and "performance" to be a performance bisection.
	//
	// In the proto schema, we support functional bisection and experiments
	// although that functionality is yet to be supported by Pinpoint.
	switch cMode {
	case pinpoint.JobSpec_PERFORMANCE:
		switch l.ComparisonMode {
		case "try":
			{
				// Then we've got an experiment.
				if expectedStates, foundStates := 2, len(l.State); expectedStates != foundStates {
					return nil, status.Errorf(codes.Internal, "invalid state count in legacy response, want %d got %d", expectedStates, foundStates)
				}

				// By convention we use the first state's change to be the
				// "base" while the second state is the "experiment".
				c0 := &l.State[0].Change
				c1 := &l.State[1].Change

				// FIXME: Find a better way to expose this data from the legacy
				p, err := parseGerritURL(c1.Patch.URL)
				if err != nil {
					return nil, err
				}
				// service's JSON response instead of parsing URLs.
				j.JobSpec.JobKind = &pinpoint.JobSpec_Experiment{
					Experiment: &pinpoint.Experiment{
						BaseCommit: &pinpoint.GitilesCommit{
							Host:    c0.Commits[0].URL,
							Project: c0.Commits[0].Repo,
							GitHash: c0.Commits[0].GitHash,
						},
						// FIXME: Fill out these details.
						BasePatch: &pinpoint.GerritChange{
							Host:     "",
							Project:  "",
							Change:   0,
							Patchset: 0,
						},
						ExperimentCommit: &pinpoint.GitilesCommit{
							Host:    c1.Commits[0].URL,
							Project: c1.Commits[0].Repo,
							GitHash: c1.Commits[0].GitHash,
						},
						ExperimentPatch: &pinpoint.GerritChange{
							// FIXME: We have two URLs in the result JSON, we
							// need to extract the relevant details for the
							// proto response.
							Host:     c1.Patch.Server,
							Project:  p.project,
							Change:   p.cl,
							Patchset: p.patchSet,
						},
					},
				}
			}
		case "performance":
			{
				// FIXME: When we're ready to support bisection results, fill this out.
				j.JobSpec.JobKind = &pinpoint.JobSpec_Bisection{
					Bisection: &pinpoint.Bisection{
						CommitRange: &pinpoint.GitilesCommitRange{
							Host:         "",
							Project:      "",
							StartGitHash: "",
							EndGitHash:   "",
						},
						Patch: &pinpoint.GerritChange{},
					},
				}
			}
		}
	}
	return j, nil
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

// Flag to configure the legacy Pinpoint service URL base.
var legacyPinpointService = flag.String("legacy_pinpoint_service", "https://pinpoint-dot-chromeperf.appspot.com", "base URL for the legacy Pinpoint service")

// Main is the actual main body function.
func Main() {
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

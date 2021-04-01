// Copyright 2021 The Chromium Authors.
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

// Package convert contains code to convert from the Legacy JSON API to the new
// Proto API, and vice-versa.
package convert

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"infra/chromeperf/pinpoint"

	"go.chromium.org/luci/common/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func gerritChangeToURL(c *pinpoint.GerritChange) (string, error) {
	var notFound []string
	if c.Host == "" {
		notFound = append(notFound, "host")
	}
	if c.Project == "" {
		notFound = append(notFound, "project")
	}
	if c.Change == 0 {
		notFound = append(notFound, "change")
	}
	if len(notFound) > 0 {
		return "", errors.Reason("the following fields are required but are not set: %v", notFound).Err()
	}
	// Patchset is optional, in which case we'll omit it.
	if c.Patchset == 0 {
		return fmt.Sprintf("https://%s/c/%s/+/%d", c.Host, c.Project, c.Change), nil
	}
	return fmt.Sprintf("https://%s/c/%s/+/%d/%d", c.Host, c.Project, c.Change, c.Patchset), nil
}

// JobToValues turns a pinpoint.JobSpec into a url.Values which can then be
// HTTP form-encoded for the legacy Pinpoint API.
func JobToValues(job *pinpoint.JobSpec, userEmail string) (url.Values, error) {
	v := url.Values{}
	v.Set("user", userEmail)

	// Lift the configuration into a key.
	v.Set("configuration", job.Config)

	// Always set the target into a key.
	v.Set("target", job.Target)

	// We're turning a floating point comparison magnitude to a string.
	if job.ComparisonMagnitude != 0.0 {
		v.Set("comparison_magnitude", fmt.Sprintf("%f", job.ComparisonMagnitude))
	}

	// Handle the spec for git hashes.
	switch jk := job.JobKind.(type) {
	case *pinpoint.JobSpec_Bisection:
		// Transcode the mode to the ones Pinpoint currently supports.
		switch job.GetComparisonMode() {
		case pinpoint.JobSpec_COMPARISON_MODE_UNSPECIFIED:
			fallthrough
		case pinpoint.JobSpec_PERFORMANCE:
			// The legacy API uses "performance" for performance bisections, and "try" for experiments
			v.Set("comparison_mode", "performance")
		case pinpoint.JobSpec_FUNCTIONAL:
			v.Set("comparison_mode", "functional")
		default:
			return nil, errors.Reason("Unknown comparison mode provided: %v", job.GetComparisonMode()).Err()
		}

		// We ignore the repository here, because legacy Pinpoint's API didn't support those.
		v.Set("start_git_hash", jk.Bisection.CommitRange.StartGitHash)
		v.Set("end_git_hash", jk.Bisection.CommitRange.EndGitHash)

		if jk.Bisection.Patch != nil {
			p := jk.Bisection.Patch
			patchURL, err := gerritChangeToURL(p)
			if err != nil {
				return nil, errors.Annotate(err, "invalid patch provided").Err()
			}
			v.Set("patch", patchURL)
		}
	case *pinpoint.JobSpec_Experiment:
		// Transcode the mode to the ones Pinpoint currently supports.
		switch job.GetComparisonMode() {
		case pinpoint.JobSpec_COMPARISON_MODE_UNSPECIFIED:
			fallthrough
		case pinpoint.JobSpec_PERFORMANCE:
			// The legacy API uses "performance" for performance bisections, and "try" for experiments
			v.Set("comparison_mode", "try")
		case pinpoint.JobSpec_FUNCTIONAL:
			// We're failing gracefully here in cases where we're still proxying to the legacy API.
			// In the future, we should support functional experiments too.
			return nil, errors.Reason("functional experiments not supported by legacy API").Err()
		default:
			return nil, errors.Reason("Unknown comparison mode provided: %v", job.GetComparisonMode()).Err()
		}

		// The legacy Pinpoint API doesn't support specifying the Gitiles host/project as it assumes the only
		// the Chromium codebase is being worked on.
		v.Set("base_git_hash", jk.Experiment.BaseCommit.GitHash)
		experimentPatchURL, err := gerritChangeToURL(jk.Experiment.ExperimentPatch)
		if err != nil {
			return nil, errors.Annotate(err, "invalid experiment patch").Err()
		}
		v.Set("patch", experimentPatchURL)

		if jk.Experiment.ExperimentCommit != nil {
			// Note the naming difference -- the legacy service supports "end_git_hash".
			v.Set("end_git_hash", jk.Experiment.ExperimentCommit.GitHash)
		}
		if jk.Experiment.BasePatch != nil {
			basePatchURL, err := gerritChangeToURL(jk.Experiment.BasePatch)
			if err != nil {
				return nil, errors.Annotate(err, "invalid base patch").Err()
			}
			v.Set("base_patch", basePatchURL)
		}
	}

	// Process the benchmark arguments.
	switch args := job.Arguments.(type) {
	case *pinpoint.JobSpec_TelemetryBenchmark:
		tb := args.TelemetryBenchmark
		v.Set("benchmark", tb.Benchmark)
		v.Set("metric", tb.Measurement)
		v.Set("grouping_label", tb.GroupingLabel)
		switch s := tb.StorySelection.(type) {
		case *pinpoint.TelemetryBenchmark_Story:
			v.Set("story", s.Story)
		case *pinpoint.TelemetryBenchmark_StoryTags:
			v.Set("story_tags", strings.Join(s.StoryTags.StoryTags, ","))
		default:
			return nil, errors.Reason("Unsupported story_selection in TelemetryBenchmark").
				InternalReason("story_selection is %v", s).Err()
		}
	case *pinpoint.JobSpec_GtestBenchmark:
		gb := args.GtestBenchmark
		v.Set("benchmark", gb.Benchmark)
		v.Set("trace", gb.Test)
		v.Set("chart", gb.Measurement)
	default:
		return nil, errors.Reason("unsupported arguments in JobSpec").
			InternalReason("args type is %v", args).Err()
	}
	return v, nil
}

// microTime is an alias to time.Time which allows us to parse microsecond-precision time.
type microTime time.Time

// UnmarshalJSON supports parsing nanosecond timestamps.
func (t *microTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `\"`)
	if s == "null" {
		*t = microTime{}
		return nil
	}
	p, err := time.Parse("2006-01-02T15:04:05.999999", s)
	if err != nil {
		return err
	}
	*t = microTime(p)
	return nil
}

type jsonJob struct {
	Arguments           map[string]string       `json:"arguments"`
	BugID               int64                   `json:"bug_id"`
	ComparisonMode      string                  `json:"comparison_mode,omitempty"`
	ComparisonMagnitude float64                 `json:"comparison_magnitude,omitempty"`
	Cfg                 string                  `json:"configuration,omitempty"`
	Created             microTime               `json:"created,omitempty"`
	Exception           *map[string]interface{} `json:"exception,omitempty"`
	JobID               string                  `json:"job_id,omitempty"`
	Metric              string                  `json:"metric,omitempty"`
	Name                string                  `json:"name,omitempty"`
	Project             *string                 `json:"project,omitempty"`
	StepLabels          []string                `json:"quests,omitempty"`
	ResultsURL          string                  `json:"results_url,omitempty"`
	StartedTime         microTime               `json:"started_time,omitempty"`
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
				Created        microTime `json:"created,omitempty"`
				GitHash        string    `json:"git_hash,omitempty"`
				Message        string    `json:"message,omitempty"`
				Repo           string    `json:"repository,omitempty"`
				ReviewURL      string    `json:"review_url,omitempty"`
				Subject        string    `json:"subject,omitempty"`
				URL            string    `json:"url,omitempty"`
			} `json:"commits"`
			Patch struct {
				Created  microTime `json:"created,omitempty"`
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
	Updated microTime `json:"updated,omitempty"`
	User    string    `json:"user,omitempty"`
}

// JobToProto converts a stream of JSON representing a Legacy Job into the new
// proto Job format.
func JobToProto(jsonSrc io.Reader) (*pinpoint.Job, error) {
	l := new(jsonJob)
	if err := json.NewDecoder(jsonSrc).Decode(l); err != nil {
		return nil, errors.Annotate(err, "received ill-formed response from legacy service").Err()
	}

	return jsonJobToProto(l)
}

// jsonJobToProto translates a parsed JSON structure into a protobuf.
// It may return a partially-translated proto along with an error.
func jsonJobToProto(l *jsonJob) (*pinpoint.Job, error) {
	var errs errors.MultiError

	// FIXME(dberris): Interpret the results better, differentiating experiments from bisections, etc.
	cMode := jsonModeToProto(l.ComparisonMode)
	j := &pinpoint.Job{
		Name:           fmt.Sprintf("jobs/legacy-%s", l.JobID),
		State:          jsonStatusToProto(l.Status),
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

	// Only set ResultFiles if the job is finished, as documented in the API.
	if j.State == pinpoint.Job_SUCCEEDED {
		if resultFile, err := urlToResultFile(l.ResultsURL); err != nil {
			errs = append(errs, errors.Annotate(err, "invalid results_url from legacy service").Err())
		} else {
			j.ResultFiles = []*pinpoint.ResultFile{resultFile}
		}
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
			// Then we've got an experiment.
			if expectedStates, foundStates := 2, len(l.State); expectedStates != foundStates {
				errs = append(errs, errors.Reason("invalid state count in legacy response: want %d got %d", expectedStates, foundStates).Err())
				break
			}

			// By convention we use the first state's change to be the
			// "base" while the second state is the "experiment".
			c0 := &l.State[0].Change
			c1 := &l.State[1].Change

			experiment := &pinpoint.Experiment{
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
			}

			j.JobSpec.JobKind = &pinpoint.JobSpec_Experiment{
				Experiment: experiment,
			}

			// FIXME: Find a better way to expose this data from the legacy
			// service's JSON response instead of parsing URLs.
			if p, err := parseGerritURL(c1.Patch.URL); err != nil {
				errs = append(errs, err)
			} else {
				experiment.ExperimentPatch = &pinpoint.GerritChange{
					// FIXME: We have two URLs in the result JSON, we
					// need to extract the relevant details for the
					// proto response.
					Host:     c1.Patch.Server,
					Project:  p.project,
					Change:   p.cl,
					Patchset: p.patchSet,
				}
			}
		case "performance":
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

	if len(errs) > 0 {
		return j, errors.Annotate(errs, "%d error(s) parsing %q", len(errs), j.Name).Err()
	}
	return j, nil
}

func jsonStatusToProto(status string) pinpoint.Job_State {
	switch status {
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

func jsonModeToProto(comparisonMode string) pinpoint.JobSpec_ComparisonMode {
	switch comparisonMode {
	case "functional":
		return pinpoint.JobSpec_FUNCTIONAL
	case "try", "performance":
		return pinpoint.JobSpec_PERFORMANCE
	}
	return pinpoint.JobSpec_COMPARISON_MODE_UNSPECIFIED
}

var (
	gerritRe = regexp.MustCompile(
		`^/c/(?P<project>[^/]+)/(?P<repo>[^+]+)/\+/(?P<cl>[1-9]\d*)(/(?P<patchset>[1-9]\d*))?$`)
	gerritProjectIdx  = gerritRe.SubexpIndex("project")
	gerritRepoIdx     = gerritRe.SubexpIndex("repo")
	gerritClIdx       = gerritRe.SubexpIndex("cl")
	gerritPatchSetIdx = gerritRe.SubexpIndex("patchset")
)

type gerritParts struct {
	project, repo string
	cl, patchSet  int64
}

func parseGerritURL(s string) (*gerritParts, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, errors.Reason("invalid patch url gotten from legacy service").Err()
	}
	m := gerritRe.FindStringSubmatch(u.Path)
	if m == nil {
		return nil, errors.Reason("invalid CL path in URL gotten from legacy service: %s", u.Path).Err()
	}
	project := m[gerritProjectIdx]
	clStr := m[gerritClIdx]
	patchSetStr := m[gerritPatchSetIdx]
	repo := m[gerritRepoIdx]

	cl, err := strconv.ParseInt(clStr, 10, 64)
	if err != nil {
		return nil, errors.Reason("invalid CL number in URL gotten from legacy service").Err()
	}
	patchSet, err := strconv.ParseInt(patchSetStr, 10, 32)
	if err != nil {
		return nil, errors.Reason("invalid patchset number in URL gotten from legacy service").Err()
	}
	return &gerritParts{
		project: project, repo: repo, cl: cl, patchSet: patchSet,
	}, nil
}

var resultsURLRe = regexp.MustCompile(`https://storage.cloud.google.com/([^/]+)/(.*)$`)

func urlToResultFile(url string) (*pinpoint.ResultFile, error) {
	m := resultsURLRe.FindStringSubmatch(url)
	if m == nil {
		return nil, errors.Reason("unknown ResultFile format %q: must match %q", url, resultsURLRe).Err()
	}
	return &pinpoint.ResultFile{
		GcsBucket: m[1],
		Path:      m[2],
	}, nil
}

// JobListToProto converts a stream of JSON representing the Legacy jobs list
// response into a list of proto Jobs. Partial results may be returned along with
// an error representing failure to parse some jobs.
func JobListToProto(jsonSrc io.Reader) ([]*pinpoint.Job, error) {
	var l struct {
		Jobs []*jsonJob `json:"jobs"`
	}
	if err := json.NewDecoder(jsonSrc).Decode(&l); err != nil {
		return nil, errors.Annotate(err, "received ill-formed response from legacy service").Err()
	}

	ret := make([]*pinpoint.Job, 0, len(l.Jobs))
	var errs errors.MultiError
	for _, jj := range l.Jobs {
		if job, err := jsonJobToProto(jj); err != nil {
			errs = append(errs, err)
		} else {
			ret = append(ret, job)
		}
	}
	if len(errs) > 0 {
		return ret, errs
	}
	return ret, nil
}

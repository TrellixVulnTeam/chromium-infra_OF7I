// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	pb "go.chromium.org/luci/resultdb/proto/v1"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
)

const (
	// The execution path for tests in Skylab envrionemnt. As of 2021Q3, all tests
	// are run inside a lxc container.
	SkylabLxcJobFolder = "/usr/local/autotest/results/lxc_job_folder"
	// The execution path for tests in CFT (F20) containers.
	CFTJobFolder = "/tmp/test/results"
)

type TastResults struct {
	BaseDir string
	Cases   []TastCase
}

// Follow CrOS test platform's convention, use case to represents the single test
// executed in a Tast run. Described in
// https://pkg.go.dev/chromium.googlesource.com/chromiumos/platform/tast.git/src/chromiumos/tast/cmd/tast/internal/run/resultsjson
//
// Fields not used by Test Results are omitted.
type TastCase struct {
	Name       string      `json:"name"`
	OutDir     string      `json:"outDir"`
	SkipReason string      `json:"skipReason"`
	Errors     []TastError `json:"errors"`
	Start      time.Time   `json:"start"`
	End        time.Time   `json:"end"`
}

type TastError struct {
	Time   time.Time `json:"time"`
	Reason string    `json:"reason"`
	File   string    `json:"file"`
	Stack  string    `json:"stack"`
}

// ConvertFromJSON reads the provided reader into the receiver.
//
// The Cases are cleared and overwritten.
func (r *TastResults) ConvertFromJSON(reader io.Reader) error {
	r.Cases = []TastCase{}
	decoder := json.NewDecoder(reader)
	// Expected to parse JSON lines instead of a full JSON file.
	for decoder.More() {
		var t TastCase
		if err := decoder.Decode(&t); err != nil {
			return err
		}
		r.Cases = append(r.Cases, t)
	}
	return nil
}

// ToProtos converts test results in r to []*sinkpb.TestResult.
func (r *TastResults) ToProtos(ctx context.Context, processArtifacts func(string) (map[string]string, error)) ([]*sinkpb.TestResult, error) {
	// Convert all tast cases to TestResult.
	var ret []*sinkpb.TestResult
	for _, c := range r.Cases {
		status := genCaseStatus(c)
		tr := &sinkpb.TestResult{
			TestId:       c.Name,
			Expected:     status == pb.TestStatus_SKIP || status == pb.TestStatus_PASS,
			Status:       status,
			TestMetadata: &pb.TestMetadata{Name: c.Name},
		}
		if status == pb.TestStatus_SKIP {
			tr.SummaryHtml = "<text-artifact artifact-id=\"Skip Reason\" />"
			tr.Artifacts = map[string]*sinkpb.Artifact{
				"Skip Reason": {
					Body:        &sinkpb.Artifact_Contents{Contents: []byte(c.SkipReason)},
					ContentType: "text/plain",
				}}
			ret = append(ret, tr)
			continue
		}
		tr.Duration = msToDuration(float64(c.End.Sub(c.Start).Milliseconds()))

		d := c.OutDir
		tr.Artifacts = map[string]*sinkpb.Artifact{}
		// For Skylab tests, the OutDir recorded by tast is different from the
		// result folder we can access on Drone server.
		if strings.HasPrefix(d, SkylabLxcJobFolder) {
			d = strings.Replace(d, SkylabLxcJobFolder, r.BaseDir, 1)
		} else if strings.HasPrefix(d, CFTJobFolder) {
			d = strings.Replace(d, CFTJobFolder, r.BaseDir, 1)
		}
		normPathToFullPath, err := processArtifacts(d)
		if err != nil {
			return nil, err
		}
		for f, p := range normPathToFullPath {
			tr.Artifacts[f] = &sinkpb.Artifact{
				Body: &sinkpb.Artifact_FilePath{FilePath: p},
			}
		}

		if len(c.Errors) > 0 {
			tr.FailureReason = &pb.FailureReason{
				PrimaryErrorMessage: truncateString(c.Errors[0].Reason, maxPrimaryErrorBytes),
			}
			errLog := ""
			for _, err := range c.Errors {
				errLog += err.Stack
				errLog += "\n"
			}
			tr.Artifacts["Test Log"] = &sinkpb.Artifact{
				Body:        &sinkpb.Artifact_Contents{Contents: []byte(errLog)},
				ContentType: "text/plain",
			}
			tr.SummaryHtml = "<text-artifact artifact-id=\"Test Log\" />"
		}
		ret = append(ret, tr)
	}
	return ret, nil
}

func genCaseStatus(c TastCase) pb.TestStatus {
	if c.SkipReason != "" {
		return pb.TestStatus_SKIP
	}
	if len(c.Errors) > 0 {
		return pb.TestStatus_FAIL
	}
	return pb.TestStatus_PASS
}

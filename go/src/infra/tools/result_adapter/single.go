// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"strings"

	pb "go.chromium.org/luci/resultdb/proto/v1"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
)

// SingleResult represents result format for a test suite with a single test.
type SingleResult struct {
	Failures []string `json:"failures"`
	Valid    bool     `json:"valid"`
}

// ConvertFromJSON reads the provided reader into the receiver.
//
// The receiver is cleared and its fields overwritten.
func (r *SingleResult) ConvertFromJSON(reader io.Reader) error {
	*r = SingleResult{}
	if err := json.NewDecoder(reader).Decode(r); err != nil {
		return err
	}

	return nil
}

// ToProtos converts test results in r to []*sinkpb.TestResult.
func (r *SingleResult) ToProtos(ctx context.Context) ([]*sinkpb.TestResult, error) {
	tr := &sinkpb.TestResult{
		// For a test suite with a single test, the suite itself is one test.
		TestId: "",
	}

	switch {
	case !r.Valid:
		tr.Expected = false
		tr.Status = pb.TestStatus_ABORT
	case len(r.Failures) == 0:
		tr.Expected = true
		tr.Status = pb.TestStatus_PASS
	default:
		tr.Expected = false
		tr.Status = pb.TestStatus_FAIL
		tr.SummaryHtml = fmt.Sprintf("<pre>%s</pre>", html.EscapeString(strings.Join(r.Failures, "\n")))
	}

	return []*sinkpb.TestResult{tr}, nil
}

// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package buildextract contains structs useful in deserializing json data from
// CBE

package messages

import (
	"fmt"
	"regexp"
	"strconv"
)

// Build represents a buildbot build.
type Build struct {
	BuilderName string `json:"builderName"`
}

// Change is an automatically generated type.
type Change struct {
	At         string          `json:"at"`
	Branch     string          `json:"branch"`
	Category   string          `json:"category"`
	Comments   string          `json:"comments"`
	Files      []Files         `json:"files"`
	Number     int64           `json:"number"`
	Project    string          `json:"project"`
	Properties [][]interface{} `json:"properties"`
	Repository string          `json:"repository"`
	Rev        string          `json:"rev"`
	Revision   string          `json:"revision"`
	Revlink    string          `json:"revlink"`
	When       EpochTime       `json:"when"`
	Who        string          `json:"who"`
}

var cpRE = regexp.MustCompile("(?m:^Cr-Commit-Position: (.*)@{#([0-9]+)})")

// CommitPosition parses the comments of a change to find something which
// looks like a commit position git footer.
func (c *Change) CommitPosition() (string, int, error) {
	parts := cpRE.FindAllStringSubmatch(c.Comments, -1)
	branch, pos := "", 0
	if len(parts) > 0 {
		branch = parts[0][1]
		var err error
		pos, err = strconv.Atoi(parts[0][2])
		if err != nil {
			return "", 0, err
		}
	}

	return branch, pos, nil
}

// Step is an automatically generated type.
type Step struct {
	Eta          EpochTime         `json:"eta"`
	Expectations [][]interface{}   `json:"expectations"`
	Hidden       bool              `json:"hidden"`
	IsFinished   bool              `json:"isFinished"`
	IsStarted    bool              `json:"isStarted"`
	Logs         [][]interface{}   `json:"logs"`
	Links        map[string]string `json:"urls"`
	Name         string            `json:"name"`
	// Results is a homogenous array. Use runtime introspection to
	// determine element types.
	Results    []interface{} `json:"results"`
	StepNumber float64       `json:"step_number"`
	Text       []string      `json:"text"`
	Times      []EpochTime   `json:"times"`
}

const (
	// ResultOK is a step result which is deemed as ok. For some reason, 1 is not
	// a failure. Copied from legacy code :/
	ResultOK = float64(1)
	// ResultInfraFailure is a step result which is deemed an infra failure.
	ResultInfraFailure = float64(4)
)

// IsOK returns if the step had an "ok" result. Ok means it didn't fail.
func (s *Step) IsOK() (bool, error) {
	r, err := s.Result()
	if err != nil {
		return false, err
	}

	return r <= ResultOK, nil
}

// Result returns the step result. It does some runtime parsing, because
// buildbot's json is weird and untyped :(
func (s *Step) Result() (float64, error) {
	if r, ok := s.Results[0].(float64); ok {
		// This 0/1 check seems to be a convention or heuristic. A 0 or 1
		// result is apparently "ok", according to the original python code.
		return r, nil
	}

	return 0, fmt.Errorf("Couldn't unmarshal first step result into a float64: %v", s.Results[0])
}

// Files is an automatically generated type.
type Files struct {
	Name string                 `json:"name"`
	URL  map[string]interface{} `json:"url"`
}

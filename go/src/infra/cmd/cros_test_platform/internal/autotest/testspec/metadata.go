// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testspec

import (
	"io"
	"io/ioutil"
	"sort"

	"infra/cmd/cros_test_platform/internal/utils"

	"go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/luci/common/errors"
)

// Get computes metadata for all test and suite control files
// found within the directory tree rooted at root.
//
// Get always returns a valid api.TestMetadataResponse. In case of
// errors, the returned metadata corredsponds to the successfully parsed
// control files.
func Get(root string) (*api.TestMetadataResponse, errors.MultiError) {
	g := getter{
		controlFileLoader:   &controlFilesLoaderImpl{},
		parseTestControlFn:  parseTestControl,
		parseSuiteControlFn: parseSuiteControl,
	}
	return g.Get(root)
}

type controlFileLoader interface {
	Discover(string) error
	Tests() map[string]io.Reader
	Suites() map[string]io.Reader
}

type testMetadata struct {
	api.AutotestTest
	Suites []string
}
type parseTestControlFn func(string) (*testMetadata, errors.MultiError)
type parseSuiteControlFn func(string) (*api.AutotestSuite, errors.MultiError)

type getter struct {
	controlFileLoader   controlFileLoader
	parseTestControlFn  parseTestControlFn
	parseSuiteControlFn parseSuiteControlFn
}

func (m *testMetadata) Validate() errors.MultiError {
	var merr errors.MultiError
	if m.AutotestTest.GetName() == "" {
		merr = append(merr, errors.Reason("missing name").Err())
	}
	if m.AutotestTest.GetExecutionEnvironment() == api.AutotestTest_EXECUTION_ENVIRONMENT_UNSPECIFIED {
		merr = append(merr, errors.Reason("unspecified execution environment").Err())
	}
	return removeNilErrors(merr)
}

func (g *getter) Get(root string) (*api.TestMetadataResponse, errors.MultiError) {
	if err := g.controlFileLoader.Discover(root); err != nil {
		return nil, errors.NewMultiError(errors.Annotate(err, "get autotest metadata").Err())
	}

	var merr errors.MultiError
	tests, errs := g.parseTests(g.controlFileLoader.Tests())
	merr = append(merr, errs...)
	suites, errs := g.parseSuites(g.controlFileLoader.Suites())
	merr = append(merr, errs...)

	collectTestsInSuites(tests, suites)
	sortTestsInSuites(suites)
	return &api.TestMetadataResponse{
		Autotest: &api.AutotestTestMetadata{
			Suites: suites,
			Tests:  extractAutotestTests(tests),
		},
	}, removeNilErrors(merr)
}

func (g *getter) parseTests(controls map[string]io.Reader) ([]*testMetadata, errors.MultiError) {
	var merr errors.MultiError
	tests := make([]*testMetadata, 0, len(controls))
	for n, t := range controls {
		bt, err := ioutil.ReadAll(t)
		if err != nil {
			merr = append(merr, errors.Annotate(err, "parse test %s", n).Err())
			continue
		}
		tm, errs := g.parseTestControlFn(string(bt))
		if errs != nil {
			merr = append(merr, utils.AnnotateEach(errs, "prase test %s", n)...)
			continue
		}
		if errs := tm.Validate(); errs != nil {
			merr = append(merr, utils.AnnotateEach(errs, "prase test %s", n)...)
			continue
		}
		tests = append(tests, tm)
	}
	return tests, removeNilErrors(merr)
}

func (g *getter) parseSuites(controls map[string]io.Reader) ([]*api.AutotestSuite, errors.MultiError) {
	var merr errors.MultiError
	suites := make([]*api.AutotestSuite, 0, len(controls))
	for n, t := range controls {
		bt, err := ioutil.ReadAll(t)
		if err != nil {
			merr = append(merr, errors.Annotate(err, "parse suite %s", n).Err())
			continue
		}
		sm, errs := g.parseSuiteControlFn(string(bt))
		if errs != nil {
			for _, err := range errs {
				merr = append(merr, errors.Annotate(err, "parse suite %s", n).Err())
			}
			continue
		}
		suites = append(suites, sm)
	}
	return suites, removeNilErrors(merr)
}

func collectTestsInSuites(tests []*testMetadata, suites []*api.AutotestSuite) {
	sm := make(map[string]*api.AutotestSuite)
	for _, s := range suites {
		sm[s.GetName()] = s
	}
	for _, t := range tests {
		for _, sn := range t.Suites {
			if s, ok := sm[sn]; ok {
				appendTestToSuite(t, s)
			}
		}
	}
}

func sortTestsInSuites(suites []*api.AutotestSuite) {
	for _, s := range suites {
		sort.SliceStable(s.Tests, func(i, j int) bool {
			return s.Tests[i].Name < s.Tests[j].Name
		})
	}
}

func appendTestToSuite(test *testMetadata, suite *api.AutotestSuite) {
	suite.Tests = append(suite.Tests, &api.AutotestSuite_TestReference{Name: test.GetName()})
}

func extractAutotestTests(tests []*testMetadata) []*api.AutotestTest {
	at := make([]*api.AutotestTest, 0, len(tests))
	for _, t := range tests {
		at = append(at, &t.AutotestTest)
	}
	return at
}

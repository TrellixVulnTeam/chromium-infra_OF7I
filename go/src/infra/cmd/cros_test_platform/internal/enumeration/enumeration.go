// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package enumeration

import (
	"go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
)

// GetForTests returns the test metadata for specified tests.
func GetForTests(metadata *api.AutotestTestMetadata, tests []*test_platform.Request_Test) ([]*steps.EnumerationResponse_AutotestInvocation, error) {
	tNames, err := testNames(tests)
	if err != nil {
		return nil, err
	}
	ts := testsByName(filterTests(metadata.GetTests(), tNames))

	var invs []*steps.EnumerationResponse_AutotestInvocation
	for _, tr := range tests {
		// Any tests of incorrect type would already be caught by testNames()
		// above. Better panic in case of failure here.
		h := tr.Harness.(*test_platform.Request_Test_Autotest_)
		if t, ok := ts[h.Autotest.Name]; ok {
			invs = append(invs, &steps.EnumerationResponse_AutotestInvocation{
				Test:        t,
				TestArgs:    h.Autotest.TestArgs,
				DisplayName: h.Autotest.DisplayName,
			})
		}
	}
	return invs, nil
}

// GetForSuites returns the test metadata for specified suites.
func GetForSuites(metadata *api.AutotestTestMetadata, suites []*test_platform.Request_Suite) []*steps.EnumerationResponse_AutotestInvocation {
	invs := []*steps.EnumerationResponse_AutotestInvocation{}
	for _, suite := range enumeratedSuitesWithNames(metadata.GetSuites(), suites) {
		tNames := extractTestNames(suite)
		tests := filterTests(metadata.GetTests(), tNames)
		invs = append(invs, autotestInvocationsForSuite(suite, tests)...)
	}
	return invs
}

// GetForEnumeration marshals the provided pre-enumerated tests into standard
// enumeration response format.
func GetForEnumeration(enumeration *test_platform.Request_Enumeration) []*steps.EnumerationResponse_AutotestInvocation {
	ret := make([]*steps.EnumerationResponse_AutotestInvocation, 0, len(enumeration.GetAutotestInvocations()))
	for _, t := range enumeration.GetAutotestInvocations() {
		ret = append(ret, &steps.EnumerationResponse_AutotestInvocation{
			Test:          t.GetTest(),
			TestArgs:      t.GetTestArgs(),
			DisplayName:   t.GetDisplayName(),
			ResultKeyvals: t.GetResultKeyvals(),
		})
	}
	return ret
}

func filterTests(tests []*api.AutotestTest, keep stringset.Set) []*api.AutotestTest {
	ret := make([]*api.AutotestTest, 0, len(keep))
	for _, t := range tests {
		if keep.Has(t.GetName()) {
			ret = append(ret, t)
		}
	}
	return ret
}

func testsByName(tests []*api.AutotestTest) map[string]*api.AutotestTest {
	ret := make(map[string]*api.AutotestTest)
	for _, t := range tests {
		ret[t.GetName()] = t
	}
	return ret
}

func autotestInvocationsForSuite(suite *api.AutotestSuite, tests []*api.AutotestTest) []*steps.EnumerationResponse_AutotestInvocation {
	ret := make([]*steps.EnumerationResponse_AutotestInvocation, 0, len(tests))
	for _, t := range tests {
		inv := steps.EnumerationResponse_AutotestInvocation{
			Test: t,
			ResultKeyvals: map[string]string{
				"suite": suite.Name,
			},
		}
		inv.Test.Dependencies = appendNewDependencies(inv.Test.Dependencies, suite.GetChildDependencies())
		ret = append(ret, &inv)
	}
	return ret
}

func appendNewDependencies(to, from []*api.AutotestTaskDependency) []*api.AutotestTaskDependency {
	seen := stringset.New(len(to))
	for _, d := range to {
		seen.Add(d.Label)
	}
	for _, d := range from {
		if !seen.Has(d.Label) {
			to = append(to, d)
		}
	}
	return to
}

func testNames(ts []*test_platform.Request_Test) (stringset.Set, error) {
	ns := stringset.New(len(ts))
	for _, t := range ts {
		switch h := t.GetHarness().(type) {
		case *test_platform.Request_Test_Autotest_:
			ns.Add(h.Autotest.Name)
		default:
			return nil, errors.Reason("unknown harness %+v", h).Err()
		}
	}
	return ns, nil
}

func enumeratedSuitesWithNames(enumeratedSuites []*api.AutotestSuite, requestedSuites []*test_platform.Request_Suite) []*api.AutotestSuite {
	sNames := stringset.New(len(requestedSuites))
	for _, s := range requestedSuites {
		sNames.Add(s.GetName())
	}

	var ret []*api.AutotestSuite
	for _, s := range enumeratedSuites {
		if sNames.Has(s.GetName()) {
			ret = append(ret, s)
		}
	}
	return ret
}

func extractTestNames(s *api.AutotestSuite) stringset.Set {
	tNames := stringset.New(len(s.GetTests()))
	for _, t := range s.GetTests() {
		tNames.Add(t.GetName())
	}
	return tNames
}

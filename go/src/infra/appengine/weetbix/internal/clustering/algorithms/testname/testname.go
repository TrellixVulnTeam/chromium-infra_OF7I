// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package testname contains the test name-based clustering algorithm for Weetbix.
package testname

import (
	"crypto/sha256"
	"fmt"
	"strconv"

	"infra/appengine/weetbix/internal/clustering"

	"go.chromium.org/luci/common/errors"
)

// TODO(crbug.com/1243174): Make this configurable on a per-project basis.
var rules = []*ClusteringRule{
	{
		Name:         "Blink Web Tests",
		Pattern:      `^ninja://:blink_web_tests/` + `(virtual/[^/]+/)?(?P<testname>([^/]+/)+[^/]+\.[a-zA-Z]+).*$`,
		LikeTemplate: "ninja://:blink\\_web\\_tests/%${testname}%",
	},
	{
		Name: "Google Test (Value-parameterized)",
		// E.g. ninja:{target}/Prefix/ColorSpaceTest.testNullTransform/11
		// Note that "Prefix/" portion may be blank/omitted.
		Pattern:      `^ninja:(?P<target>[\w/]+:\w+)/` + `(\w+/)?(?P<suite>\w+)\.(?P<case>\w+)/\w+$`,
		LikeTemplate: `ninja:${target}/%${suite}.${case}%`,
	},
	{
		Name: "Google Test (Type-parameterized)",
		// E.g. ninja:{target}/Prefix/GLES2DecoderPassthroughFixedCommandTest/5.InvalidCommand
		// Note that "Prefix/" portion may be blank/omitted.
		// https://github.com/google/googletest/blob/1b18723e874b256c1e39378c6774a90701d70f7a/googletest/include/gtest/internal/gtest-internal.h#L710
		Pattern:      `^ninja:(?P<target>[\w/]+:\w+)/` + `(\w+/)?(?P<suite>\w+)/\w+\.(?P<case>\w+)$`,
		LikeTemplate: `ninja:${target}/%${suite}/%.${case}`,
	},
}

var compiledRules []RuleEvaluator

func init() {
	compiledRules = make([]RuleEvaluator, len(rules))
	for i, rule := range rules {
		eval, err := rule.Compile()
		if err != nil {
			panic(errors.Annotate(err, "compiling test name clustering rule").Err())
		}
		compiledRules[i] = eval
	}
}

// AlgorithmVersion is the version of the clustering algorithm. The algorithm
// version should be incremented whenever existing test results may be
// clustered differently (i.e. Cluster(f) returns a different value for some
// f that may have been already ingested).
const AlgorithmVersion = 2

// AlgorithmName is the identifier for the clustering algorithm.
// Weetbix requires all clustering algorithms to have a unique identifier.
// Must match the pattern ^[a-z0-9-.]{1,32}$.
//
// The AlgorithmName must encode the algorithm version, so that each version
// of an algorithm has a different name.
var AlgorithmName = fmt.Sprintf("testname-v%v", AlgorithmVersion)

// Algorithm represents an instance of the test name-based clustering
// algorithm.
type Algorithm struct{}

// Name returns the identifier of the clustering algorithm.
func (a *Algorithm) Name() string {
	return AlgorithmName
}

// clusterLike returns the test name LIKE expression that defines
// the cluster the given test result belongs to.
//
// By default this LIKE expression encodes just the test
// name itself. However, by using rules, projects can configure
// it to mask out parts of the test name (e.g. corresponding
// to test variants).
// "ninja://chrome/test:interactive_ui_tests/ColorSpaceTest.testNullTransform/%"
func clusterLike(failure *clustering.Failure) (like string, ok bool) {
	testID := failure.TestID
	for _, r := range compiledRules {
		like, ok := r(testID)
		if ok {
			return like, true
		}
	}
	// No rule matches. Match the test name literally.
	return "", false
}

// Cluster clusters the given test failure and returns its cluster ID (if it
// can be clustered) or nil otherwise.
func (a *Algorithm) Cluster(failure *clustering.Failure) []byte {
	// Get the like expression that defines the cluster.
	key, ok := clusterLike(failure)
	if !ok {
		// Fall back to clustering on the exact test name.
		key = failure.TestID
	}

	// Hash the expressionto generate a unique fingerprint.
	h := sha256.Sum256([]byte(key))
	// Take first 16 bytes as the ID. (Risk of collision is
	// so low as to not warrant full 32 bytes.)
	return h[0:16]
}

const bugDescriptionTemplateLike = `This bug is for all test failures with a test name like: %s`
const bugDescriptionTemplateExact = `This bug is for all test failures with the test name: %s`

// ClusterDescription returns a description of the cluster, for use when
// filing bugs, with the help of the given example failure.
func (a *Algorithm) ClusterDescription(example *clustering.Failure) *clustering.ClusterDescription {
	// Get the like expression that defines the cluster.
	like, ok := clusterLike(example)
	if ok {
		return &clustering.ClusterDescription{
			Title:       like,
			Description: fmt.Sprintf(bugDescriptionTemplateLike, like),
		}
	} else {
		// No matching clustering rule. Fall back to the exact test name.
		return &clustering.ClusterDescription{
			Title:       example.TestID,
			Description: fmt.Sprintf(bugDescriptionTemplateExact, example.TestID),
		}

	}
}

// FailureAssociationRule returns a failure association rule that
// captures the definition of cluster containing the given example.
func (a *Algorithm) FailureAssociationRule(example *clustering.Failure) string {
	like, ok := clusterLike(example)
	if ok {
		return fmt.Sprintf("test LIKE %s", strconv.QuoteToGraphic(like))
	} else {
		return fmt.Sprintf("test = %s", strconv.QuoteToGraphic(example.TestID))
	}
}

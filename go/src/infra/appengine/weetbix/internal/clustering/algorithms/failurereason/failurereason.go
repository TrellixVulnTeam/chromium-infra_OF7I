// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package failurereason contains the failure reason clustering algorithm
// for Weetbix.
//
// This algorithm removes ips, temp file names, numbers and other such tokens
// to cluster similar reasons together.
package failurereason

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/config/compiledcfg"
)

// AlgorithmVersion is the version of the clustering algorithm. The algorithm
// version should be incremented whenever existing test results may be
// clustered differently (i.e. Cluster(f) returns a different value for some
// f that may have been already ingested).
const AlgorithmVersion = 3

// AlgorithmName is the identifier for the clustering algorithm.
// Weetbix requires all clustering algorithms to have a unique identifier.
// Must match the pattern ^[a-z0-9-.]{1,32}$.
//
// The AlgorithmName must encode the algorithm version, so that each version
// of an algorithm has a different name.
var AlgorithmName = fmt.Sprintf("%sv%v", clustering.FailureReasonAlgorithmPrefix, AlgorithmVersion)

// BugTemplate is the template for the content of bugs created for failure
// reason clusters. A list of test IDs is included to improve searchability
// by test name.
var BugTemplate = template.Must(template.New("reasonTemplate").Parse(
	`This bug is for all test failures where the primary error message is similiar to the following (ignoring numbers and hexadecimal values):
{{.FailureReason}}

The following test(s) were observed to have matching failures at this time (at most five examples listed):
{{range .TestIDs}}- {{.}}
{{end}}`))

// To match any 1 or more digit numbers, or hex values (often appear in temp
// file names or prints of pointers), which will be replaced.
var clusterExp = regexp.MustCompile(`[/+0-9a-zA-Z]{10,}=+|[\-0-9a-fA-F \t]{16,}|[0-9a-fA-Fx]{8,}|[0-9]+`)

// Algorithm represents an instance of the reason-based clustering
// algorithm.
type Algorithm struct{}

// Name returns the identifier of the clustering algorithm.
func (a *Algorithm) Name() string {
	return AlgorithmName
}

// Cluster clusters the given test failure and returns its cluster ID (if it
// can be clustered) or nil otherwise.
func (a *Algorithm) Cluster(config *compiledcfg.ProjectConfig, failure *clustering.Failure) []byte {
	if failure.Reason == nil || failure.Reason.PrimaryErrorMessage == "" {
		return nil
	}
	// Replace numbers and hex values.
	id := clusterExp.ReplaceAllString(failure.Reason.PrimaryErrorMessage, "0")
	// sha256 hash the resulting string.
	h := sha256.Sum256([]byte(id))
	// Take first 16 bytes as the ID. (Risk of collision is
	// so low as to not warrant full 32 bytes.)
	return h[0:16]
}

// ClusterDescription returns a description of the cluster, for use when
// filing bugs, with the help of the given example failure.
func (a *Algorithm) ClusterDescription(config *compiledcfg.ProjectConfig, summary *clustering.ClusterSummary) (*clustering.ClusterDescription, error) {
	if summary.Example.Reason == nil || summary.Example.Reason.PrimaryErrorMessage == "" {
		return nil, errors.New("cluster summary must contain example with failure reason")
	}
	type templateData struct {
		FailureReason string
		TestIDs       []string
	}
	var input templateData

	// Quote and escape.
	primaryError := strconv.QuoteToGraphic(summary.Example.Reason.PrimaryErrorMessage)
	// Unquote, so we are left with the escaped error message only.
	primaryError = primaryError[1 : len(primaryError)-1]

	input.FailureReason = primaryError
	for _, t := range summary.TopTests {
		input.TestIDs = append(input.TestIDs, clustering.EscapeToGraphical(t))
	}
	var b bytes.Buffer
	if err := BugTemplate.Execute(&b, input); err != nil {
		return nil, err
	}

	return &clustering.ClusterDescription{
		Title:       primaryError,
		Description: b.String(),
	}, nil
}

// ClusterTitle returns a title for the cluster containing the given
// example, to display on the cluster page or cluster listing.
func (a *Algorithm) ClusterTitle(config *compiledcfg.ProjectConfig, example *clustering.Failure) string {
	if example.Reason == nil || example.Reason.PrimaryErrorMessage == "" {
		return ""
	}
	return clustering.EscapeToGraphical(example.Reason.PrimaryErrorMessage)
}

// FailureAssociationRule returns a failure association rule that
// captures the definition of cluster containing the given example.
func (a *Algorithm) FailureAssociationRule(config *compiledcfg.ProjectConfig, example *clustering.Failure) string {
	if example.Reason == nil || example.Reason.PrimaryErrorMessage == "" {
		return ""
	}
	// Escape \, % and _ so that they are not interpreted as by LIKE
	// pattern matching.
	rewriter := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	likePattern := rewriter.Replace(example.Reason.PrimaryErrorMessage)

	// Replace hexadecimal sequences with wildcard matches. This is technically
	// broader than our original cluster definition, but is more readable, and
	// usually ends up matching the exact same set of failures.
	likePattern = clusterExp.ReplaceAllString(likePattern, "%")

	// Escape the pattern as a string literal.
	stringLiteral := strconv.QuoteToGraphic(likePattern)
	return fmt.Sprintf("reason LIKE %s", stringLiteral)
}

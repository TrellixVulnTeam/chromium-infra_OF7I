// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugs

import (
	"fmt"
	"regexp"

	mpb "infra/monorailv2/api/v3/api_proto"

	"infra/appengine/weetbix/internal/clustering"
)

const (
	FailureReasonTemplate = `This bug is for all test failures where the primary error message is similar to the following (ignoring numbers and hexadecimal values):
%s

This bug has been automatically filed by Weetbix in response to a cluster of test failures.`

	TestNameTemplate = `This bug is for all test failures with the test name: %s

This bug has been automatically filed by Weetbix in response to a cluster of test failures.`
)

// whitespaceRE matches blocks of whitespace, including new lines tabs and
// spaces.
var whitespaceRE = regexp.MustCompile(`[ \t\n]+`)

// IssueGenerator generates bugs for failure clusters.
type IssueGenerator struct {
	// reporter is the email address of the user that will be listed as the
	// issue reporter in monorail. This is usually the GAE service account.
	reporter string
}

// NewIssueGenerator initialises a new IssueGenerator.
func NewIssueGenerator(reporter string) *IssueGenerator {
	return &IssueGenerator{
		reporter: reporter,
	}
}

// PrepareNew prepares a new bug based on the given cluster.
func (i *IssueGenerator) PrepareNew(cluster *clustering.Cluster) *mpb.MakeIssueRequest {
	title := cluster.ClusterID
	if cluster.ExampleFailureReason.Valid {
		title = cluster.ExampleFailureReason.StringVal
	}
	return &mpb.MakeIssueRequest{
		// Analysis clusters are currently hardcoded to one project: chromium.
		// We also have no configuration mapping LUCI projects to the monorail
		// projects.
		Parent: "chromium",
		Issue: &mpb.Issue{
			Summary:  fmt.Sprintf("Tests are failing: %v", sanitiseTitle(title, 150)),
			Reporter: i.reporter,
			State:    mpb.IssueContentState_ACTIVE,
			Status:   &mpb.Issue_StatusValue{Status: "Untriaged"},
			Labels: []*mpb.Issue_LabelValue{{
				Label: "Restrict-View-Google",
			}, {
				Label: "Weetbix-Managed",
			}},
		},
		Description: bugDescription(cluster),
		NotifyType:  mpb.NotifyType_NO_NOTIFICATION,
	}
}

// bugDescription returns the description that should be used when creating
// a new bug for the given cluster.
func bugDescription(cluster *clustering.Cluster) string {
	if cluster.ExampleFailureReason.Valid {
		// We should escape the failure reason in future, if monorail is
		// extended to support markdown.
		return fmt.Sprintf(FailureReasonTemplate, cluster.ExampleFailureReason.String())
	} else {
		return fmt.Sprintf(TestNameTemplate, cluster.ClusterID)
	}
}

// sanitiseTitle removes tabs and line breaks from input, replacing them with
// spaces, and truncates the output to the given number of runes.
func sanitiseTitle(input string, maxLength int) string {
	// Replace blocks of whitespace, including new lines and tabs, with just a
	// single space.
	strippedInput := whitespaceRE.ReplaceAllString(input, " ")

	// Truncate to desired length.
	runes := []rune(strippedInput)
	if len(runes) > maxLength {
		return string(runes[0:maxLength-3]) + "..."
	}
	return strippedInput
}

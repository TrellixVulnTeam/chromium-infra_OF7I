// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"fmt"
	"regexp"
	"strings"

	mpb "infra/monorailv2/api/v3/api_proto"

	"infra/appengine/weetbix/internal/clustering"

	"google.golang.org/genproto/protobuf/field_mask"
)

const (
	FailureReasonTemplate = `This bug is for all test failures where the primary error message is similar to the following (ignoring numbers and hexadecimal values):
%s

This bug has been automatically filed by Weetbix in response to a cluster of test failures.`

	TestNameTemplate = `This bug is for all test failures with the test name: %s

This bug has been automatically filed by Weetbix in response to a cluster of test failures.`
)

const (
	manualPriorityLabel = "Weetbix-Manual-Priority"
	restrictViewLabel   = "Restrict-View-Google"
	managedLabel        = "Weetbix-Managed"
)

const (
	// typeFieldName is the name of the type field in the chromium project.
	// For other projects, the definitions vary.
	typeFieldName = "projects/chromium/fieldDefs/10"
	// priorityFieldName is the name of the priority field in the chromium
	// project. for other projects, the definitions vary.
	priorityFieldName = "projects/chromium/fieldDefs/11"
)

// whitespaceRE matches blocks of whitespace, including new lines tabs and
// spaces.
var whitespaceRE = regexp.MustCompile(`[ \t\n]+`)

// priorityRE matches chromium monorail priority values.
var priorityRE = regexp.MustCompile(`^Pri-([0123])$`)

// AutomationUsers are the identifiers of Weetbix automation users in monorail.
var AutomationUsers = []string{
	"users/4149141945", // chops-weetbix-dev@appspot.gserviceaccount.com
}

// PrepareNew prepares a new bug from the given cluster.
func PrepareNew(cluster *clustering.Cluster) *mpb.MakeIssueRequest {
	title := cluster.ClusterID
	if cluster.ExampleFailureReason.Valid {
		title = cluster.ExampleFailureReason.StringVal
	}
	return &mpb.MakeIssueRequest{
		// Analysis clusters are currently hardcoded to one project: chromium.
		// We also have no configuration mapping LUCI projects to the monorail
		// projects.
		Parent: "projects/chromium",
		Issue: &mpb.Issue{
			Summary: fmt.Sprintf("Tests are failing: %v", sanitiseTitle(title, 150)),
			State:   mpb.IssueContentState_ACTIVE,
			Status:  &mpb.Issue_StatusValue{Status: "Untriaged"},
			FieldValues: []*mpb.FieldValue{
				{
					Field: typeFieldName,
					Value: "Bug",
				},
				{
					Field: priorityFieldName,
					Value: clusterPriority(cluster),
				},
			},
			Labels: []*mpb.Issue_LabelValue{{
				Label: restrictViewLabel,
			}, {
				Label: managedLabel,
			}},
		},
		Description: bugDescription(cluster),
		NotifyType:  mpb.NotifyType_NO_NOTIFICATION,
	}
}

// NeedsUpdate determines if the bug for the given cluster needs to be updated.
func NeedsUpdate(cluster *clustering.Cluster, issue *mpb.Issue) bool {
	// Bugs must have restrict view label to be updated.
	if !hasLabel(issue, restrictViewLabel) {
		return false
	}
	if hasLabel(issue, manualPriorityLabel) {
		return false
	}
	return clusterPriority(cluster) != IssuePriority(issue)
}

// MakeUpdate prepares an updated for the bug associated with a given cluster.
// Must ONLY be called if NeedsUpdate(...) returns true.
func MakeUpdate(cluster *clustering.Cluster, issue *mpb.Issue, comments []*mpb.Comment) *mpb.ModifyIssuesRequest {
	if hasManuallySetPriority(comments) {
		// We were not the last to update the priority of this issue.
		// Set the 'manually controlled priority' label to reflect
		// the state of this bug and avoid further attempts to update.
		return &mpb.ModifyIssuesRequest{
			Deltas: []*mpb.IssueDelta{
				{
					Issue: &mpb.Issue{
						Name: issue.Name,
						Labels: []*mpb.Issue_LabelValue{{
							Label: manualPriorityLabel,
						}},
					},
					UpdateMask: &field_mask.FieldMask{
						Paths: []string{"labels"},
					},
				},
			},
			NotifyType:     mpb.NotifyType_NO_NOTIFICATION,
			CommentContent: fmt.Sprintf("The bug priority has been manually set. To re-enable automatic priority updates by Weetbix, remove the %s label.", manualPriorityLabel),
		}
	}
	// We were the last to update the bug priority.
	// Apply the priority update.
	return &mpb.ModifyIssuesRequest{
		Deltas: []*mpb.IssueDelta{
			{
				Issue: &mpb.Issue{
					Name: issue.Name,
					FieldValues: []*mpb.FieldValue{
						{
							Field: priorityFieldName,
							Value: clusterPriority(cluster),
						},
					},
				},
				UpdateMask: &field_mask.FieldMask{
					Paths: []string{"field_values"},
				},
			},
		},
		NotifyType:     mpb.NotifyType_NO_NOTIFICATION,
		CommentContent: "The impact of this bug's test failures has changed. Weetbix has adjusted the bug priority.",
	}
}

// hasManuallySetPriority returns whether the the given issue has a manually
// controlled priority, based on its comments.
func hasManuallySetPriority(comments []*mpb.Comment) bool {
	// Example comment showing a user changing priority:
	// {
	// 	name: "projects/chromium/issues/915761/comments/1"
	// 	state: ACTIVE
	// 	type: COMMENT
	// 	commenter: "users/2627516260"
	// 	create_time: {
	// 	  seconds: 1632111572
	// 	}
	// 	amendments: {
	// 	  field_name: "Labels"
	// 	  new_or_delta_value: "Pri-1"
	// 	}
	// }
	for i := len(comments) - 1; i >= 0; i-- {
		c := comments[i]

		isManualPriorityUpdate := false
		isRevertToAutomaticPriority := false
		for _, a := range c.Amendments {
			if a.FieldName == "Labels" {
				deltaLabels := strings.Split(a.NewOrDeltaValue, " ")
				for _, lbl := range deltaLabels {
					if lbl == "-"+manualPriorityLabel {
						isRevertToAutomaticPriority = true
					}
					if priorityRE.MatchString(lbl) {
						if !isAutomationUser(c.Commenter) {
							isManualPriorityUpdate = true
						}
					}
				}
			}
		}
		if isRevertToAutomaticPriority {
			return false
		}
		if isManualPriorityUpdate {
			return true
		}
	}
	// No manual changes to priority indicates the bug is still under
	// automatic control.
	return false
}

func isAutomationUser(user string) bool {
	for _, u := range AutomationUsers {
		if u == user {
			return true
		}
	}
	return false
}

// hasLabel returns whether the bug the specified label.
func hasLabel(issue *mpb.Issue, label string) bool {
	for _, l := range issue.Labels {
		if l.Label == label {
			return true
		}
	}
	return false
}

// IssuePriority returns the priority of the given issue.
func IssuePriority(issue *mpb.Issue) string {
	for _, fv := range issue.FieldValues {
		if fv.Field == priorityFieldName {
			return fv.Value
		}
	}
	return ""
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

// clusterPriority returns the priority of the bug that should be created
// for a given cluster.
func clusterPriority(cluster *clustering.Cluster) string {
	switch {
	case cluster.UnexpectedFailures1d > 1000:
		return "0"
	case cluster.UnexpectedFailures1d > 500:
		return "1"
	case cluster.UnexpectedFailures1d > 100:
		return "2"
	}
	return "3"
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

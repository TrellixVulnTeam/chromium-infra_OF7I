// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"fmt"
	"regexp"
	"strings"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/config"
	mpb "infra/monorailv2/api/v3/api_proto"

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

// whitespaceRE matches blocks of whitespace, including new lines tabs and
// spaces.
var whitespaceRE = regexp.MustCompile(`[ \t\n]+`)

// priorityRE matches chromium monorail priority values.
var priorityRE = regexp.MustCompile(`^Pri-([0123])$`)

// AutomationUsers are the identifiers of Weetbix automation users in monorail.
var AutomationUsers = []string{
	"users/4149141945", // chops-weetbix-dev@appspot.gserviceaccount.com
}

// VerifiedStatus is that status of bugs that have been fixed and verified.
const VerifiedStatus = "Verified"

// AssignedStatus is the status of bugs that are open and assigned to an owner.
const AssignedStatus = "Assigned"

// UntriagedStatus is the status of bugs that have just been opened.
const UntriagedStatus = "Untriaged"

// Generator provides access to a methods to generate a new bug and/or bug
// updates for a cluster.
type Generator struct {
	// The cluster to generate monorail changes for.
	cluster *clustering.Cluster
	// The monorail configuration to use.
	monorailCfg *config.MonorailProject
}

// NewGenerator initialises a new Generator.
func NewGenerator(cluster *clustering.Cluster, monorailCfg *config.MonorailProject) *Generator {
	return &Generator{
		cluster:     cluster,
		monorailCfg: monorailCfg,
	}
}

// PrepareNew prepares a new bug from the given cluster.
func (g *Generator) PrepareNew() *mpb.MakeIssueRequest {
	title := g.cluster.ClusterID
	if g.cluster.ExampleFailureReason.Valid {
		title = g.cluster.ExampleFailureReason.StringVal
	}
	issue := &mpb.Issue{
		Summary: fmt.Sprintf("Tests are failing: %v", sanitiseTitle(title, 150)),
		State:   mpb.IssueContentState_ACTIVE,
		Status:  &mpb.Issue_StatusValue{Status: UntriagedStatus},
		FieldValues: []*mpb.FieldValue{
			{
				Field: g.priorityFieldName(),
				Value: g.clusterPriority(),
			},
		},
		Labels: []*mpb.Issue_LabelValue{{
			Label: restrictViewLabel,
		}, {
			Label: managedLabel,
		}},
	}
	for _, fv := range g.monorailCfg.DefaultFieldValues {
		issue.FieldValues = append(issue.FieldValues, &mpb.FieldValue{
			Field: fmt.Sprintf("projects/%s/fieldDefs/%v", g.monorailCfg.Project, fv.FieldId),
			Value: fv.Value,
		})
	}

	return &mpb.MakeIssueRequest{
		Parent:      fmt.Sprintf("projects/%s", g.monorailCfg.Project),
		Issue:       issue,
		Description: g.bugDescription(),
		NotifyType:  mpb.NotifyType_EMAIL,
	}
}

func (g *Generator) priorityFieldName() string {
	return fmt.Sprintf("projects/%s/fieldDefs/%v", g.monorailCfg.Project, g.monorailCfg.PriorityFieldId)
}

// NeedsUpdate determines if the bug for the given cluster needs to be updated.
func (g *Generator) NeedsUpdate(issue *mpb.Issue) bool {
	// Bugs must have restrict view label to be updated.
	if !hasLabel(issue, restrictViewLabel) {
		return false
	}
	// Cases that a bug may be updated follow.
	switch {
	case g.clusterResolved() != issueVerified(issue):
		return true
	case !hasLabel(issue, manualPriorityLabel) &&
		!g.clusterResolved() &&
		g.clusterPriority() != g.IssuePriority(issue):
		// The priority has changed on a cluster which is not verified as fixed
		// and the user isn't manually controlling the priority.
		return true
	default:
		return false
	}
}

// MakeUpdate prepares an updated for the bug associated with a given cluster.
// Must ONLY be called if NeedsUpdate(...) returns true.
func (g *Generator) MakeUpdate(issue *mpb.Issue, comments []*mpb.Comment) *mpb.ModifyIssuesRequest {
	delta := &mpb.IssueDelta{
		Issue: &mpb.Issue{
			Name: issue.Name,
		},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{},
		},
	}

	var commentary []string
	notify := false
	if g.clusterResolved() != issueVerified(issue) {
		// Verify or reopen the issue.
		comment := g.prepareBugVerifiedUpdate(issue, delta)
		commentary = append(commentary, comment)
		notify = true
	}
	if !hasLabel(issue, manualPriorityLabel) &&
		!g.clusterResolved() &&
		g.clusterPriority() != g.IssuePriority(issue) {

		if hasManuallySetPriority(comments) {
			// We were not the last to update the priority of this issue.
			// Set the 'manually controlled priority' label to reflect
			// the state of this bug and avoid further attempts to update.
			comment := prepareManualPriorityUpdate(issue, delta)
			commentary = append(commentary, comment)
		} else {
			// We were the last to update the bug priority.
			// Apply the priority update.
			comment := g.preparePriorityUpdate(issue, delta)
			commentary = append(commentary, comment)
			// Notify if new priority is higher than existing priority.
			notify = notify || g.isHigherPriority(g.clusterPriority(), g.IssuePriority(issue))
		}
	}

	update := &mpb.ModifyIssuesRequest{
		Deltas: []*mpb.IssueDelta{
			delta,
		},
		NotifyType:     mpb.NotifyType_NO_NOTIFICATION,
		CommentContent: strings.Join(commentary, "\n\n"),
	}
	if notify {
		update.NotifyType = mpb.NotifyType_EMAIL
	}
	return update
}

func (g *Generator) prepareBugVerifiedUpdate(issue *mpb.Issue, update *mpb.IssueDelta) string {
	resolved := g.clusterResolved()
	var status string
	var comment string
	if resolved {
		status = VerifiedStatus
		comment = "No further occurances of the failure cluster have been identified. Weetbix is marking the issue verified."
	} else {
		if issue.GetOwner().GetUser() != "" {
			status = AssignedStatus
		} else {
			status = UntriagedStatus
		}
		comment = "Weetbix has identified new occurances of the failure cluster. The bug has been re-opened."
	}
	update.Issue.Status = &mpb.Issue_StatusValue{Status: status}
	update.UpdateMask.Paths = append(update.UpdateMask.Paths, "status")
	return comment
}

func prepareManualPriorityUpdate(issue *mpb.Issue, update *mpb.IssueDelta) string {
	update.Issue.Labels = []*mpb.Issue_LabelValue{{
		Label: manualPriorityLabel,
	}}
	update.UpdateMask.Paths = append(update.UpdateMask.Paths, "labels")
	return fmt.Sprintf("The bug priority has been manually set. To re-enable automatic priority updates by Weetbix, remove the %s label.", manualPriorityLabel)
}

func (g *Generator) preparePriorityUpdate(issue *mpb.Issue, update *mpb.IssueDelta) string {
	update.Issue.FieldValues = []*mpb.FieldValue{
		{
			Field: g.priorityFieldName(),
			Value: g.clusterPriority(),
		},
	}
	update.UpdateMask.Paths = append(update.UpdateMask.Paths, "field_values")
	return fmt.Sprintf("The impact of this bug's test failures has changed. "+
		"Weetbix has adjusted the bug priority from %v to %v.", g.IssuePriority(issue), g.clusterPriority())
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
func (g *Generator) IssuePriority(issue *mpb.Issue) string {
	priorityFieldName := g.priorityFieldName()
	for _, fv := range issue.FieldValues {
		if fv.Field == priorityFieldName {
			return fv.Value
		}
	}
	return ""
}

func issueVerified(issue *mpb.Issue) bool {
	return issue.Status.Status == VerifiedStatus
}

// isHigherPriority returns whether priority p1 is higher than priority p2.
func (g *Generator) isHigherPriority(p1 string, p2 string) bool {
	i1 := g.indexOfPriority(p1)
	i2 := g.indexOfPriority(p2)
	// higher priority means lower index.
	return i1 < i2
}

func (g *Generator) indexOfPriority(priority string) int {
	for i, p := range g.monorailCfg.Priorities {
		if p.Priority == priority {
			return i
		}
	}
	// If we can't find the priority, treat it as one lower than
	// the lowest priority we know about.
	return len(g.monorailCfg.Priorities)
}

// bugDescription returns the description that should be used when creating
// a new bug for the cluster.
func (g *Generator) bugDescription() string {
	if g.cluster.ExampleFailureReason.Valid {
		// We should escape the failure reason in future, if monorail is
		// extended to support markdown.
		return fmt.Sprintf(FailureReasonTemplate, g.cluster.ExampleFailureReason.String())
	} else {
		return fmt.Sprintf(TestNameTemplate, g.cluster.ClusterID)
	}
}

// clusterPriority returns the priority of the bug that should be created
// for the cluster.
func (g *Generator) clusterPriority() string {
	if len(g.monorailCfg.Priorities) == 0 {
		// This should never happen; it means configuration is being used for
		// a project that has never passed validation.
		panic(fmt.Sprintf("invalid configuration in use for monorail project %q; no monorail priorities configured", g.monorailCfg.Project))
	}
	// Default to using the lowest priority.
	priority := g.monorailCfg.Priorities[len(g.monorailCfg.Priorities)-1]
	for i := len(g.monorailCfg.Priorities) - 2; i >= 0; i-- {
		p := g.monorailCfg.Priorities[i]
		if !g.cluster.MeetsThreshold(p.Threshold) {
			// A cluster cannot reach a higher priority unless it has
			// met the thresholds for all lower priorities.
			break
		}
		priority = p
	}
	return priority.Priority
}

// clusterResolved returns whether the cluster has been resolved.
func (g *Generator) clusterResolved() bool {
	lowestPriority := g.monorailCfg.Priorities[len(g.monorailCfg.Priorities)-1]
	return !g.cluster.MeetsThreshold(lowestPriority.Threshold)
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

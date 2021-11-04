// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"context"
	"regexp"
	"time"

	"cloud.google.com/go/spanner"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/bugclusters/rules"
	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/config"
	spanutil "infra/appengine/weetbix/internal/span"
)

// RuleIDRe matches validly formed rule IDs.
var RuleIDRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// FailureAssociationRule associates failures with a bug. When the rule
// is used to match incoming test failures, the resultant cluster is
// known as a 'bug cluster' because the cluster is associated with a bug
// (via the failure association rule).
type FailureAssociationRule struct {
	// The LUCI Project for which this rule is defined.
	Project string `json:"project"`
	// The unique identifier for the failure association rule,
	// as 32 lowercase hexadecimal characters.
	RuleID string `json:"ruleId"`
	// The rule predicate, defining which failures are being associated.
	RuleDefinition string `json:"ruleDefinition"`
	// The time the rule was created. Output only.
	CreationTime time.Time `json:"creationTime"`
	// The time the rule was last updated. Output only.
	LastUpdated time.Time `json:"lastUpdated"`
	// Bug is the identifier of the bug that the failures are
	// associated with.
	Bug bugs.BugID `json:"bug"`
	// Whether the bug should be updated by Weetbix, and whether failures
	// should still be matched against the rule.
	IsActive bool `json:"isActive"`
	// The suggested cluster this rule was created from (if any).
	// Until re-clustering is complete and has reduced the residual impact
	// of the source cluster, this cluster ID tells bug filing to ignore
	// the source cluster when determining whether new bugs need to be filed.
	SourceCluster clustering.ClusterID `json:"sourceCluster"`
}

// ReadActive reads all active Weetbix failure association rules in the given LUCI project.
func ReadActive(ctx context.Context, projectID string) ([]*FailureAssociationRule, error) {
	stmt := spanner.NewStatement(`
		SELECT RuleId, RuleDefinition, BugSystem, BugId,
		  CreationTime, LastUpdated,
		  SourceClusterAlgorithm, SourceClusterId
		FROM FailureAssociationRules
		WHERE IsActive AND Project = @projectID
		ORDER BY BugSystem, BugId
	`)
	stmt.Params = map[string]interface{}{
		"projectID": projectID,
	}
	it := span.Query(ctx, stmt)
	bcs := []*FailureAssociationRule{}
	err := it.Do(func(r *spanner.Row) error {
		var ruleID, ruleDefinition, bugSystem, bugID string
		var creationTime, lastUpdated time.Time
		var sourceClusterAlgorithm, sourceClusterID string
		err := r.Columns(
			&ruleID, &ruleDefinition, &bugSystem, &bugID,
			&creationTime, &lastUpdated,
			&sourceClusterAlgorithm, &sourceClusterID,
		)
		if err != nil {
			return errors.Annotate(err, "read rule row").Err()
		}

		bc := &FailureAssociationRule{
			Project:        projectID,
			RuleID:         ruleID,
			RuleDefinition: ruleDefinition,
			CreationTime:   creationTime,
			LastUpdated:    lastUpdated,
			Bug:            bugs.BugID{System: bugSystem, ID: bugID},
			IsActive:       true,
			SourceCluster: clustering.ClusterID{
				Algorithm: sourceClusterAlgorithm,
				ID:        sourceClusterID,
			},
		}
		bcs = append(bcs, bc)
		return nil
	})
	if err != nil {
		return nil, errors.Annotate(err, "query active rules").Err()
	}
	return bcs, nil
}

// Create inserts a new failure association rule with the specified details.
func Create(ctx context.Context, r *FailureAssociationRule) error {
	if err := validateRule(r); err != nil {
		return err
	}
	ms := spanutil.InsertMap("FailureAssociationRules", map[string]interface{}{
		"Project":        r.Project,
		"RuleId":         r.RuleID,
		"RuleDefinition": r.RuleDefinition,
		"CreationTime":   spanner.CommitTimestamp,
		"LastUpdated":    spanner.CommitTimestamp,
		"BugSystem":      r.Bug.System,
		"BugID":          r.Bug.ID,
		// IsActive uses the value 'NULL' to indicate false, and true to indicate true.
		"IsActive":               spanner.NullBool{Bool: r.IsActive, Valid: r.IsActive},
		"SourceClusterAlgorithm": r.SourceCluster.Algorithm,
		"SourceClusterId":        r.SourceCluster.ID,
	})
	span.BufferWrite(ctx, ms)
	return nil
}

func validateRule(r *FailureAssociationRule) error {
	switch {
	case !config.ProjectRe.MatchString(r.Project):
		return errors.New("project must be valid")
	case !RuleIDRe.MatchString(r.RuleID):
		return errors.New("rule ID must be valid")
	case r.Bug.Validate() != nil:
		return errors.Annotate(r.Bug.Validate(), "bug is not valid").Err()
	case r.SourceCluster.Validate() != nil && !r.SourceCluster.IsEmpty():
		return errors.Annotate(r.SourceCluster.Validate(), "source cluster ID is not valid").Err()
	}
	_, err := rules.Parse(r.RuleDefinition, "test", "reason")
	if err != nil {
		return errors.Annotate(err, "rule definition is not valid").Err()
	}
	return nil
}

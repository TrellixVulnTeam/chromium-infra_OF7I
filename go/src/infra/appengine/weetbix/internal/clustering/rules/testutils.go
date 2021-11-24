// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering"
	spanutil "infra/appengine/weetbix/internal/span"
	"infra/appengine/weetbix/internal/testutil"
)

const testProject = "myproject"

// RuleBuilder provides methods to build a failure asociation rule
// for testing.
type RuleBuilder struct {
	uniqifier     int
	project       string
	active        bool
	definition    string
	creationTime  time.Time
	lastUpdated   time.Time
	sourceCluster clustering.ClusterID
}

// NewRule starts building a new Rule.
func NewRule(uniqifier int) *RuleBuilder {
	return &RuleBuilder{
		project:      testProject,
		uniqifier:    uniqifier,
		active:       true,
		definition:   "reason LIKE \"%exit code 5%\" AND test LIKE \"tast.arc.%\"",
		creationTime: time.Date(1900, 1, 2, 3, 4, 5, uniqifier, time.UTC),
		lastUpdated:  time.Date(1900, 1, 2, 3, 4, 5, uniqifier, time.UTC),
		sourceCluster: clustering.ClusterID{
			Algorithm: fmt.Sprintf("clusteralg%v", uniqifier),
			ID:        hex.EncodeToString([]byte(fmt.Sprintf("id%v", uniqifier))),
		},
	}
}

// WithProject specifies the project to use on the rule.
func (b *RuleBuilder) WithProject(project string) *RuleBuilder {
	b.project = project
	return b
}

// WithActive specifies whether the rule will be active.
func (b *RuleBuilder) WithActive(active bool) *RuleBuilder {
	b.active = active
	return b
}

// WithCreationTime specifies the creation time of the rule.
func (b *RuleBuilder) WithCreationTime(value time.Time) *RuleBuilder {
	b.creationTime = value
	return b
}

// WithLastUpdated specifies the LastUpdated time on the rule.
func (b *RuleBuilder) WithLastUpdated(lastUpdated time.Time) *RuleBuilder {
	b.lastUpdated = lastUpdated
	return b
}

// WithRuleDefinition specifies the definition of the rule.
func (b *RuleBuilder) WithRuleDefinition(definition string) *RuleBuilder {
	b.definition = definition
	return b
}

// WithSourceCluster specifies the source suggested cluster that triggered the
// creation of the rule.
func (b *RuleBuilder) WithSourceCluster(value clustering.ClusterID) *RuleBuilder {
	b.sourceCluster = value
	return b
}

func (b *RuleBuilder) Build() *FailureAssociationRule {
	ruleIDBytes := sha256.Sum256([]byte(fmt.Sprintf("rule-id%v", b.uniqifier)))
	return &FailureAssociationRule{
		Project:        b.project,
		RuleID:         hex.EncodeToString(ruleIDBytes[0:16]),
		RuleDefinition: b.definition,
		Bug:            bugs.BugID{System: "monorail", ID: fmt.Sprintf("chromium/%v", b.uniqifier)},
		IsActive:       b.active,
		CreationTime:   b.creationTime,
		LastUpdated:    b.lastUpdated,
		SourceCluster:  b.sourceCluster,
	}
}

// SetRulesForTesting replaces the set of stored rules to match the given set.
func SetRulesForTesting(ctx context.Context, rs []*FailureAssociationRule) error {
	testutil.MustApply(ctx,
		spanner.Delete("FailureAssociationRules", spanner.AllKeys()))
	// Insert some FailureAssociationRules.
	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		for _, r := range rs {
			ms := spanutil.InsertMap("FailureAssociationRules", map[string]interface{}{
				"Project":        r.Project,
				"RuleId":         r.RuleID,
				"RuleDefinition": r.RuleDefinition,
				"CreationTime":   r.CreationTime,
				"LastUpdated":    r.LastUpdated,
				"BugSystem":      r.Bug.System,
				"BugID":          r.Bug.ID,
				// IsActive uses the value 'NULL' to indicate false, and true to indicate true.
				"IsActive":               spanner.NullBool{Bool: r.IsActive, Valid: r.IsActive},
				"SourceClusterAlgorithm": r.SourceCluster.Algorithm,
				"SourceClusterId":        r.SourceCluster.ID,
			})
			span.BufferWrite(ctx, ms)
		}
		return nil
	})
	return err
}

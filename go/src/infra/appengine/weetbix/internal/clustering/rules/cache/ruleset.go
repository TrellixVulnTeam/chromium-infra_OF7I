package cache

import (
	"context"
	"time"

	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/clustering/rules/lang"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/trace"
	"go.chromium.org/luci/server/span"
)

// CachedRule represents a "compiled" version of a failure
// association rule.
// It should be treated as immutable, and is therefore safe to
// share across multiple threads.
type CachedRule struct {
	// The unique identifier for the failure association rule.
	RuleID string
	// The time the rule was last updated.
	LastUpdated time.Time
	// The parsed and compiled failure association rule.
	Expr *lang.Expr
}

// NewCachedRule initialises a new CachedRule from the given failure
// association rule.
func NewCachedRule(rule *rules.FailureAssociationRule) (*CachedRule, error) {
	expr, err := lang.Parse(rule.RuleDefinition, rules.Identifiers...)
	if err != nil {
		return nil, err
	}
	return &CachedRule{
		RuleID:      rule.RuleID,
		LastUpdated: rule.LastUpdated,
		Expr:        expr,
	}, nil
}

// Ruleset represents a version of the set of failure
// association rules in use by a LUCI Project.
// It should be treated as immutable, and therefore safe to share
// across multiple threads.
type Ruleset struct {
	// The LUCI Project.
	Project string
	// Rules is the set of compiled failure association rules.
	Rules []*CachedRule
	// RulesVersion is the Spanner commit timestamp describing
	// the version of the ruleset.
	RulesVersion time.Time
	// LastRefresh is when the ruleset was last refreshed.
	LastRefresh time.Time
}

// NewRuleset initialises a new ruleset. This initial ruleset is invalid
// and must be refreshed before use.
func NewRuleset(project string) *Ruleset {
	return &Ruleset{
		Project: project,
		Rules:   nil,
		// The zero time is not a valid RulesVersion and will be rejected
		// by clustering state validation if we ever try to save it to
		// Spanner.
		RulesVersion: time.Time{},
		LastRefresh:  time.Time{},
	}
}

// refresh updates the ruleset. To ensure existing users of the rulset
// do not observe changes while they are using it, a new copy is returned.
func (r *Ruleset) refresh(ctx context.Context) (ruleset *Ruleset, err error) {
	// Under our design assumption of 10,000 active rules per project,
	// pulling and compiling all rules could take a meaningful amount
	// of time (@ 1KB per rule, = ~10MB).
	ctx, s := trace.StartSpan(ctx, "infra/appengine/weetbix/internal/clustering/rules/cache.Refresh")
	s.Attribute("project", r.Project)
	defer func() { s.End(err) }()

	txn, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()

	var cachedRules []*CachedRule
	if r.RulesVersion == (time.Time{}) {
		// On the first refresh, query all active rules.
		activeRules, err := rules.ReadActive(txn, r.Project)
		if err != nil {
			return nil, err
		}
		cachedRules, err = cachedRulesFromActive(activeRules)
		if err != nil {
			return nil, err
		}
	} else {
		// On subsequent refreshes, query just the differences.
		delta, err := rules.ReadDelta(txn, r.Project, r.RulesVersion)
		if err != nil {
			return nil, err
		}
		cachedRules, err = cachedRulesFromDelta(r.Rules, delta)
		if err != nil {
			return nil, err
		}
	}

	// Get the version of set of rules read by ReadActive/ReadDelta.
	// This is expressed as the Spanner time of the most recent update
	// to the set of rules in the project.
	// Must occur in the same spanner transaction as ReadActive/ReadDelta.
	// If the project has no rules, this returns rules.StartingEpoch.
	lastUpdated, err := rules.ReadLastUpdated(txn, r.Project)
	if err != nil {
		return nil, err
	}

	return &Ruleset{
		Project:      r.Project,
		Rules:        cachedRules,
		RulesVersion: lastUpdated,
		LastRefresh:  clock.Now(ctx),
	}, nil
}

// cachedRulesFromActive obtains a set of cached rules from the given set of
// active failure association rules.
func cachedRulesFromActive(activeRules []*rules.FailureAssociationRule) ([]*CachedRule, error) {
	var result []*CachedRule
	for _, r := range activeRules {
		cr, err := NewCachedRule(r)
		if err != nil {
			return nil, errors.Annotate(err, "rule %s is invalid", r.RuleID).Err()
		}
		result = append(result, cr)
	}
	return result, nil
}

// cachedRulesFromDelta applies deltas to an existing list of rules,
// to obtain an updated set of rules.
func cachedRulesFromDelta(existing []*CachedRule, delta []*rules.FailureAssociationRule) ([]*CachedRule, error) {
	ruleByID := make(map[string]*CachedRule)
	for _, r := range existing {
		ruleByID[r.RuleID] = r
	}
	for _, d := range delta {
		existing, ok := ruleByID[d.RuleID]
		if ok && existing.LastUpdated == d.LastUpdated {
			// Rule was already known and has not changed, so move on.
			// This saves parsing and recompiling rules which are unchanged.
			continue
		}
		if d.IsActive {
			cr, err := NewCachedRule(d)
			if err != nil {
				return nil, errors.Annotate(err, "rule %s is invalid", d.RuleID).Err()
			}
			ruleByID[d.RuleID] = cr
		} else {
			// Delete the rule, if it exists.
			delete(ruleByID, d.RuleID)
		}
	}
	var results []*CachedRule
	for _, r := range ruleByID {
		results = append(results, r)
	}
	return results, nil
}

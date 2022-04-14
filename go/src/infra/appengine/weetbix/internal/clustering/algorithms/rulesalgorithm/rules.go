package rulesalgorithm

import (
	"fmt"
	"time"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/rules/cache"
)

type Algorithm struct{}

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
var AlgorithmName = fmt.Sprintf("%sv%v", clustering.RulesAlgorithmPrefix, AlgorithmVersion)

// Cluster incrementally (re-)clusters the given test failure, updating the
// matched cluster IDs. The passed existingRulesVersion and ruleIDs
// should be the ruleset.RulesVersion and cluster IDs of the previous call
// to Cluster (if any) from which incremental clustering should occur.
//
// If clustering has not been performed previously, and clustering is to be
// performed from scratch, existingRulesVersion should be rules.StartingEpoch
// and ruleIDs should be an empty list.
//
// This method is on the performance-critical path for re-clustering.
//
// To avoid unnecessary allocations, the method will modify the passed ruleIDs.
func (a *Algorithm) Cluster(ruleset *cache.Ruleset, existingRulesVersion time.Time, ruleIDs map[string]struct{}, failure *clustering.Failure) {
	for id := range ruleIDs {
		// Remove matches with rules that are no longer active.
		if !ruleset.IsRuleActive(id) {
			delete(ruleIDs, id)
		}
	}

	// For efficiency, only match new/modified rules since the
	// last call to Cluster(...).
	newRules := ruleset.ActiveRulesWithPredicateUpdatedSince(existingRulesVersion)
	for _, r := range newRules {
		if r.Expr.Evaluate(failure) {
			ruleIDs[r.Rule.RuleID] = struct{}{}
		} else {
			// If this is a modified rule (rather than a new rule)
			// it may have matched previously. Delete any existing
			// match.
			delete(ruleIDs, r.Rule.RuleID)
		}
	}
}

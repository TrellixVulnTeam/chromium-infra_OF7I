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
const AlgorithmVersion = 1

// AlgorithmName is the identifier for the clustering algorithm.
// Weetbix requires all clustering algorithms to have a unique identifier.
// Must match the pattern ^[a-z0-9-.]{1,32}$.
//
// The AlgorithmName must encode the algorithm version, so that each version
// of an algorithm has a different name.
var AlgorithmName = fmt.Sprintf("%sv%v", clustering.RulesAlgorithmPrefix, AlgorithmVersion)

// Cluster incrementally (re-)clusters the given test failure, returning the
// matching cluster IDs. The passed existinRulesVersion and existingIDs
// should be the ruleset.RulesVersion and cluster IDs of the previous call
// to Cluster (if any) from which incremental clustering should occur.
//
// If clustering has not been performed previously, and clustering is to be
// performed from scratch, existingRulesVersion should be rules.StartingEpoch
// and existingIDs should be an empty set.
func (a *Algorithm) Cluster(ruleset *cache.Ruleset, existingRulesVersion time.Time, existingIDs map[string]struct{}, failure *clustering.Failure) map[string]struct{} {
	values := map[string]string{
		"test":   failure.TestID,
		"reason": failure.Reason.GetPrimaryErrorMessage(),
	}

	newIDs := make(map[string]struct{})
	for id := range existingIDs {
		// Keep matches with rules that are still active.
		if ruleset.IsRuleActive(id) {
			newIDs[id] = struct{}{}
		}
	}

	// For efficiency, only match new/modified rules since the
	// last call to Cluster(...).
	newRules := ruleset.ActiveRulesUpdatedSince(existingRulesVersion)
	for _, r := range newRules {
		if r.Expr.Evaluate(values) {
			newIDs[r.RuleID] = struct{}{}
		} else {
			// If this is a modified rule (rather than a new rule)
			// it may have matched previously. Delete any existing
			// match.
			delete(newIDs, r.RuleID)
		}
	}
	return newIDs
}

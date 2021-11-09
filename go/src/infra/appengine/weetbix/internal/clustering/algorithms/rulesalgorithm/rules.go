package rulesalgorithm

import (
	"fmt"

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

// Cluster clusters the given test failure and returns its cluster IDs.
func (a *Algorithm) Cluster(ruleset *cache.Ruleset, failure *clustering.Failure) []*clustering.ClusterID {
	values := map[string]string{
		"test":   failure.TestID,
		"reason": failure.Reason.GetPrimaryErrorMessage(),
	}

	var clusters []*clustering.ClusterID
	for _, r := range ruleset.Rules {
		if r.Expr.Evaluate(values) {
			id := &clustering.ClusterID{
				Algorithm: AlgorithmName,
				ID:        r.RuleID,
			}
			clusters = append(clusters, id)
		}
	}
	return clusters
}

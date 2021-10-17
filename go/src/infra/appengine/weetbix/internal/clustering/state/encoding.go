// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package state

import (
	"fmt"

	"infra/appengine/weetbix/internal/clustering"
	cpb "infra/appengine/weetbix/internal/clustering/proto"

	"go.chromium.org/luci/common/errors"
)

// decodeClusters decodes the clusters assigned to each test result from the protobuf representation.
func decodeClusters(cc *cpb.ChunkClusters) ([][]*clustering.ClusterID, error) {
	if cc == nil {
		return nil, errors.New("proto must be specified")
	}
	typeCount := int64(len(cc.ClusterTypes))
	clusterCount := int64(len(cc.ReferencedClusters))

	results := make([][]*clustering.ClusterID, len(cc.ResultClusters))
	for i, rc := range cc.ResultClusters {
		clusters := make([]*clustering.ClusterID, len(rc.ClusterRefs))
		for j, ref := range rc.ClusterRefs {
			if ref < 0 || ref >= clusterCount {
				return nil, fmt.Errorf("reference to non-existent cluster (%v) from result %v; only %v referenced clusters defined", ref, i, clusterCount)
			}
			cluster := cc.ReferencedClusters[ref]
			if cluster.TypeRef < 0 || cluster.TypeRef >= typeCount {
				return nil, fmt.Errorf("reference to non-existent type (%v) from referenced cluster %v; only %v types defined", cluster.TypeRef, ref, typeCount)
			}
			t := cc.ClusterTypes[cluster.TypeRef]
			clusters[j] = &clustering.ClusterID{
				Algorithm: t.Algorithm,
				ID:        cluster.ClusterId,
			}
		}
		results[i] = clusters
	}
	return results, nil
}

// encodeClusters encodes the clusters assigned to each test result to the protobuf representation.
func encodeClusters(clusterRefs [][]*clustering.ClusterID) *cpb.ChunkClusters {
	rb := newRefBuilder()
	resultClusters := make([]*cpb.TestResultClusters, len(clusterRefs))
	for i, refs := range clusterRefs {
		clusters := &cpb.TestResultClusters{}
		clusters.ClusterRefs = make([]int64, len(refs))
		for j, r := range refs {
			clusters.ClusterRefs[j] = rb.ReferenceCluster(r)
		}
		resultClusters[i] = clusters
	}
	result := &cpb.ChunkClusters{
		ClusterTypes:       rb.types,
		ReferencedClusters: rb.refs,
		ResultClusters:     resultClusters,
	}
	return result
}

// refBuilder assists in constructing the type and cluster references used in
// the proto representation.
type refBuilder struct {
	types []*cpb.ClusterType
	// typeMap is a mapping from algorithm name to the index in types.
	typeMap map[string]int
	refs    []*cpb.ReferencedCluster
	// refMap is a mapping from (algorithm name, cluster ID) to the
	// the corresponding cluster reference in refs.
	refMap map[string]int
}

func newRefBuilder() *refBuilder {
	return &refBuilder{
		typeMap: make(map[string]int),
		refMap:  make(map[string]int),
	}
}

func (rb *refBuilder) ReferenceCluster(ref *clustering.ClusterID) int64 {
	refKey := ref.Key()
	idx, ok := rb.refMap[refKey]
	if !ok {
		ref := &cpb.ReferencedCluster{
			TypeRef:   rb.ReferenceClusterType(ref.Algorithm),
			ClusterId: ref.ID,
		}
		idx = len(rb.refs)
		rb.refMap[refKey] = idx
		rb.refs = append(rb.refs, ref)
	}
	return int64(idx)
}

func (rb *refBuilder) ReferenceClusterType(algorithm string) int64 {
	idx, ok := rb.typeMap[algorithm]
	if !ok {
		// Cluster type does not exist.
		idx = len(rb.types)
		rb.typeMap[algorithm] = idx
		rb.types = append(rb.types, &cpb.ClusterType{Algorithm: algorithm})
	}
	return int64(idx)
}

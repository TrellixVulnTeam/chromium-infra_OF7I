// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"path"
	"sort"

	"google.golang.org/protobuf/proto"

	dirmdpb "infra/tools/dirmd/proto"
)

// Mapping is a mapping from a directory to its metadata.
//
// It wraps the corresponding protobuf message and adds utility functions.
type Mapping dirmdpb.Mapping

// NewMapping initializes an empty mapping.
func NewMapping(size int) *Mapping {
	return &Mapping{
		Dirs: make(map[string]*dirmdpb.Metadata, size),
	}
}

// Compute computes metadata for the given directory key.
func (m *Mapping) Compute(key string) *dirmdpb.Metadata {
	parent := path.Dir(key)
	if parent == key {
		return cloneMD(m.Dirs[key])
	}

	ret := m.Compute(parent)
	Merge(ret, m.Dirs[key])
	return ret
}

// Proto converts m back to the protobuf message.
func (m *Mapping) Proto() *dirmdpb.Mapping {
	return (*dirmdpb.Mapping)(m)
}

// Clone returns a deep copy of m.
func (m *Mapping) Clone() *Mapping {
	return (*Mapping)(proto.Clone(m.Proto()).(*dirmdpb.Mapping))
}

// ComputeAll computes full metadata for each dir.
func (m *Mapping) ComputeAll() {
	// Process directories in the shorest-path to longest-path order,
	// such that, when computing the expanded metadata for a given directory,
	// we only need to check the nearest ancestor.
	for _, dir := range m.keysByLength() {
		meta := cloneMD(m.nearestAncestor(dir))
		Merge(meta, m.Dirs[dir])
		m.Dirs[dir] = meta
	}
}

// nearestAncestor returns metadata of the nearest ancestor.
func (m *Mapping) nearestAncestor(dir string) *dirmdpb.Metadata {
	for {
		parent := path.Dir(dir)
		if parent == dir {
			// We have reached the root.
			return nil
		}
		dir = parent

		if meta, ok := m.Dirs[dir]; ok {
			return meta
		}
	}
}

// keysByLength returns keys sorted by length.
// Key "." is treated as shortest of all.
func (m *Mapping) keysByLength() []string {
	ret := make([]string, 0, len(m.Dirs))
	for k := range m.Dirs {
		ret = append(ret, k)
	}

	sortKey := func(dirKey string) int {
		// "." is considered shortest of all.
		if dirKey == "." {
			return -1
		}
		return len(dirKey)
	}
	sort.Slice(ret, func(i, j int) bool {
		return sortKey(ret[i]) < sortKey(ret[j])
	})
	return ret
}

// Merge merges metadata from src to dst, where dst is metadata inherited from
// ancestors and src contains directory-specific metadata.
// Does nothing is src is nil.
//
// The current implementation is just proto.Merge, but it may change in the
// future.
func Merge(dst, src *dirmdpb.Metadata) {
	if src != nil {
		proto.Merge(dst, src)
	}
}

// cloneMD returns a deep copy of meta.
// If md is nil, returns a new message.
func cloneMD(md *dirmdpb.Metadata) *dirmdpb.Metadata {
	if md == nil {
		return &dirmdpb.Metadata{}
	}
	return proto.Clone(md).(*dirmdpb.Metadata)
}

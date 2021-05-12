// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"path"
	"sort"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	dirmdpb "infra/tools/dirmd/proto"
)

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

// ComputeAll computes full metadata for each dir.
func (m *Mapping) ComputeAll() {
	// Process directories in the shorest-path to longest-path order,
	// such that, when computing the expanded metadata for a given directory,
	// we only need to check the nearest ancestor.
	for _, dir := range m.keysByLength(true) {
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
func (m *Mapping) keysByLength(asc bool) []string {
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
	if asc {
		sort.Slice(ret, func(i, j int) bool {
			return sortKey(ret[i]) < sortKey(ret[j])
		})
	} else {
		sort.Slice(ret, func(i, j int) bool {
			return sortKey(ret[i]) > sortKey(ret[j])
		})
	}
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

// Reduce removes all redundant information.
func (m *Mapping) Reduce() {
	// First, compute metadata for each node.
	m.ComputeAll()

	// Then, remove nodes that do not add any new info wrt their nearest ancestor.
	// Process directories in the longest-path to shortest-path order,
	// such that, when computing the expanded metadata for a given directory,
	// we only need to check the nearest ancestor.
	// The shortest-to-longest order doesn't work because we need a complete ancestor
	// to decide which parts of the descendant are redundant, so remove data in
	// the bottom-to-top order.
	for _, dir := range m.keysByLength(false) {
		meta := m.Dirs[dir]
		if ancestor := m.nearestAncestor(dir); ancestor != nil {
			excludeSame(meta.ProtoReflect(), ancestor.ProtoReflect())
		}
		if isEmpty(meta.ProtoReflect()) {
			delete(m.Dirs, dir)
		}
	}
}

// excludeSame mutates m in-place to clear fields that have same values as ones
// in exclude.
func excludeSame(m, exclude protoreflect.Message) {
	m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case !exclude.Has(f):
			// It cannot be the same.
			return true

		case f.Cardinality() == protoreflect.Repeated:
			// TODO(crbug.com/1103287) handle exclude same elements from repeated fields.

		case f.Kind() == protoreflect.MessageKind:
			// Recurse.
			excludeSame(v.Message(), exclude.Get(f).Message())
			// Clear the field if it became empty.
			if isEmpty(v.Message()) {
				m.Clear(f)
			}

		case scalarValuesEqual(v, exclude.Get(f), f.Kind()):
			m.Clear(f)
		}
		return true
	})
}

// scalarValuesEqual returns true if a and b are determined to be equal.
// May return false negatives.
func scalarValuesEqual(a, b protoreflect.Value, kind protoreflect.Kind) bool {
	switch kind {
	case protoreflect.BoolKind:
		return a.Bool() == b.Bool()
	case protoreflect.EnumKind:
		return a.Enum() == b.Enum()
	case protoreflect.Int32Kind, protoreflect.Int64Kind:
		return a.Int() == b.Int()
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return a.Float() == b.Float()
	case protoreflect.StringKind:
		return a.String() == b.String()
	default:
		return false
	}
}

// isEmpty returns true if m has no populated fields.
func isEmpty(m protoreflect.Message) bool {
	found := false
	m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		found = true
		return false
	})
	return !found
}

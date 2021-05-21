// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"path"
	"sort"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"go.chromium.org/luci/common/errors"

	dirmdpb "infra/tools/dirmd/proto"
)

// Compute computes metadata for the given directory key.
func (m *Mapping) Compute(key string) (*dirmdpb.Metadata, error) {
	parent := path.Dir(key)
	if parent == key {
		return cloneMD(m.Dirs[key]), nil
	}

	ret, err := m.Compute(parent)
	if err != nil {
		return nil, err
	}
	return ret, m.apply(ret, key)
}

// ComputeAll computes full metadata for each dir.
func (m *Mapping) ComputeAll() error {
	// Process directories in the shorest-path to longest-path order,
	// such that, when computing the expanded metadata for a given directory,
	// we only need to check the nearest ancestor.
	for _, dir := range m.keysByLength() {
		meta := cloneMD(m.nearestAncestor(dir))
		if err := m.apply(meta, dir); err != nil {
			return errors.Annotate(err, "dir %q", dir).Err()
		}
		m.Dirs[dir] = meta
	}
	return nil
}

// Reduce removes all redundant information.
func (m *Mapping) Reduce() error {
	// This function implemenation is similar to ComputeAll()'s implementation,
	// but the redundant info is determined after mixins are applied and before
	// metadata of the current node is added.
	// The function maintains two mappings: computed and reduced, and both are
	// being built simultaneously, top-bottom.
	// In the end, m contains the reduced mapping.

	// Process directories in the shorest-path to longest-path order,
	// such that, when computing the expanded metadata for a given directory,
	// we only need to check the nearest ancestor.
	computed := NewMapping(len(m.Dirs))
	for _, dir := range m.keysByLength() {
		md := m.Dirs[dir]

		// First, inherit from the ancestor and apply mixins.
		mdComputed := cloneMD(computed.nearestAncestor(dir))
		if err := m.applyMixins(mdComputed, md, dir); err != nil {
			return err
		}

		// At this point mdComputed contains the inherited and mixed-in metadata.
		// If md and mdComputed have the same value for an attribute, then it is
		// redundant. Remove such metadata from md.
		excludeSame(md.ProtoReflect(), mdComputed.ProtoReflect())
		if isEmpty(md.ProtoReflect()) {
			delete(m.Dirs, dir)
		}

		// Apply the non-redundant metadata to mdComputed, so that
		// descendants of this dir can pick it up in the next iterations of the loop.
		Merge(mdComputed, md)
		mdComputed.Mixins = nil
		computed.Dirs[dir] = mdComputed
	}
	return nil
}

// apply updates dst with the metadata for the dir key.
// The applied metadata includes mixins.
// dst.Mixins are cleared.
func (m *Mapping) apply(dst *dirmdpb.Metadata, dirKey string) error {
	src := m.Dirs[dirKey]

	// First apply mixins.
	if err := m.applyMixins(dst, src, dirKey); err != nil {
		return err
	}

	// Then apply the metadata of this directory.
	Merge(dst, src)

	// Clear the mixin list after applying, to avoid accidental double importing.
	// Do it only after merging src, otherwise it would be re-populated.
	dst.Mixins = nil
	return nil
}

// applyMixins applies the src.Mixins to dst.
func (m *Mapping) applyMixins(dst, src *dirmdpb.Metadata, srcDirKey string) error {
	if len(src.GetMixins()) > 0 {
		repo := m.repoFor(srcDirKey)
		if repo == nil {
			return errors.Reason("repo entry not found").Err()
		}
		for _, im := range src.Mixins {
			imMD := repo.Mixins[im]
			if imMD == nil {
				return errors.Reason("mixin %q not found", im).Err()
			}
			Merge(dst, imMD)
		}
	}
	return nil
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

// repoFor returns the Repo for the directory.
// Returns nil if the repo is not found.
func (m *Mapping) repoFor(dir string) *dirmdpb.Repo {
	for {
		if repo, ok := m.Repos[dir]; ok {
			return repo
		}

		parent := path.Dir(dir)
		if parent == dir {
			// We have reached the root.
			return nil
		}
		dir = parent
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

// Merge merges metadata from src to dst. Does nothing if src is nil.
//
// The current implementation is just proto.Merge, but it may change in the
// future.
func Merge(dst, src *dirmdpb.Metadata) {
	if src != nil {
		proto.Merge(dst, src)
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

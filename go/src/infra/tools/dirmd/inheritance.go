// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"path"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"go.chromium.org/luci/common/errors"

	dirmdpb "infra/tools/dirmd/proto"
)

// NoInheritance is a valid value of Metadata.InheritFrom which means metadata
// must not be inherited from anywhere.
// This is equivalent to `set noparent` in OWNERS files.
const NoInheritance = "-"

// ComputeAll computes full metadata for each dir.
//
// If an inheritance cycle is detected, returns an error.
func (m *Mapping) ComputeAll() error {
	return m.computeAll(false)
}

// Reduce removes all redundant information.
// For example, if a child inherits its metadata from its parent, and both
// define the same attribute value, then it is removed from the child.
//
// If an inheritance cycle is detected, returns an error.
func (m *Mapping) Reduce() error {
	return m.computeAll(true)
}

func (m *Mapping) computeAll(reduce bool) error {
	// Mapping from directory names to fully-computed Metadata messages.
	// Acts as a cache, and becomes m.Dirs in the end.
	computed := make(map[string]*dirmdpb.Metadata, len(m.Dirs))

	type edge struct {
		dir                 string
		md                  *dirmdpb.Metadata // a value in `computed` map
		inheritFromComputed *dirmdpb.Metadata // computed metadata of the dir md inherieted from
	}
	// topologicalOrder is used for reduction.
	topologicalOrder := make([]edge, 0, len(m.Dirs))

	// A set of metadata messages currently in the inheritance chain, used for
	// cycle detection.
	// Use *Metadata as the map key because comparing pointers is much faster than
	// strings, especially when strings are long directory paths.
	currentChain := make(map[*dirmdpb.Metadata]struct{})

	// compute returns computed metadata for the dir.
	var compute func(dir string, md *dirmdpb.Metadata) (computedMD *dirmdpb.Metadata, err error)
	compute = func(dir string, md *dirmdpb.Metadata) (mdComputed *dirmdpb.Metadata, err error) {
		// Do not compute twice.
		if ret, ok := computed[dir]; ok {
			return ret, nil
		}
		defer func() {
			if mdComputed != nil {
				computed[dir] = mdComputed
			}
		}()

		// Check for cycles.
		if md == nil {
			panic("nil metadata")
		}
		if _, ok := currentChain[md]; ok {
			return nil, errors.Reason("inheritance cycle with dir %q is detected", dir).Err()
		}
		currentChain[md] = struct{}{}
		defer delete(currentChain, md)

		// Retrieve the original (not computed) metadata to inherit from.
		inheritFrom, inheritFromMD, err := m.findInheritFrom(dir, md)
		switch {
		case err != nil:
			return nil, err

		case inheritFromMD == nil:
			// Nothing to inherit. Return the original metadata.
			return md, nil
		}

		// Compute the full metadata.
		inheritFromComputed, err := compute(inheritFrom, inheritFromMD)
		if err != nil {
			return nil, err
		}

		mdComputed = cloneMD(inheritFromComputed)
		merge(mdComputed, md)
		// Force md's inheritFrom, especially if it is empty.
		mdComputed.InheritFrom = md.GetInheritFrom()
		topologicalOrder = append(topologicalOrder, edge{dir: dir, md: mdComputed, inheritFromComputed: inheritFromComputed})
		return mdComputed, nil
	}

	for dir, md := range m.Dirs {
		if _, err := compute(dir, md); err != nil {
			return err
		}
	}

	if reduce {
		// Drop redundant entries.
		// Do it only after metadata for all dirs is computed.
		// Reduce in the reversed topological order because this loop mutates
		// metadata, i.e. start with leafs and finish with the root.
		for i := len(topologicalOrder) - 1; i >= 0; i-- {
			e := topologicalOrder[i]
			excludeSame(e.md.ProtoReflect(), e.inheritFromComputed.ProtoReflect())
			if isEmpty(e.md.ProtoReflect()) {
				delete(computed, e.dir)
			}
		}
	}

	m.Dirs = computed
	return nil
}

// findInheritFrom returns the dir to inherit metadata from.
// May return ("", nil, nil) which means nothing to inherit from.
func (m *Mapping) findInheritFrom(dir string, md *dirmdpb.Metadata) (inheritFrom string, inheritFromMD *dirmdpb.Metadata, err error) {
	switch {
	case md.GetInheritFrom() == "":
		// Inherit from the parent.
		if dir == "." {
			// But this is root!
			return "", nil, nil
		}
		inheritFrom = path.Dir(dir)

	case md.InheritFrom == NoInheritance:
		return "", nil, nil

	case strings.HasPrefix(md.InheritFrom, "//"):
		inheritFrom = strings.TrimPrefix(md.InheritFrom, "//")

	default:
		return "", nil, errors.Reason("unexpected inherit_from value %q in dir %q", md.InheritFrom, dir).Err()
	}

	// Walk ancestors because `inheritFrom` might not define its own metadata and
	// the default inheritance is from the parent.
	for {
		if md, ok := m.Dirs[inheritFrom]; ok {
			return inheritFrom, md, nil
		}

		if inheritFrom == "." {
			// We have reached the root - cannot go higher.
			return "", nil, nil
		}
		inheritFrom = path.Dir(inheritFrom)
	}
}

// merge merges metadata from src to dst, where dst is metadata inherited from
// ancestors and src contains directory-specific metadata.
// Does nothing is src is nil.
//
// The current implementation is just proto.merge, but it may change in the
// future.
func merge(dst, src *dirmdpb.Metadata) {
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

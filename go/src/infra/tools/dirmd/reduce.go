// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Reduce removes all redundant information.
func (m *Mapping) Reduce() {
	// First, compute metadata for each node.
	m.ComputeAll()

	// Then, remove nodes that do not add any new info wrt their nearest ancestor.
	// Process directories in the shorest-path to longest-path order,
	// such that, when computing the expanded metadata for a given directory,
	// we only need to check the nearest ancestor.
	for _, dir := range m.keysByLength() {
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

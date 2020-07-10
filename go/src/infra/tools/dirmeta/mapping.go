// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmeta

import (
	dirmetapb "infra/tools/dirmeta/proto"
)

// Mapping is a mapping from a directory to its metadata.
//
// It wraps the corresponding protobuf message and adds utility functions.
type Mapping dirmetapb.Mapping

// Proto converts m back to the protobuf message.
func (m *Mapping) Proto() *dirmetapb.Mapping {
	return (*dirmetapb.Mapping)(m)
}

// Reduce returns a new mapping with all redundancies removed.
func (m *Mapping) Reduce() *Mapping {
	panic("not implemented")
}

// Expand returns a new mapping where each dir has attributes inherited
// from the parent dir.
func (m *Mapping) Expand() *Mapping {
	panic("not implemented")
}

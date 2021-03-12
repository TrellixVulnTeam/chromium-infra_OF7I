// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
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

// Proto converts m back to the protobuf message.
func (m *Mapping) Proto() *dirmdpb.Mapping {
	return (*dirmdpb.Mapping)(m)
}

// Clone returns a deep copy of m.
func (m *Mapping) Clone() *Mapping {
	return (*Mapping)(proto.Clone(m.Proto()).(*dirmdpb.Mapping))
}

// cloneMD returns a deep copy of meta.
// If md is nil, returns a new message.
func cloneMD(md *dirmdpb.Metadata) *dirmdpb.Metadata {
	if md == nil {
		return &dirmdpb.Metadata{}
	}
	return proto.Clone(md).(*dirmdpb.Metadata)
}

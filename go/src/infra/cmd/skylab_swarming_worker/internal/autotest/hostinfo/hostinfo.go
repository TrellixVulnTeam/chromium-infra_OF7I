// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package hostinfo provides support for Autotest's serialized
// hostinfo data.
package hostinfo

import (
	"encoding/json"
)

// HostInfo stores the host information.  Hostinfo files are used to
// pass host information to Autotest and receive host information
// changes from Autotest.
type HostInfo struct {
	Labels         []string          `json:"labels"`
	Attributes     map[string]string `json:"attributes"`
	StableVersions map[string]string `json:"stable_versions"`
}

// versionedHostInfo is only used for backwards-compatibility when
// writing hostinfo information
type versionedHostInfo struct {
	*HostInfo
	SerializerVersion int `json:"serializer_version"`
}

// currentSerializerVersion is emitted for backwards compatibility only
const currentSerializerVersion = 1

// Unmarshal deserializes a HostInfo struct from a slice of bytes.
// Unmarshal accepts a serialized HostInfo or serialized versionedHostInfo
// it always ignores all version information
func Unmarshal(blob []byte) (*HostInfo, error) {
	var hi HostInfo
	err := json.Unmarshal(blob, &hi)
	if err == nil {
		return &hi, nil
	}
	return nil, err
}

// Marshal serializes the HostInfo struct into a slice of bytes.
func Marshal(hi *HostInfo) ([]byte, error) {
	ensureDefaultFieldValues(hi)
	vhi := versionedHostInfo{
		HostInfo:          hi,
		SerializerVersion: currentSerializerVersion,
	}
	return json.Marshal(vhi)
}

// ensureDefaultFieldValues -- ensure that maps and slices in HostInfo are non-nil
func ensureDefaultFieldValues(hi *HostInfo) {
	if hi.Labels == nil {
		hi.Labels = []string{}
	}
	if hi.Attributes == nil {
		hi.Attributes = make(map[string]string)
	}
	if hi.StableVersions == nil {
		hi.StableVersions = make(map[string]string)
	}
}

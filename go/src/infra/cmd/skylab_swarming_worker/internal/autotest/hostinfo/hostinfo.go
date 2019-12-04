// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package hostinfo provides support for Autotest's serialized
// hostinfo data.
package hostinfo

import (
	"encoding/json"
	"log"
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
	log.Printf("hostinfo::Marshal: before default values applied (%#v)", hi)
	ensureDefaultFieldValues(hi)
	log.Printf("hostinfo::Marshal: after default values applied (%#v)", hi)
	vhi := versionedHostInfo{
		HostInfo:          hi,
		SerializerVersion: currentSerializerVersion,
	}
	out, err := json.Marshal(vhi)
	log.Printf("hostinfo::Marshal: marshalled hostinfo (%s) err (%s)", string(out), err)
	return out, err
}

// MarshalIndent serializes the HostInfo struct into a slice of bytes.
// The prefix and indent options of json.MarshalIndent are not exposed to the user because You Don't Need Them.
func MarshalIndent(hi *HostInfo) ([]byte, error) {
	log.Printf("hostinfo::MarshalIndent: before default values applied (%#v)", hi)
	ensureDefaultFieldValues(hi)
	log.Printf("hostinfo::MarshalIndent: after default values applied (%#v)", hi)
	vhi := versionedHostInfo{
		HostInfo:          hi,
		SerializerVersion: currentSerializerVersion,
	}
	out, err := json.MarshalIndent(vhi, "", "\t")
	log.Printf("hostinfo::MarshalIndent: marshalled hostinfo (%s) err (%s)", string(out), err)
	return out, err
}

// ensureDefaultFieldValues -- ensure that maps and slices in HostInfo are non-nil
func ensureDefaultFieldValues(hi *HostInfo) {
	if hi.Labels == nil {
		log.Printf("ensureDefaultFieldValues: empty Labels")
		hi.Labels = []string{}
	}
	if hi.Attributes == nil {
		log.Printf("ensureDefaultFieldValues: empty Attributes")
		hi.Attributes = make(map[string]string)
	}
	if hi.StableVersions == nil {
		log.Printf("ensureDefaultFieldValues: empty StableVersions")
		hi.StableVersions = make(map[string]string)
	}
}

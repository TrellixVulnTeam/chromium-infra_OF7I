// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"fmt"
	"sort"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	proto "github.com/golang/protobuf/proto"

	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
)

// AddUpdatedCros add and update the new cros stable version to old.
func AddUpdatedCros(old, updated []*sv.StableCrosVersion) []*sv.StableCrosVersion {
	oldM := make(map[string]*sv.StableCrosVersion, len(old))
	for _, osv := range old {
		oldM[crosSVKey(osv)] = osv
	}

	for _, u := range updated {
		k := crosSVKey(u)
		osv, ok := oldM[k]
		if ok {
			osv.Version = u.GetVersion()
		} else {
			old = append(old, u)
		}
	}
	return old
}

// AddUpdatedFirmware add and update the new firmware stable version to old.
func AddUpdatedFirmware(old, updated []*sv.StableFirmwareVersion) []*sv.StableFirmwareVersion {
	oldM := make(map[string]*sv.StableFirmwareVersion, len(old))
	for _, osv := range old {
		oldM[firmwareSVKey(osv)] = osv
	}

	for _, u := range updated {
		k := firmwareSVKey(u)
		osv, ok := oldM[k]
		if ok {
			osv.Version = u.GetVersion()
		} else {
			old = append(old, u)
		}
	}
	return old
}

func crosSVKey(c *sv.StableCrosVersion) string {
	return c.GetKey().GetBuildTarget().GetName()
}

func firmwareSVKey(f *sv.StableFirmwareVersion) string {
	return fmt.Sprintf("%s:%s", f.GetKey().GetBuildTarget().GetName(), f.GetKey().GetModelId().GetValue())
}

func faftSVKey(f *sv.StableFaftVersion) string {
	return fmt.Sprintf("%s:%s", f.GetKey().GetBuildTarget().GetName(), f.GetKey().GetModelId().GetValue())
}

// WriteSVToString marshals stable version information into a string.
func WriteSVToString(s *sv.StableVersions) (string, error) {
	all := proto.Clone(s).(*sv.StableVersions)
	sortSV(all)
	return (&jsonpb.Marshaler{Indent: "\t"}).MarshalToString(all)
}

func sortSV(s *sv.StableVersions) {
	if s == nil {
		return
	}

	c := s.Cros
	sort.SliceStable(c, func(i, j int) bool {
		return strings.ToLower(crosSVKey(c[i])) < strings.ToLower(crosSVKey(c[j]))
	})
	fi := s.Firmware
	sort.SliceStable(fi, func(i, j int) bool {
		return strings.ToLower(firmwareSVKey(fi[i])) < strings.ToLower(firmwareSVKey(fi[j]))
	})
	faft := s.Faft
	sort.SliceStable(faft, func(i, j int) bool {
		return strings.ToLower(faftSVKey(faft[i])) < strings.ToLower(faftSVKey(faft[j]))
	})
}

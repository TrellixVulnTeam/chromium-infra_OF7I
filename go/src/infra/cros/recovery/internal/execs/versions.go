// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/tlw"
)

// Local implementation of Versioner interface.
type versioner struct {
	a tlw.Access
}

// Versioner returns versioner to read any version.
func (ei *ExecInfo) Versioner() components.Versioner {
	return &versioner{
		a: ei.RunArgs.Access,
	}
}

// Cros return version info for request Chrome OS device.
func (v *versioner) Cros(ctx context.Context, resource string) (*components.CrosVersionInfo, error) {
	r, err := v.a.Version(ctx, &tlw.VersionRequest{
		Resource: resource,
		Type:     tlw.VersionRequest_CROS,
	})
	if err != nil {
		return nil, errors.Annotate(err, "cros version").Err()
	}
	if len(r.GetValue()) < 1 {
		return nil, errors.Reason("cros version: no version received").Err()
	}
	res := &components.CrosVersionInfo{}
	if v, ok := r.GetValue()["os_image"]; ok {
		res.OSImage = v
	}
	if v, ok := r.GetValue()["fw_image"]; ok {
		res.FwImage = v
	}
	if v, ok := r.GetValue()["fw_version"]; ok {
		res.FwVersion = v
	}
	return res, nil
}

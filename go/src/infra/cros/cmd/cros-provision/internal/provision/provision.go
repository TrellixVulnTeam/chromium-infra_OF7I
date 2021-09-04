// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package provision run provisioning for DUT.
package provision

import (
	"context"
	"log"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/common/errors"
)

// Result holds result data for provisioning the DUT.
type Result struct {
	Out *api.DutOutput
	Err error
}

// Run runs provisioning software dependencies per DUT.
func Run(ctx context.Context, in *api.DutInput, localAddr string) *Result {
	res := &Result{
		Out: &api.DutOutput{
			Id: in.GetId(),
		},
	}
	if in == nil || in.GetProvisionState() == nil {
		res.Err = errors.Reason("run provision: DUT input is empty").Err()
		return res
	}
	if localAddr == "" {
		res.Err = errors.Reason("run provision: local address is not provided").Err()
		return res
	}
	log.Printf("Starting provisioning of %q", in.GetId().GetValue())
	return res
}

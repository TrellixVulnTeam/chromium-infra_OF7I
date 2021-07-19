// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package loader provides functionality to load configuration and verify it.
package loader

import (
	"context"
	"encoding/json"
	"io"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/planpb"
)

// TODO(otabek@): Add data validation for loaded config.
// 1) Looping actions

// LoadConfiguration performs loading the configuration source with data validation.
func LoadConfiguration(ctx context.Context, r io.Reader) (*planpb.Configuration, error) {
	log.Debug(ctx, "Load configuration: started.")
	if r == nil {
		return nil, errors.Reason("load configuration: reader is not provided").Err()
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Annotate(err, "load configuration").Err()
	}
	if len(data) == 0 {
		return nil, errors.Reason("load configuration: configuration is empty").Err()
	}
	config := planpb.Configuration{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.Annotate(err, "load configuration").Err()
	}
	for _, p := range config.GetPlans() {
		createMissingActions(p, p.GetCriticalActions())
		for _, a := range p.GetActions() {
			createMissingActions(p, a.GetConditions())
			createMissingActions(p, a.GetDependencies())
			createMissingActions(p, a.GetRecoveryActions())
		}
		if err := setAndVerifyExecs(p); err != nil {
			return nil, errors.Annotate(err, "load configuration").Err()
		}
	}
	log.Debug(ctx, "Load configuration: finished successfully.")
	return &config, nil
}

// createMissingActions creates missing actions to the plan.
func createMissingActions(p *planpb.Plan, actions []string) {
	for _, a := range actions {
		if _, ok := p.GetActions()[a]; !ok {
			p.GetActions()[a] = &planpb.Action{}
		}
	}
}

// execsExist is link to the function to check if exec function is present.
// Link created to create ability to override for local testing.
var execsExist = execs.Exist

// setAndVerifyExecs sets exec-name if missing and validate whether exec is present
// in recovery-lib.
func setAndVerifyExecs(p *planpb.Plan) error {
	for an, a := range p.GetActions() {
		if a.GetExecName() == "" {
			a.ExecName = an
		}
		if !execsExist(a.GetExecName()) {
			return errors.Reason("exec %q is not exist", a.GetExecName()).Err()
		}
	}
	return nil
}

// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package plan provides struts and functionality to use plans and actions.
package plan

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/plan/execs"
)

// Plan describes a recovery plan, including the verification actions used to determine success.
type Plan struct {
	// Unique name.
	// Can be used only predefined names of plans.
	Name string
	// List of verifiers used to determine success.
	Verifiers []*Action
	// If set true then the plan is allowed to fail without affecting the final result.
	AllowFail bool
}

// Error to request run start-over all verifiers.
// Requested side has to clean cache by itself before request start-over.
var startOver = errors.Reason("request to start over").Err()

// Run runs the recovery plan.
func (p *Plan) Run(ctx context.Context, args *execs.RunArgs) error {
	log.Info(ctx, "Plan %q: started", p.Name)
	log.Debug(ctx, "\n%s", p.Describe())
	c := newCache()
	defer c.close()
	for {
		if err := p.runVerifiers(ctx, c, args); err != nil {
			if err == startOver {
				log.Info(ctx, "Plan %q: received request to start over.", p.Name)
				// Reset cache for all verifiers and dependencies.
				for _, v := range p.Verifiers {
					c.resetForAction(ctx, v)
				}
				continue
			}
			return errors.Annotate(err, "run plan %q", p.Name).Err()
		}
		break
	}
	log.Info(ctx, "Plan %q: finished successfully.", p.Name)
	return nil
}

// runVerifiers runs verifiers of the plan.
// Method the first check the result of verifier from cache and if not exist then perform the verifier.
func (p *Plan) runVerifiers(ctx context.Context, c *runCache, args *execs.RunArgs) error {
	for i, v := range p.Verifiers {
		if err, ok := c.getActionError(v); ok {
			if err == nil {
				log.Info(ctx, "Verifier %q: pass (cached).", v.Name)
				continue
			} else if v.AllowFail {
				log.Info(ctx, "Verifier %q: fail (cached). Error: %s", v.Name, err)
				v.logAllowedFailFault(ctx, i, len(p.Verifiers))
				continue
			}
			return errors.Annotate(err, "run verifier %q: fail (cached)", v.Name).Err()
		}
		if err := v.run(ctx, c, args); err != nil {
			if err == startOver {
				log.Info(ctx, "Verifier %q: received request to start over.", v.Name)
				return err
			}
			if v.AllowFail {
				log.Info(ctx, "Verifier %q: fail. Error: %s", v.Name, err)
				v.logAllowedFailFault(ctx, i, len(p.Verifiers))
			} else {
				return errors.Annotate(err, "run verifier %q", v.Name).Err()
			}
		} else {
			log.Info(ctx, "Verifier %q: finished successfully.", v.Name)
		}
	}
	return nil
}

// Describe describes the plan details with verifiers.
func (p *Plan) Describe() string {
	r := fmt.Sprintf("Plan %q, AllowFail: %v ", p.Name, p.AllowFail)
	if len(p.Verifiers) > 0 {
		prefix := "\n "
		r += fmt.Sprintf("%sVerifiers:", prefix)
		for i, a := range p.Verifiers {
			r += fmt.Sprintf("%s %d: %s", prefix, i, a.Describe(prefix+"  "))
		}
	} else {
		r += "\n No verifiers"
	}
	return r
}

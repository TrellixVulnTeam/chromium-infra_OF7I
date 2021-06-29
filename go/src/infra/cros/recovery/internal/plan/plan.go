// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package plan provides struts and functionality to use plans and actions.
package plan

import (
	"context"
	"log"

	"go.chromium.org/luci/common/errors"

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

// Run runs the recovery plan.
func (p *Plan) Run(ctx context.Context, args *execs.RunArgs) error {
	log.Printf("Plan %q: started.", p.Name)
	c := newRunCache()
	defer c.close()
	// TODO(otabek@): Add start-over loop if any recovery action was used and passed.
	if err := p.runVerifiers(ctx, c, args); err != nil {
		return errors.Annotate(err, "run plan %q", p.Name).Err()
	}
	log.Printf("Plan %q: finished successfully.", p.Name)
	return nil
}

// runVerifiers runs verifiers of the plan.
// Method the first check the result of verifier from cache and if not exist then perform the verifier.
func (p *Plan) runVerifiers(ctx context.Context, c *runCache, args *execs.RunArgs) error {
	for i, v := range p.Verifiers {
		if err, ok := c.getActionError(v); ok {
			if err == nil {
				log.Printf("Verifier %q: pass (cached).", v.Name)
				continue
			} else if v.AllowFail {
				log.Printf("Verifier %q: fail (cached). Error: %s", v.Name, err)
				v.logAllowedFailFault(i, len(p.Verifiers))
				continue
			}
			return errors.Annotate(err, "run verifier %q: fail (cached)", v.Name).Err()
		}
		if err := v.run(ctx, c, args); err != nil {
			if v.AllowFail {
				log.Printf("Verifier %q: fail. Error: %s", v.Name, err)
				v.logAllowedFailFault(i, len(p.Verifiers))
			} else {
				return errors.Annotate(err, "run verifier %q", v.Name).Err()
			}
		} else {
			log.Printf("Verifier %q: finished successfully.", v.Name)
		}
	}
	return nil
}

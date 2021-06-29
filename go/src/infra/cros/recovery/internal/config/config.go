// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package config provides struts to holds and read configs.
package config

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/plan"
)

// LoadPlans loads plans from the config.
// Steps:
// 1) Read default or custom configuration.
// 2) Parse plan's data one-by-one.
// 3) Collect and convert plan's actions.
// 4) Create plan and check verifiers.
func LoadPlans(ctx context.Context, planNames []string, cr io.Reader) (plans []*plan.Plan, err error) {
	var data []byte
	if cr != nil {
		log.Printf("Load plans: using provided custom config.")
		data, err = io.ReadAll(cr)
		if err != nil {
			return plans, errors.Annotate(err, "load plans").Err()
		}
	} else {
		log.Printf("Load plans: use default config.")
		data = []byte(defaultPlans)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return plans, errors.Annotate(err, "load plans").Err()
	}
	for _, planName := range planNames {
		log.Printf("Load plan %q: started.", planName)
		j, ok := config[planName]
		if !ok {
			return plans, errors.Reason("load plan %q: not found", planName).Err()
		}
		pj, ok := j.(map[string]interface{})
		if !ok {
			return plans, errors.Reason("load plan %q: expected to be json-object", planName).Err()
		}
		var p *plan.Plan
		if p, err = parsePlan(planName, pj); err != nil {
			return plans, errors.Annotate(err, "load plans").Err()
		}
		plans = append(plans, p)
		log.Printf("Load plan %q: finished.", planName)
	}
	return plans, nil
}

// parsePlan parses and converts data expected plan.
func parsePlan(name string, j map[string]interface{}) (p *plan.Plan, err error) {
	var verifiers []string
	var allowFail bool
	actions := make(map[string]*plan.Action)
	for k, v := range j {
		switch k {
		case "actions":
			aj, ok := v.(map[string]interface{})
			if !ok {
				return nil, errors.Reason("parse plan %q: expected to be json-object", name).Err()
			}
			actions, err = parseActions(aj)
			if err != nil {
				return nil, errors.Annotate(err, "parse plan %q", name).Err()
			}
		case "verifiers":
			verifiers = readStringSlice(v)
		case "allow_fail":
			allowFail = readBool(v)
		}
	}
	p = &plan.Plan{
		Name:      name,
		AllowFail: allowFail,
	}
	for _, v := range verifiers {
		if v == "" {
			// Skip verifier with empty name as it can be just mistake.
			continue
		}
		a, ok := actions[v]
		if !ok {
			a = createDefaultAction(v)
			actions[v] = a
		} else {
			// TODO(otabek@): Implement verification of actions loop.
		}
		p.Verifiers = append(p.Verifiers, a)
	}
	return p, nil
}

// action holds json representation of the action from config file.
type action struct {
	name         string
	execName     string
	dependencies []string
	recoveries   []string
	allowFail    bool
	allowCache   bool
}

// parseActions parses and converts data to map of actions available in the plan.
func parseActions(j map[string]interface{}) (map[string]*plan.Action, error) {
	am := make(map[string]*action)
	for k, v := range j {
		if k == "" {
			// Skipping actions with empty name.
			continue
		}
		aj, ok := v.(map[string]interface{})
		if !ok {
			return nil, errors.Reason("parse action %q: expected to be json-object", k).Err()
		}
		a := parseAction(k, aj)
		if _, ok := am[a.name]; ok {
			return nil, errors.Reason("parse actions: duplicate action %q", a.name).Err()
		}
		am[a.name] = a
	}
	actions := make(map[string]*plan.Action)
	if len(am) == 0 {
		// No actions to convert.
		// Always return initialize map as it still can be used later.
		return actions, nil
	}
	for _, as := range am {
		a := &plan.Action{
			Name:       as.name,
			ExecName:   as.execName,
			AllowFail:  as.allowFail,
			AllowCache: as.allowCache,
		}
		if a.ExecName == "" {
			a.ExecName = a.Name
		}
		actions[a.Name] = a
	}
	for _, a := range actions {
		as, ok := am[a.Name]
		if !ok {
			// We won't have json representation of action if it was added actions during checking dependencies of recoveries actions.
			continue
		}
		for _, d := range as.dependencies {
			if d == a.Name {
				return nil, errors.Reason("convert actions: found self-looped dependency %q", d).Err()
			}
			da, ok := actions[d]
			if !ok {
				// If dependency declared but did not have json representation then the action will be created with default settings.
				da = createDefaultAction(d)
				actions[d] = da
			}
			a.Dependencies = append(a.Dependencies, da)
		}
		for _, r := range as.recoveries {
			if r == a.Name {
				return nil, errors.Reason("convert actions: found self-looped recovery action %q", r).Err()
			}
			ra, ok := actions[r]
			if !ok {
				// If recovery declared but did not have json representation then the action will be created with default settings.
				ra = createDefaultAction(r)
				actions[r] = ra
			}
			a.Recoveries = append(a.Recoveries, ra)
		}
	}
	return actions, nil
}

// parseAction parses action data for local representation.
func parseAction(name string, j map[string]interface{}) *action {
	// TODO(otabek@): Add support fo custom configured action.
	a := &action{
		name:       name,
		allowCache: true,
	}
	for k, v := range j {
		switch k {
		case "dependencies":
			a.dependencies = readStringSlice(v)
		case "recoveries":
			a.recoveries = readStringSlice(v)
		case "exec_name":
			switch v.(type) {
			case string:
				a.execName = v.(string)
			}
		case "allow_fail":
			a.allowFail = readBool(v)
		case "allow_cache":
			switch v.(type) {
			case string:
				if "never" == strings.ToLower(v.(string)) {
					a.allowCache = false
				}
			}
		}
	}
	return a
}

// createDefaultAction creates action with default settings.
// This action is creating when a name is declared as dependency/recovery/verifier but was not described in actions by config.
func createDefaultAction(name string) *plan.Action {
	return &plan.Action{
		Name:       name,
		ExecName:   name,
		AllowFail:  false,
		AllowCache: true,
	}
}

// readStringSlice parses to the slice of strings.
func readStringSlice(s interface{}) (r []string) {
	if jl, ok := s.([]interface{}); ok {
		for _, v := range jl {
			switch v.(type) {
			case string:
				r = append(r, v.(string))
			}
		}
	}
	return
}

// Convert json value representation to bool.
// Support values as true: (int)1, (string)true, (bool)true.
func readBool(v interface{}) bool {
	switch v.(type) {
	case bool:
		return v.(bool)
	case string:
		return strings.ToLower(v.(string)) == "true"
	case int:
		return v.(int) == 1
	}
	return false
}

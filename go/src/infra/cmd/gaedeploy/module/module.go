// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package module implements reading and processing of GAE module YAMLs.
package module

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
)

// Module is a loaded mutable module's YAML.
type Module struct {
	Name string // e.g. "default"

	conf map[string]interface{} // deserialized YAML config
}

// ReadYAML loads the module's YAML in memory.
func ReadYAML(path string) (*Module, error) {
	blob, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Annotate(err, "failed to read the file").Err()
	}
	return parseYAML(blob)
}

// parseYAML unmarshals YAML and returns *Module from it.
func parseYAML(blob []byte) (*Module, error) {
	m := &Module{conf: map[string]interface{}{}}
	if err := yaml.Unmarshal(blob, &m.conf); err != nil {
		return nil, errors.Annotate(err, "failed to unmarshal").Err()
	}

	// Extract module's name, verify it is a string.
	name := m.conf["service"]
	if name == nil {
		name = m.conf["module"] // legacy name for the same property
	}
	if name == nil {
		m.Name = "default"
	} else if nm, ok := name.(string); ok {
		m.Name = nm
	} else {
		return nil, errors.Reason("bad service name %v, not a string", name).Err()
	}

	return m, nil
}

// Process renders and normalizes module's YAML.
//
// It throws away all legacy and unsupported fields and renders configuration
// variables. Roughly matches what luci-py's gae.py tool does.
//
// For variables substitution it uses very non-standard luci-py's extension.
// Variables can be defined in the YAML via `luci_gae_vars` field, like so:
//
//  luci_gae_vars:
//    example-app-id-dev:
//      AUTH_SERVICE_HOST: auth-service-dev.appspot.com
//    example-app-id-prod:
//      AUTH_SERVICE_HOST: auth-service-prod.appspot.com
//
// And then they can appear in the YAML as e.g. `${AUTH_SERVICE_HOST}`.
//
// We use what's in the YAML as a baseline (for compatibility with existing
// manifests), but additionally allow overriding such vars via CLI flags
// of `gaedeploy` (supplied here via `vars`).
//
// For the python counterpart, see
// https://chromium.googlesource.com/infra/luci/luci-py/+/9963ccd99c7/appengine/components/tool_support/gae_sdk_utils.py#221
//
// On success returns a set of variable names that were encountered in the YAML.
func (m *Module) Process(appID string, vars map[string]string) (stringset.Set, error) {
	// Delete legacy fields.
	delete(m.conf, "application") // provided via CLI flags
	delete(m.conf, "version")     // provided via CLI flags
	delete(m.conf, "module")      // we'll use "service" instead
	m.conf["service"] = m.Name

	// Pop our very custom section from the YAML, gcloud doesn't like it.
	definedVars := m.conf["luci_gae_vars"]
	delete(m.conf, "luci_gae_vars")

	// Don't mess any further with YAMLs that do not define any vars.
	if definedVars == nil {
		return stringset.New(0), nil
	}

	// Peel off few layers of interface{} ambiguity from definedVars.
	asMap, ok := asStrMap(definedVars)
	if !ok {
		return nil, errors.Reason("`luci_gae_vars` section should be a dict or dicts").Err()
	}
	varsPerAppID := make(varsDecl, len(asMap))
	for appID, appVars := range asMap {
		var ok bool
		if varsPerAppID[appID], ok = asStrMap(appVars); !ok {
			return nil, errors.Reason("`luci_gae_vars` section should be a dict or dicts").Err()
		}
	}

	// Replace ${...} with their concrete values.
	mutated, consumed, err := renderVars(m.conf, appID, varsPerAppID, vars)
	if err != nil {
		return nil, err
	}
	m.conf = mutated
	return consumed, nil
}

// DumpYAML returns a pretty-printed module's YAML.
func (m *Module) DumpYAML() ([]byte, error) {
	return yaml.Marshal(m.conf)
}

////////////////////////////////////////////////////////////////////////////////

// varsDecl is appID => var name => its value, as read from the YAML.
type varsDecl map[string]map[string]interface{}

// varsProvider takes a variable name and returns its value (int or string).
type varsProvider func(name string) (interface{}, error)

// asStrMap converts d to `map[string]interface{}` if possible.
func asStrMap(d interface{}) (map[string]interface{}, bool) {
	m, ok := d.(map[interface{}]interface{})
	if !ok {
		return nil, false
	}
	typed := make(map[string]interface{}, len(m))
	for k, v := range m {
		k, ok := k.(string)
		if !ok {
			return nil, false
		}
		typed[k] = v
	}
	return typed, true
}

// renderVars visits `m` and replaces ${VAR} with concrete values.
//
// Returns a partially modified copy of `m` and a set of var names that were
// substituted.
func renderVars(m map[string]interface{}, appID string, decl varsDecl, vals map[string]string) (map[string]interface{}, stringset.Set, error) {
	type varType int
	const Undef varType = 0
	const Int varType = 1
	const Str varType = 2

	// Grab a union of all possible vars in `decl` along with their types. We
	// support only strings and integers.
	types := map[string]varType{}
	for _, m := range decl {
		for k, v := range m {
			typ := Undef
			if _, ok := v.(int); ok {
				typ = Int
			} else if _, ok := v.(string); ok {
				typ = Str
			} else {
				return nil, nil, errors.Reason("variable %q has unsupported type %T", k, v).Err()
			}
			// Verify all per-app sections in `decl` agree on the type.
			if existing := types[k]; existing != Undef && existing != typ {
				return nil, nil, errors.Reason("variable %q has ambiguous type", k).Err()
			}
			types[k] = typ
		}
	}

	// Verify types in `vals` match (so basically check int-typed vars can be
	// parsed as integers). Note that we allow to pass variables that are not
	// mentioned in `decl`. They have string type.
	for k, v := range vals {
		if types[k] == Int {
			if _, err := strconv.ParseInt(v, 10, 32); err != nil {
				return nil, nil, errors.Reason("the value of variable %q is expected to be an integer, got %q", k, v).Err()
			}
		}
	}

	// Values of variables as specified in `decl` for the given appID (if any).
	baseline, _ := decl[appID]

	// We'll keep track of what vars were actually used.
	consumed := stringset.New(len(vals))

	// Provider takes a variable name and returns its value (either string or
	// int), or an error if such variable is not defined.
	provider := func(key string) (interface{}, error) {
		typ := types[key]
		if typ == Undef {
			typ = Str // default for variables not mentioned in `decl`
		}

		consumed.Add(key)

		// Have it in `vals`?
		if val, ok := vals[key]; ok {
			switch typ {
			case Int:
				i, err := strconv.ParseInt(val, 10, 32)
				if err != nil {
					panic("impossible") // already checked this
				}
				return int(i), nil
			case Str:
				return val, nil
			default:
				panic("impossible")
			}
		}

		// Have it `baseline`? No need to check the type, it already matches by
		// construction.
		if val, ok := baseline[key]; ok {
			return val, nil
		}

		return nil, errors.Reason("a value for variable %q is not provided", key).Err()
	}

	// `visit` doesn't understand map[string]interface{} specifically, it
	// wants interface{} keys. So traverse the top-layer explicitly.
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		var err error
		if out[k], err = visit(v, provider); err != nil {
			return nil, nil, err
		}
	}
	return out, consumed, nil
}

// visit recursively visits `obj` substituting vars in it via `p`.
//
// Returns either `obj` itself or a partially modified copy.
func visit(obj interface{}, p varsProvider) (out interface{}, err error) {
	switch o := obj.(type) {
	case string:
		out, err = renderString(o, p)

	case map[interface{}]interface{}:
		mut := make(map[interface{}]interface{}, len(o))
		for k, v := range o {
			if mut[k], err = visit(v, p); err != nil {
				return
			}
		}
		out = mut

	case []interface{}:
		mut := make([]interface{}, len(o))
		for i, v := range o {
			if mut[i], err = visit(v, p); err != nil {
				return
			}
		}
		out = mut

	default:
		out = obj
	}
	return
}

var varRe = regexp.MustCompile(`\$\{[\w]+\}`)

// renderString renders variables in a string resolving them via `p`.
func renderString(s string, p varsProvider) (out interface{}, err error) {
	// Detect direct hits like "${VAR}" to return the var values as is, to
	// preserve its type. This is important for integer-valued fields like
	// `max_concurrent_requests`.
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		key := strings.TrimSuffix(strings.TrimPrefix(s, "${"), "}")
		if !strings.ContainsAny(key, "{}") {
			return p(key)
		}
	}

	out = varRe.ReplaceAllStringFunc(s, func(match string) string {
		if err != nil {
			return "" // don't care, already failing
		}
		var repl interface{}
		repl, err = p(strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}"))
		return fmt.Sprintf("%v", repl) // to convert potential int to string
	})
	return
}

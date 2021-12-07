// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package manifest

import (
	"flag"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/flag/stringsetflag"
)

// Restrictions restrict what can be targeted by a manifest.
//
// This is useful when `cloudbuildhelper` operates over partially untrusted
// manifests to limit what they can reach.
//
// If some particular set is empty, the corresponding property is considered
// unrestricted.
type Restrictions struct {
	targets       stringsetflag.Flag
	steps         stringsetflag.Flag
	storage       stringsetflag.Flag
	registry      stringsetflag.Flag
	build         stringsetflag.Flag
	notifications stringsetflag.Flag
}

// AddFlags register -restrict-* flags in the flagset.
func (r *Restrictions) AddFlags(fs *flag.FlagSet) {
	fs.Var(&r.targets, "restrict-targets", "Prefixes of allowed target names.")
	fs.Var(&r.steps, "restrict-build-steps", "Allowed build step kinds.")
	fs.Var(&r.storage, "restrict-storage", "Prefixes of allowed tarball destinations.")
	fs.Var(&r.registry, "restrict-container-registry", "Allowed Container Registries.")
	fs.Var(&r.build, "restrict-cloud-build", "Allowed Cloud Build projects.")
	fs.Var(&r.notifications, "restrict-notifications", "Prefixes of allowed notification destinations.")
}

// CheckTargetName checks if the target name restrictions are violated.
//
// Returns a list of violations as human readable messages with details.
func (r *Restrictions) CheckTargetName(name string) (violations []string) {
	report(&violations, "target name", name, &r.targets, true)
	return
}

// CheckBuildSteps checks if any of the build step restrictions are violated.
//
// Returns a list of violations as human readable messages with details.
func (r *Restrictions) CheckBuildSteps(steps []*BuildStep) (violations []string) {
	for _, s := range steps {
		report(&violations, "build step kind", s.Concrete().Kind(), &r.steps, false)
	}
	return
}

// CheckInfra checks if any of the infra restrictions are violated.
//
// Returns a list of violations as human readable messages with details.
func (r *Restrictions) CheckInfra(m *Infra) (violations []string) {
	report(&violations, "Google Storage destination", m.Storage, &r.storage, true)
	report(&violations, "Container Registry destination", m.Registry, &r.registry, false)
	report(&violations, "Cloud Build project", m.CloudBuild.Project, &r.build, false)
	for _, n := range m.Notify {
		report(&violations, "notification destination", n.DestinationID(), &r.notifications, true)
	}
	return
}

func report(violations *[]string, title, value string, set *stringsetflag.Flag, pfx bool) {
	if value == "" || set.Data.Len() == 0 {
		return
	}

	all := set.Data.ToSortedSlice()
	for _, allowed := range all {
		if value == allowed || (pfx && strings.HasPrefix(value, allowed)) {
			return // allowed
		}
	}

	noun := "values"
	if pfx {
		noun = "prefixes"
	}

	*violations = append(*violations, fmt.Sprintf(
		"forbidden %s %q (allowed %s are %q)", title, value, noun, all))
}

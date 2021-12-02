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

// Restrictions restrict what infrastructure can be targeted by a manifest.
//
// This is useful when `cloudbuildhelper` operates over partially untrusted
// manifests to limit what they can reach.
//
// If some particular set is empty, the corresponding property is considered
// unrestricted.
type Restrictions struct {
	storage       stringsetflag.Flag
	registry      stringsetflag.Flag
	build         stringsetflag.Flag
	notifications stringsetflag.Flag
}

// AddFlags register -restrict-* flags in the flagset.
func (r *Restrictions) AddFlags(fs *flag.FlagSet) {
	fs.Var(&r.storage, "restrict-storage", "Prefixes of allowed tarball destinations.")
	fs.Var(&r.registry, "restrict-container-registry", "Allowed Container Registries.")
	fs.Var(&r.build, "restrict-cloud-build", "Allowed Cloud Build projects.")
	fs.Var(&r.notifications, "restrict-notifications", "Prefixes of allowed notification destinations.")
}

// CheckInfra checks if any of the restrictions are violated.
//
// Returns a list of violations as human readable messages with details.
func (r *Restrictions) CheckInfra(m *Infra) (violations []string) {
	reportViolations := func(title, value string, set *stringsetflag.Flag, pfx bool) {
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

		violations = append(violations, fmt.Sprintf(
			"forbidden %s %q (allowed %s are %q)", title, value, noun, all))
	}

	reportViolations("Google Storage destination", m.Storage, &r.storage, true)
	reportViolations("Container Registry destination", m.Registry, &r.registry, false)
	reportViolations("Cloud Build project", m.CloudBuild.Project, &r.build, false)
	for _, n := range m.Notify {
		reportViolations("notification destination", n.DestinationID(), &r.notifications, true)
	}

	return
}

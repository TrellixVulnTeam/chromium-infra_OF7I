// Copyright 2019 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"net/mail"
	"net/url"
	"regexp"
	"time"

	"github.com/golang/protobuf/ptypes"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/config/validation"
)

// The regex rule that all assigner IDs must conform to.
var assignerIDRegex = regexp.MustCompile(`^([a-z0-9]+-?)*[a-z0-9]$`)

// The regex rule for RotaNG rotation names.
var rotationNameRegex = regexp.MustCompile(`^([[:alnum:]][[:word:]- ]?)*[[:alnum:]]$`)

// The regex rule for rotation-proxy rotation names.
var rotationProxyNameRegex = regexp.MustCompile(`^(oncallator|grotation):[a-z0-9\-]+$`)

func validateConfig(c *validation.Context, cfg *Config) {
	validateAccessGroup(c, cfg.AccessGroup)
	validateMonorailHostname(c, cfg.MonorailHostname)
	validateAssigners(c, cfg.Assigners)
	validateRotangHostname(c, cfg.RotangHostname)
}

func validateAccessGroup(c *validation.Context, group string) {
	c.Enter("access_group: %s", group)
	if group == "" {
		c.Errorf("empty value is not allowed")
	}
	c.Exit()
}

func validateMonorailHostname(c *validation.Context, hostname string) {
	c.Enter("monorail_hostname")
	if hostname == "" {
		c.Errorf("empty value is not allowed")
	} else if _, err := url.Parse(hostname); err != nil {
		c.Errorf("invalid hostname: %s", hostname)
	}
	c.Exit()
}

func validateRotangHostname(c *validation.Context, hostname string) {
	c.Enter("rotang_hostname")
	if hostname == "" {
		c.Errorf("empty value is not allowed")
	} else if _, err := url.Parse(hostname); err != nil {
		c.Errorf("invalid hostname: %s", hostname)
	}
	c.Exit()
}

func validateAssigners(c *validation.Context, assigners []*Assigner) {
	// check duplicate IDs.
	seen := stringset.New(len(assigners))
	for i, assigner := range assigners {
		c.Enter("assigner #%d:%s", i+1, assigner.Id)
		if !seen.Add(assigner.Id) {
			c.Errorf("duplicate id")
		}
		validateAssigner(c, assigner)
		c.Exit()
	}
}

func validateAssigner(c *validation.Context, assigner *Assigner) {
	// to make URLs short and simple when they are made with assigner ids.
	if !assignerIDRegex.MatchString(assigner.Id) {
		c.Errorf(
			"invalid id; only lowercase alphabet letters and numbers are " +
				"allowed. A hyphen may be placed between letters and numbers",
		)
	}

	// owners should be all valid email addresses.
	for _, owner := range assigner.Owners {
		c.Enter("owner %q", owner)
		if _, err := mail.ParseAddress(owner); err != nil {
			c.Errorf("invalid email address: %s", err)
		}
		c.Exit()
	}

	if assigner.Interval == nil {
		c.Errorf("missing interval")
	} else {
		d, err := ptypes.Duration(assigner.Interval)
		if err != nil {
			c.Errorf("invalid interval: %s", err)
		} else if d < time.Minute {
			c.Errorf("interval should be at least one minute")
		}
	}

	if assigner.IssueQuery == nil {
		c.Errorf("missing issue_query")
	} else {
		c.Enter("issue_query")
		if assigner.IssueQuery.Q == "" {
			c.Errorf("missing q")
		}
		if len(assigner.IssueQuery.ProjectNames) == 0 {
			c.Errorf("missing project_names")
		}
		c.Exit()
	}

	if len(assigner.Assignees) == 0 && len(assigner.Ccs) == 0 {
		c.Errorf("at least one of assignees or ccs must be given")
	} else {
		for i, source := range assigner.Assignees {
			c.Enter("assignee %d", i+1)
			validateUserSource(c, source)
			c.Exit()
		}
		for i, source := range assigner.Ccs {
			c.Enter("cc %d", i+1)
			validateUserSource(c, source)
			c.Exit()
		}
	}
}

func validateUserSource(c *validation.Context, source *UserSource) {
	if oncall := source.GetOncall(); oncall != nil {
		validateOncall(c, oncall)
	} else if rotation := source.GetRotation(); rotation != nil {
		validateRotation(c, rotation)
	} else if email := source.GetEmail(); email != "" {
		validateEmail(c, email)
	} else {
		c.Errorf("missing or unknown user source")
	}
}

func validateOncall(c *validation.Context, oncall *Oncall) {
	var name string
	if oncall.Name != "" {
		if oncall.Rotation != "" {
			c.Errorf("both name and rotation are specified")
		}
		name = oncall.Name
	} else {
		if oncall.Rotation == "" {
			c.Errorf("either name or rotation must be specified")
		}
		name = oncall.Rotation
	}

	if name != "" && !rotationNameRegex.MatchString(name) {
		c.Errorf(
			"invalid id; only alphabet and numeric characters are allowed, " +
				"but a space, hyphen, or underscore may be put between " +
				"the first and last characters.",
		)
	}
	if oncall.Position == Oncall_UNSET {
		c.Errorf("missing oncall position")
	}
}

func validateRotation(c *validation.Context, rotation *Oncall) {
	var name string
	if rotation.Name != "" {
		if rotation.Rotation != "" {
			c.Errorf("both name and rotation are specified")
		}
		name = rotation.Name
	} else {
		if rotation.Rotation == "" {
			c.Errorf("either name or rotation must be specified")
		}
		name = rotation.Rotation
	}

	if name != "" && !rotationProxyNameRegex.MatchString(name) {
		c.Errorf(
			"invalid id; prefix must be 'oncallator:' or 'grotation:' " +
				"followed by a name containing only alphanumeric " +
				"characters and dashes",
		)
	}
	if rotation.Position == Oncall_UNSET {
		c.Errorf("missing rotation position")
	}
}

func validateEmail(c *validation.Context, email string) {
	// All Monorail users should be valid email addresses.
	if _, err := mail.ParseAddress(email); err != nil {
		c.Errorf("invalid email: %s", err)
	}
}

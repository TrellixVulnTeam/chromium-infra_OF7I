// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"regexp"

	"go.chromium.org/luci/common/errors"
	configpb "go.chromium.org/luci/common/proto/config"
)

// Filter reports true if we should visit the given project.
type Filter func(projectID string) bool

// NewFilter constructs a filter from a list of regexps.
//
// If the list is empty, returns a filter that rejects all projects.
func NewFilter(regexps []string) (Filter, error) {
	regs := make([]*regexp.Regexp, len(regexps))
	for i, str := range regexps {
		str = "^(" + str + ")$"
		var err error
		if regs[i], err = regexp.Compile(str); err != nil {
			return nil, errors.Annotate(err, "when compiling %q", str).Err()
		}
	}
	return func(projectID string) bool {
		for _, reg := range regs {
			if reg.MatchString(projectID) {
				return true
			}
		}
		return false
	}, nil
}

// Apply returns projects that passed the filter.
func (f Filter) Apply(projs []*configpb.Project) []*configpb.Project {
	var filtered []*configpb.Project
	for _, proj := range projs {
		if f(proj.Id) {
			filtered = append(filtered, proj)
		}
	}
	return filtered
}

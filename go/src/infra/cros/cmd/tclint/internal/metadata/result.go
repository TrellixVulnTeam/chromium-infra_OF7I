// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package metadata provides functions to lint Chrome OS integration test
// metadata.
package metadata

import (
	"fmt"
	"sort"
	"strings"

	"go.chromium.org/luci/common/errors"
)

// Result contains diagnostic messages from metadata lint.
type Result struct {
	Errors errors.MultiError
}

// Merge merges another result into the current result.
func (r *Result) Merge(o Result) {
	r.Errors = append(r.Errors, o.Errors...)
}

// MergeWithContext merges another result into the current result, prefixed with
// some context.
func (r *Result) MergeWithContext(o Result, fmt string, args ...interface{}) {
	for _, err := range o.Errors {
		// This captures the wrong stack frame. errors.Annotate() doesn't have
		// a way to specify skipping N frames (similar to testing.T.Helper())
		// yet. We don't actually render the stack trace, so this is OK.
		r.Errors = append(r.Errors, errors.Annotate(err, fmt, args...).Err())
	}
}

// AppendError appends an error to result.
func (r *Result) AppendError(fmt string, args ...interface{}) {
	r.Errors = append(r.Errors, errors.Reason(fmt, args...).Err())
}

// AppendErrorWithContext appends an error to result, prefixed with some
// context.
func (r *Result) AppendErrorWithContext(err error, fmt string, args ...interface{}) {
	r.Errors = append(r.Errors, errors.Annotate(err, fmt, args...).Err())
}

// Display returns a user-friendly display of diagnostics from a Result.
func (r *Result) Display() []string {
	ss := []string{}
	if r.Errors != nil {
		ss = append(ss, "## Errors ##")
		for _, err := range r.Errors {
			ss = append(ss, err.Error())
		}
	}
	sort.Strings(ss)
	return breakAndIndentMultiLine(ss)
}

func errorResult(fmt string, args ...interface{}) Result {
	return Result{
		Errors: errors.NewMultiError(errors.Reason(fmt, args...).Err()),
	}
}

func breakAndIndentMultiLine(ss []string) []string {
	ret := make([]string, 0, len(ss))
	for _, s := range ss {
		ps := strings.Split(s, "\n")
		// strings.Split() returns at least one element, even for empty input.
		ret = append(ret, ps[0])
		for _, p := range ps[1:] {
			ret = append(ret, fmt.Sprintf("  %s", p))
		}
	}
	return ret
}

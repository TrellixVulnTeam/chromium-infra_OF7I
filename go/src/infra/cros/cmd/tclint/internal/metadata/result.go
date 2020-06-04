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
	"sync"

	"go.chromium.org/luci/common/errors"
)

// Result contains diagnostic messages from metadata lint.
type Result struct {
	Errors   errors.MultiError
	prefixes []string
}

// PushContext adds context to be prepended to all diagnostic messages.
//
// Contexts can be stacked by calling PushContext() repeatedly.
// Returns a function to pop the pushed context.
func (r *Result) PushContext(c string) (popContext func()) {
	r.prefixes = append(r.prefixes, fmt.Sprintf("%s: ", c))
	// Stay true to our API -- returned function can only pop one context.
	// We still don't deal with the possibility that the returned closers from
	// multiple PushContext() calls can be called in arbitrary order, and will
	// have the same effect.
	o := sync.Once{}
	return func() {
		o.Do(r.dropContext)
	}
}

// Merge merges another result into the current result.
//
// Diagnostic messages from the incoming Result are prefixed with the current
// Result's context.
// Context from the incoming Result is ignored.
func (r *Result) Merge(o Result) {
	for _, err := range o.Errors {
		r.AppendError(err.Error())
	}
}

// AppendError appends an error to result, prefixed with current context.
func (r *Result) AppendError(fmt string, args ...interface{}) {
	r.Errors = append(r.Errors, errors.Reason(r.prefixWithContext(fmt), args...).Err())
}

// IsValid returns false if the result contains any validation errors, true
// otherwise.
func (r *Result) IsValid() bool {
	return len(r.Errors) == 0
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

func (r *Result) dropContext() {
	r.prefixes = r.prefixes[:len(r.prefixes)-1]
}

func (r *Result) prefixWithContext(s string) string {
	return strings.Join(r.prefixes, "") + s
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

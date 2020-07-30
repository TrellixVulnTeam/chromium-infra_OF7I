// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// To define a new rule, create a new struct which implements interface
// redirectRule. Then, add it to newRedirectRules.
package main

import (
	"errors"
	"regexp"
)

var errNoMatch = errors.New("no match found")
var numberRedirectRegex = regexp.MustCompile(`^/(\d{1,8})$`)
var fullCommitHashRegex = regexp.MustCompile(`^/([[:xdigit:]]{40})$`)

type redirectRule interface {
	// (redirect url, error) is returned if redirect rule is able to handle
	// requested URL. If there is no match, error=noMatchFound is returned.
	// All other errors indicate dependency issues (e.g. database
	// connectivity).
	getRedirect(url string) (string, error)
}

// numberRedirectRule redirects from sequential numbers to the git commit in
// chromium/src.
type numberRedirectRule struct{}

func (r *numberRedirectRule) getRedirect(url string) (string,
	error) {
	result := numberRedirectRegex.FindStringSubmatch(url)
	if len(result) == 0 {
		return "", errNoMatch
	}

	// TODO(https://crbug.com/1109315): Implement
	return "", errors.New("number redirect not implemented")
}

// fullCommitHashRule finds a commit across all indexed repositories and, if
// found, returns URL to the commit. If there are multiple matches (for mirrors
// and forks), it uses repo priority to determine where user should be
// redirected.
type fullCommitHashRule struct{}

func (r *fullCommitHashRule) getRedirect(url string) (string, error) {
	result := fullCommitHashRegex.FindStringSubmatch(url)
	if len(result) == 0 {
		return "", errNoMatch
	}

	// TODO(https://crbug.com/1109315): Implement
	return "", errors.New("full commit hash redirect not implemented")
}

type redirectRules struct {
	rules []redirectRule
}

// TODO(https://crbug.com/1109315): pass redirect struct
func newRedirectRules() *redirectRules {
	return &redirectRules{
		rules: []redirectRule{
			&numberRedirectRule{},
			&fullCommitHashRule{},
		},
	}
}

// findRedirectURL returns destination URL on the first matching redirectRule.
// If nothing is found, errNoMatch is returned.
func (r *redirectRules) findRedirectURL(url string) (string, error) {
	for _, rule := range r.rules {
		url, err := rule.getRedirect(url)
		if err == errNoMatch {
			continue
		}
		return url, err
	}
	return "", errNoMatch
}

// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package redirect contains logic for resovling ambiquios redirects and
// generic Git Web UI URLs.
// To define a new redirect rule, create a new struct which implements interface
// redirectRule. Then, add it to NewRules.
package redirect

import (
	"context"
	"infra/appengine/cr-rev/models"
	"infra/appengine/cr-rev/utils"
	"regexp"
	"strconv"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/gae/service/datastore"
)

var numberRedirectRegex = regexp.MustCompile(`^/(\d{1,8})(?:/(.*))?$`)
var fullCommitHashRegex = regexp.MustCompile(`^/([[:xdigit:]]{40})(?:/(.*))?$`)

// List of valid positions refs for numberRedirectRule
var chromiumPositionRefs = stringset.Set{
	"refs/heads/main":                         struct{}{},
	"refs/heads/master":                       struct{}{},
	"svn://svn.chromium.org/chrome":           struct{}{},
	"svn://svn.chromium.org/chrome/trunk":     struct{}{},
	"svn://svn.chromium.org/chrome/trunk/src": struct{}{},
}

type redirectRule interface {
	// (redirect url, error) is returned if redirect rule is able to handle
	// requested URL. If there is no match, error=noMatchFound is returned.
	// All other errors indicate dependency issues (e.g. database
	// connectivity).
	getRedirect(ctx context.Context, url string) (string, *models.Commit, error)
}

// numberRedirectRule redirects from sequential numbers to the git commit in
// chromium/src.
type numberRedirectRule struct {
	gitRedirect GitRedirect
}

func (r *numberRedirectRule) getRedirect(ctx context.Context, url string) (string,
	*models.Commit, error) {
	result := numberRedirectRegex.FindStringSubmatch(url)
	if len(result) == 0 {
		return "", nil, ErrNoMatch
	}
	id, err := strconv.Atoi(result[1])
	if err != nil {
		return "", nil, err
	}

	commits := []*models.Commit{}
	q := datastore.NewQuery("Commit").
		Eq("PositionNumber", id).
		Eq("Host", "chromium").
		Eq("Repository", "chromium/src")

	err = datastore.GetAll(ctx, q, &commits)
	if err != nil {
		return "", nil, err
	}

	for _, commit := range commits {
		if chromiumPositionRefs.Has(commit.PositionRef) {
			path := ""
			if len(result) == 3 {
				path = result[2]
			}
			url, err = r.gitRedirect.Commit(*commit, path)
			if err != nil {
				return "", nil, err
			}
			return url, commit, nil
		}
	}
	return "", nil, ErrNoMatch
}

// fullCommitHashRule finds a commit across all indexed repositories and, if
// found, returns URL to the commit. If there are multiple matches (for mirrors
// and forks), it uses repo priority to determine where user should be
// redirected.
type fullCommitHashRule struct {
	gitRedirect GitRedirect
}

func (r *fullCommitHashRule) getRedirect(ctx context.Context, url string) (string,
	*models.Commit, error) {
	result := fullCommitHashRegex.FindStringSubmatch(url)
	if len(result) == 0 {
		return "", nil, ErrNoMatch
	}
	commits, err := models.FindCommitsByHash(ctx, result[1])
	if err != nil {
		return "", nil, err
	}

	commit := utils.FindBestCommit(ctx, commits)
	if commit == nil {
		return "", nil, ErrNoMatch
	}

	path := ""
	if len(result) == 3 {
		path = result[2]
	}
	url, err = r.gitRedirect.Commit(*commit, path)
	if err != nil {
		return "", nil, err
	}
	return url, commit, nil
}

// Rules holds all available redirect rules. The order of rules
// matter, so generic / catch-all rules should be last.
type Rules struct {
	rules []redirectRule
}

// NewRules creates new instance of RedirectRules with hardcoded rules.
// New rules should be added here.
func NewRules(redirect GitRedirect) *Rules {
	return &Rules{
		rules: []redirectRule{
			&numberRedirectRule{
				gitRedirect: redirect,
			},
			&fullCommitHashRule{
				gitRedirect: redirect,
			},
		},
	}
}

// FindRedirectURL returns destination URL on the first matching redirectRule.
// If nothing is found, errNoMatch is returned.
func (r *Rules) FindRedirectURL(ctx context.Context, url string) (string, *models.Commit, error) {
	for _, rule := range r.rules {
		url, commit, err := rule.getRedirect(ctx, url)
		if err == ErrNoMatch {
			continue
		}
		return url, commit, err
	}
	return "", nil, ErrNoMatch
}

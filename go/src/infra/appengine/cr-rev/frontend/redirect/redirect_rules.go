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
	"fmt"
	"infra/appengine/cr-rev/common"
	"infra/appengine/cr-rev/models"
	"infra/appengine/cr-rev/utils"
	"regexp"
	"strconv"

	"go.chromium.org/luci/gae/service/datastore"
)

var rietveldRedirectRegex = regexp.MustCompile(`^/(\d{9,39})(?:/(.*))?$`)
var numberRedirectRegex = regexp.MustCompile(`^/(\d{1,8})(?:/(.*))?$`)
var fullCommitHashRegex = regexp.MustCompile(`^/([[:xdigit:]]{40})(?:/(.*))?$`)
var shortCommitHashRegex = regexp.MustCompile(`^/([[:xdigit:]]{6,39})(?:/(.*))?$`)
var diffFullHashRegex = regexp.MustCompile(`^/([[:xdigit:]]{40})(\.{2,3})([[:xdigit:]]{40})`)
var diffShortHashRegex = regexp.MustCompile(`^/([[:xdigit:]]{6,39})(\.{2,3})([[:xdigit:]]{6,40})`)

// List of valid positions refs for numberRedirectRule
var chromiumPositionRefs = []string{
	"refs/heads/main",
	"refs/heads/master",
	"svn://svn.chromium.org/chrome",
	"svn://svn.chromium.org/chrome/trunk",
	"svn://svn.chromium.org/chrome/trunk/src",
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

	queries := make([]*datastore.Query, len(chromiumPositionRefs))
	for i, ref := range chromiumPositionRefs {
		queries[i] = datastore.NewQuery("Commit").
			Eq("PositionNumber", id).
			Eq("Host", "chromium").
			Eq("Repository", "chromium/src").
			Eq("PositionRef", ref)
	}

	var commit *models.Commit
	err = datastore.RunMulti(ctx, queries, func(c *models.Commit) {
		if commit != nil {
			return
		}
		commit = c
	})
	if err != nil {
		return "", nil, err
	}
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
		// If there are no matches, always redirect to chromium/src
		commit = &models.Commit{
			Host:       "chromium",
			Repository: "chromium/src",
			CommitHash: result[1],
		}
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

// shortCommitHashRule always redirects to chromium/src in hope that the commit
// exists.
// TODO Add support to query datastore by short hash. This may be done using
// .Gte(hash) and .Lt(hash+1).
type shortCommitHashRule struct {
	gitRedirect GitRedirect
}

func (r *shortCommitHashRule) getRedirect(ctx context.Context, url string) (string,
	*models.Commit, error) {
	result := shortCommitHashRegex.FindStringSubmatch(url)
	if len(result) == 0 {
		return "", nil, ErrNoMatch
	}
	commit := models.Commit{
		Host:       "chromium",
		Repository: "chromium/src",
		CommitHash: result[1],
	}
	path := ""
	if len(result) == 3 {
		path = result[2]
	}
	url, err := r.gitRedirect.Commit(commit, path)
	if err != nil {
		return "", nil, err
	}
	return url, &commit, nil
}

// diffFullHashRule finds two commits across all indexed repositories and, if
// found, returns URL to the commit. If there are multiple matches (for mirrors
// and forks), it uses repo priority to determine where user should be
// redirected.
type diffFullHashRule struct {
	gitRedirect GitRedirect
}

func (r *diffFullHashRule) getRedirect(ctx context.Context, url string) (string,
	*models.Commit, error) {
	result := diffFullHashRegex.FindStringSubmatch(url)
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

	gitCommit := common.GitCommit{
		Repository: common.GitRepository{
			Host: commit.Host,
			Name: commit.Repository,
		},
		Hash: result[3],
	}
	commit2 := &models.Commit{
		ID: gitCommit.ID(),
	}
	err = datastore.Get(ctx, commit2)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return "", nil, ErrNoMatch
		}
		return "", nil, err
	}

	url, err = r.gitRedirect.Diff(*commit, *commit2)
	if err != nil {
		return "", nil, err
	}
	return url, commit, nil
}

// diffShortCommitHashRule always redirects to chromium/src in hope that
// commits exist.
type diffShortHashRule struct {
	gitRedirect GitRedirect
}

func (r *diffShortHashRule) getRedirect(ctx context.Context, url string) (string,
	*models.Commit, error) {
	result := diffShortHashRegex.FindStringSubmatch(url)
	if len(result) < 4 {
		return "", nil, ErrNoMatch
	}
	commit1 := models.Commit{
		Host:       "chromium",
		Repository: "chromium/src",
		CommitHash: result[1],
	}
	commit2 := models.Commit{
		Host:       "chromium",
		Repository: "chromium/src",
		CommitHash: result[3],
	}
	url, err := r.gitRedirect.Diff(commit1, commit2)
	if err != nil {
		return "", nil, err
	}
	return url, &commit1, nil
}

type rietveldRule struct {
}

func (r *rietveldRule) getRedirect(ctx context.Context, url string) (string,
	*models.Commit, error) {
	result := rietveldRedirectRegex.FindStringSubmatch(url)
	if len(result) == 0 {
		return "", nil, ErrNoMatch
	}
	url = fmt.Sprintf("https://codereview.chromium.org/%s", result[1])
	return url, nil, nil
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
			&rietveldRule{},
			&shortCommitHashRule{
				gitRedirect: redirect,
			},
			&diffFullHashRule{
				gitRedirect: redirect,
			},
			&diffShortHashRule{
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

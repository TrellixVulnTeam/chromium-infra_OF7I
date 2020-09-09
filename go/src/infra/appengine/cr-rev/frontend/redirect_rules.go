// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// To define a new rule, create a new struct which implements interface
// redirectRule. Then, add it to newRedirectRules.
package main

import (
	"context"
	"errors"
	"infra/appengine/cr-rev/config"
	"infra/appengine/cr-rev/models"
	"regexp"
	"strconv"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
)

var errNoMatch = errors.New("no match found")

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

// findBestCommit finds the best commit to redirect to based on configuration:
// * if commit's repository has a priority set, it's returned immedietely
// * if commit's repository has do not index, it won't be returned unless it's
// only available commit
// If config can't be retrieved, very first commit is returned.
func findBestCommit(ctx context.Context, commits []models.Commit) *models.Commit {
	if len(commits) == 0 {
		return nil
	}

	cfg, err := config.Get(ctx)
	if err != nil {
		logging.Errorf(ctx, "Couldn't get config, using first commit as the best")
		return &commits[0]
	}
	repoPriorityMap := map[string]map[string]*config.Repository{}
	for _, host := range cfg.Hosts {
		m := map[string]*config.Repository{}
		for _, repo := range host.GetRepos() {
			m[repo.GetName()] = repo
		}
		repoPriorityMap[host.GetName()] = m
	}
	ret := &commits[0]

	for _, commit := range commits {
		m, ok := repoPriorityMap[commit.Host]
		if !ok {
			continue
		}
		cfg, ok := m[commit.Repository]
		if !ok {
			continue
		}
		if cfg.GetPriority() {
			return &commit
		}
		if cfg.GetDoNotIndex() {
			continue
		}
		ret = &commit
	}
	return ret
}

type redirectRule interface {
	// (redirect url, error) is returned if redirect rule is able to handle
	// requested URL. If there is no match, error=noMatchFound is returned.
	// All other errors indicate dependency issues (e.g. database
	// connectivity).
	getRedirect(ctx context.Context, url string) (string, error)
}

// numberRedirectRule redirects from sequential numbers to the git commit in
// chromium/src.
type numberRedirectRule struct {
	gitRedirect gitRedirect
}

func (r *numberRedirectRule) getRedirect(ctx context.Context, url string) (string,
	error) {
	result := numberRedirectRegex.FindStringSubmatch(url)
	if len(result) == 0 {
		return "", errNoMatch
	}
	id, err := strconv.Atoi(result[1])
	if err != nil {
		return "", err
	}

	commits := []models.Commit{}
	q := datastore.NewQuery("Commit").
		Eq("PositionNumber", id).
		Eq("Host", "chromium").
		Eq("Repository", "chromium/src")

	err = datastore.GetAll(ctx, q, &commits)
	if err != nil {
		return "", err
	}

	for _, commit := range commits {
		if chromiumPositionRefs.Has(commit.PositionRef) {
			path := ""
			if len(result) == 3 {
				path = result[2]
			}
			return r.gitRedirect.commit(commit, path)
		}
	}
	return "", errNoMatch
}

// fullCommitHashRule finds a commit across all indexed repositories and, if
// found, returns URL to the commit. If there are multiple matches (for mirrors
// and forks), it uses repo priority to determine where user should be
// redirected.
type fullCommitHashRule struct {
	gitRedirect gitRedirect
}

func (r *fullCommitHashRule) getRedirect(ctx context.Context, url string) (string, error) {
	result := fullCommitHashRegex.FindStringSubmatch(url)
	if len(result) == 0 {
		return "", errNoMatch
	}

	commits := []models.Commit{}
	q := datastore.NewQuery("Commit").Eq("CommitHash", result[1])
	err := datastore.GetAll(ctx, q, &commits)
	if err != nil {
		return "", err
	}

	commit := findBestCommit(ctx, commits)
	if commit == nil {
		return "", errNoMatch
	}

	path := ""
	if len(result) == 3 {
		path = result[2]
	}
	return r.gitRedirect.commit(*commit, path)
}

type redirectRules struct {
	rules []redirectRule
}

// TODO(https://crbug.com/1109315): pass redirect struct
func newRedirectRules() *redirectRules {
	redirect := &gitilesRedirect{}
	return &redirectRules{
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

// findRedirectURL returns destination URL on the first matching redirectRule.
// If nothing is found, errNoMatch is returned.
func (r *redirectRules) findRedirectURL(ctx context.Context, url string) (string, error) {
	for _, rule := range r.rules {
		url, err := rule.getRedirect(ctx, url)
		if err == errNoMatch {
			continue
		}
		return url, err
	}
	return "", errNoMatch
}

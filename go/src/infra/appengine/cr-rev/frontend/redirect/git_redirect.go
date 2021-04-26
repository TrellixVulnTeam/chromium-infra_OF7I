// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package redirect

import (
	"errors"
	"fmt"

	"infra/appengine/cr-rev/models"
)

var (
	// ErrNoMatch indicates no commit can be found.
	ErrNoMatch = errors.New("no match found")

	errNotIdenticalRepositories = errors.New("Not identical repositories")
	errNotSupportedRepository   = errors.New("Repository is not supported")
)

// Maps GoB hosts to Codesearch URL
var codesearchMapping = map[string]string{
	"chromium": "https://source.chromium.org/chromium/",
}

// GitRedirect is interface for generating URLs.
type GitRedirect interface {
	// Commit returns URL to a specific commit.
	Commit(models.Commit, string) (string, error)
	// Diff returns URL for viewing difference between two commits.
	Diff(models.Commit, models.Commit) (string, error)
}

type gitilesRedirect struct{}

// NewGitilesRedirect returns struct that implements GitRedirect interface.
// All methods return URLs to Gitiles instance.
func NewGitilesRedirect() GitRedirect {
	return &gitilesRedirect{}
}

// Commit returns URL to a specific commit.
func (r *gitilesRedirect) Commit(c models.Commit, path string) (string, error) {
	url := fmt.Sprintf("https://%s.googlesource.com/%s/+/%s", c.Host, c.Repository, c.CommitHash)
	if path != "" {
		url += "/" + path
	}
	return url, nil
}

// Diff returns URL for viewing difference between two commits.
func (r *gitilesRedirect) Diff(c1, c2 models.Commit) (string, error) {
	if !c1.SameRepoAs(c2) {
		return "", errNotIdenticalRepositories
	}
	url := fmt.Sprintf(
		"https://%s.googlesource.com/%s/+/%s..%s",
		c1.Host, c1.Repository, c1.CommitHash, c2.CommitHash)
	return url, nil
}

type codesearchRedirect struct{}

// NewCodesearchRedirect returns struct that implements GitRedirect interface.
// All methods return URLs to source.chromium.org instance.
func NewCodesearchRedirect() GitRedirect {
	return &codesearchRedirect{}
}

// Commit returns URL to a specific commit.
func (r *codesearchRedirect) Commit(c models.Commit, path string) (string, error) {
	url, ok := codesearchMapping[c.Host]
	if !ok {
		return "", errNotSupportedRepository
	}
	url += fmt.Sprintf("%s/+/%s", c.Repository, c.CommitHash)
	if path != "" {
		url += ":" + path
	}
	return url, nil
}

// Diff returns URL for viewing difference between two commits.
func (r *codesearchRedirect) Diff(c1, c2 models.Commit) (string, error) {
	if !c1.SameRepoAs(c2) {
		return "", errNotIdenticalRepositories
	}

	url, ok := codesearchMapping[c1.Host]
	if !ok {
		return "", errNotSupportedRepository
	}

	url = fmt.Sprintf(
		"%s%s/+/%s...%s",
		url, c1.Repository, c1.CommitHash, c2.CommitHash)
	return url, nil
}

package main

import (
	"errors"
	"fmt"

	"infra/appengine/cr-rev/models"
)

var errNotIdenticalRepositories = errors.New("Not identical repositories")
var errNotSupportedRepository = errors.New("Repository is not supported")

// Maps GoB hosts to Codesearch URL
var codesearchMapping = map[string]string{
	"chromium": "https://source.chromium.org/chromium/",
}

type gitRedirect interface {
	commit(models.Commit, string) (string, error)
	diff(models.Commit, models.Commit) (string, error)
}

type gitilesRedirect struct{}

func (r *gitilesRedirect) commit(c models.Commit, path string) (string, error) {
	url := fmt.Sprintf("https://%s.googlesource.com/%s/+/%s", c.Host, c.Repository, c.CommitHash)
	if path != "" {
		url += "/" + path
	}
	return url, nil
}

func (r *gitilesRedirect) diff(c1, c2 models.Commit) (string, error) {
	if !c1.SameRepoAs(c2) {
		return "", errNotIdenticalRepositories
	}
	url := fmt.Sprintf(
		"https://%s.googlesource.com/%s/+/%s...%s",
		c1.Host, c1.Repository, c1.CommitHash, c2.CommitHash)
	return url, nil
}

type codesearchRedirect struct{}

func (r *codesearchRedirect) commit(c models.Commit, path string) (string, error) {
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

func (r *codesearchRedirect) diff(c1, c2 models.Commit) (string, error) {
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

package main

import (
	"errors"
	"fmt"

	"infra/appengine/cr-rev/common"
)

var errNotIdenticalRepositories = errors.New("Not identical repositories")
var errNotSupportedRepository = errors.New("Repository is not supported")

// Maps GoB hosts to Codesearch URL
var codesearchMapping = map[string]string{
	"chromium.googlesource.com": "https://source.chromium.org/chromium/",
}

func areIdenticalRepositories(c1, c2 common.GitCommit) bool {
	return c1.Host == c2.Host && c1.Repository == c2.Repository
}

type gitRedirect interface {
	commit(common.GitCommit, string) (string, error)
	diff(common.GitCommit, common.GitCommit) (string, error)
}

type gitilesRedirect struct{}

func (r *gitilesRedirect) commit(c common.GitCommit, path string) (string, error) {
	url := fmt.Sprintf("https://%s/%s/+/%s", c.Host, c.Repository, c.Sha1)
	if path != "" {
		url += "/" + path
	}
	return url, nil
}

func (r *gitilesRedirect) diff(c1, c2 common.GitCommit) (string, error) {
	if !areIdenticalRepositories(c1, c2) {
		return "", errNotIdenticalRepositories
	}
	url := fmt.Sprintf(
		"https://%s/%s/+/%s...%s",
		c1.Host, c1.Repository, c1.Sha1, c2.Sha1)
	return url, nil
}

type codesearchRedirect struct{}

func (r *codesearchRedirect) commit(c common.GitCommit, path string) (string, error) {
	url, ok := codesearchMapping[c.Host]
	if !ok {
		return "", errNotSupportedRepository
	}
	url += fmt.Sprintf("%s/+/%s", c.Repository, c.Sha1)
	if path != "" {
		url += ":" + path
	}
	return url, nil
}

func (r *codesearchRedirect) diff(c1, c2 common.GitCommit) (string, error) {
	if !areIdenticalRepositories(c1, c2) {
		return "", errNotIdenticalRepositories
	}

	url, ok := codesearchMapping[c1.Host]
	if !ok {
		return "", errNotSupportedRepository
	}

	url = fmt.Sprintf(
		"%s%s/+/%s...%s",
		url, c1.Repository, c1.Sha1, c2.Sha1)
	return url, nil
}

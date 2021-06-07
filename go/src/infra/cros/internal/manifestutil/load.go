// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package manifestutil

import (
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"

	"infra/cros/internal/gerrit"
	"infra/cros/internal/repo"

	"go.chromium.org/luci/common/errors"
)

var (
	xmlFileRegex = regexp.MustCompile(`.*\.xml$`)
)

func loadManifestTree(file string, getFile func(file string) ([]byte, error), recur bool) (map[string]*repo.Manifest, error) {
	results := make(map[string]*repo.Manifest)

	data, err := getFile(file)
	if err != nil {
		return nil, errors.Annotate(err, "failed to open and read %s", file).Err()
	}
	manifest := &repo.Manifest{}
	if err = xml.Unmarshal(data, manifest); err != nil {
		return nil, errors.Annotate(err, "failed to unmarshal %s", file).Err()
	}
	manifest.XMLName = xml.Name{}
	results[filepath.Base(file)] = manifest

	// Recursively fetch manifests listed in "include" elements.
	if recur {
		for _, incl := range manifest.Includes {
			subResults, err := loadManifestTree(incl.Name, getFile, recur)
			if err != nil {
				return nil, err
			}
			for k, v := range subResults {
				results[filepath.Join(filepath.Dir(incl.Name), k)] = v
			}
		}
	}
	return results, nil
}

func loadManifest(file string, getFile func(file string) ([]byte, error), mergeManifests bool) (*repo.Manifest, error) {
	manifestMap, err := loadManifestTree(file, getFile, mergeManifests)
	if err != nil {
		return nil, err
	}
	if mergeManifests {
		return repo.MergeManifests(filepath.Base(file), &manifestMap)
	}
	manifest, exists := manifestMap[filepath.Base(file)]
	if !exists {
		return nil, fmt.Errorf("failed to read %s", file)
	}

	return manifest, nil
}

func loadManifestFromFile(file string, mergeManifests bool) (*repo.Manifest, error) {
	getFile := func(f string) ([]byte, error) {
		path := filepath.Join(filepath.Dir(file), f)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, errors.Annotate(err, "failed to open and read %s", path).Err()
		}
		return data, nil
	}
	return loadManifest(filepath.Base(file), getFile, mergeManifests)
}

func loadManifestFromGitiles(ctx context.Context, authedClient *http.Client, host, project, branch, file string, mergeManifests bool) (*repo.Manifest, error) {
	getFile := loadFromGitilesInnerFunc(ctx, authedClient, host, project, branch, file)
	return loadManifest(file, getFile, mergeManifests)
}

// LoadManifestTree loads the manifest at the given file path into
// a Manifest struct. It also loads all included manifests.
// Returns a map mapping manifest filenames to file contents.
func LoadManifestTreeFromFile(file string) (map[string]*repo.Manifest, error) {
	getFile := func(f string) ([]byte, error) {
		path := filepath.Join(filepath.Dir(file), f)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, errors.Annotate(err, "failed to open and read %s", path).Err()
		}
		return data, nil
	}
	return loadManifestTree(filepath.Base(file), getFile, true)
}

// LoadManifestFromFile loads the manifest at the given file into a
// Manifest struct.
func LoadManifestFromFile(file string) (*repo.Manifest, error) {
	return loadManifestFromFile(file, false)
}

// LoadManifestFromFileWithIncludes loads the manifest at the given files but also
// calls MergeManifests to resolve includes.
func LoadManifestFromFileWithIncludes(file string) (*repo.Manifest, error) {
	return loadManifestFromFile(file, true)
}

// LoadManifestFromFileRaw loads the manifest at the given file and returns
// the file contents as a byte array.
func LoadManifestFromFileRaw(file string) ([]byte, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Annotate(err, "failed to open and read %s", file).Err()
	}
	return data, nil
}

func loadFromGitilesInnerFunc(ctx context.Context, authedClient *http.Client, host, project, branch, file string) (getFile func(file string) ([]byte, error)) {
	return func(f string) ([]byte, error) {
		path := filepath.Join(filepath.Dir(file), f)
		data, err := gerrit.DownloadFileFromGitiles(ctx, authedClient, host, project, branch, f)
		if err != nil {
			return nil, errors.Annotate(err, "failed to open and read %s", path).Err()
		}
		// If the manifest file just contains another file name, it's a symlink
		// and we need to follow it.
		for xmlFileRegex.MatchString(data) {
			data, err = gerrit.DownloadFileFromGitiles(ctx, authedClient, host, project, branch, data)
			if err != nil {
				return nil, errors.Annotate(err, "failed to open and read %s", path).Err()
			}
		}
		return []byte(data), nil
	}
}

// LoadManifestTree loads the manifest from the specified remote location into
// a Manifest struct. It also loads all included manifests.
// Returns a map mapping manifest filenames to file contents.
func LoadManifestTreeFromGitiles(ctx context.Context, authedClient *http.Client, host, project, branch, file string) (map[string]*repo.Manifest, error) {
	getFile := loadFromGitilesInnerFunc(ctx, authedClient, host, project, branch, file)
	return loadManifestTree(file, getFile, true)
}

// LoadManifestFromGitiles loads the manifest from the specified remote location
// using the Gitiles API.
func LoadManifestFromGitiles(ctx context.Context, authedClient *http.Client, host, project, branch, file string) (*repo.Manifest, error) {
	return loadManifestFromGitiles(ctx, authedClient, host, project, branch, file, false)
}

// LoadManifestFromGitilesWithIncludes loads the manifest from the specified remote location
// using the Gitiles API and also calls MergeManifests to resolve includes.
func LoadManifestFromGitilesWithIncludes(ctx context.Context, authedClient *http.Client, host, project, branch, file string) (*repo.Manifest, error) {
	return loadManifestFromGitiles(ctx, authedClient, host, project, branch, file, true)
}

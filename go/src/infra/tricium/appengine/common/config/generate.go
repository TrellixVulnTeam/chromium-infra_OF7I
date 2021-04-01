// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.chromium.org/luci/common/errors"

	admin "infra/tricium/api/admin/v1"
	tricium "infra/tricium/api/v1"
)

// Generate generates a Tricium workflow based on the provided configs and
// paths to analyze.
//
// Previously a workflow would be a tree including isolators. Now, all
// functions should be recipe-based analyzers and workflows just have a
// list of workers, filtered based on path filters, although in practice,
// most path filtering will happen inside recipes.
func Generate(sc *tricium.ServiceConfig, pc *tricium.ProjectConfig,
	files []*tricium.Data_File, gitRef, gitURL string) (*admin.Workflow, error) {
	mergedFunctions, err := mergeSelectedFunctions(sc, pc)
	if err != nil {
		return nil, errors.Annotate(err, "failed to merge function definitions").Err()
	}
	var workers []*admin.Worker
	functions := []*tricium.Function{}
	for _, s := range pc.Selections {
		f, ok := mergedFunctions[s.Function]
		if !ok {
			return nil, errors.Annotate(err, "failed to lookup project function").Err()
		}
		functions = append(functions, f)
		shouldInclude, err := includeFunction(f, files)
		if err != nil {
			return nil, errors.Annotate(err, "failed include function check").Err()
		}
		if shouldInclude {
			w, err := createWorker(s, sc, f, gitRef, gitURL)
			if err != nil {
				// The worker will fail creation if it is a legacy
				// (non-recipe-based) analyzer. Just skip such workers to help
				// ease the transition.
				continue
			}
			workers = append(workers, w)
		}
	}

	return &admin.Workflow{
		Workers:               workers,
		BuildbucketServerHost: sc.BuildbucketServerHost,
		Functions:             functions,
	}, nil
}

// includeFunction checks if an analyzer should be included based on paths.
//
// The paths are checked against the path filters included for the function. If
// there are no path filters or no paths, then the function is included
// without further checking. With both paths and path filters, there needs to
// be at least one path match for the function to be included.
//
// The path filter only applies to the last part of the path.
//
// Also, path filters are only provided for analyzers; analyzer functions are
// always included regardless of path matching.
func includeFunction(f *tricium.Function, files []*tricium.Data_File) (bool, error) {
	if f.Type == tricium.Function_ISOLATOR || len(files) == 0 || len(f.PathFilters) == 0 {
		return true, nil
	}
	for _, file := range files {
		p := file.Path
		for _, filter := range f.PathFilters {
			ok, err := filepath.Match(filter, filepath.Base(p))
			if err != nil {
				return false, errors.Reason("failed to check path filter %s for path %s", filter, p).Err()
			}
			if ok {
				return true, nil
			}
		}
	}
	return false, nil
}

// createWorker creates a worker from the provided function, selection and
// service config.
//
// The provided function is assumed to be verified.
func createWorker(s *tricium.Selection, sc *tricium.ServiceConfig, f *tricium.Function,
	gitRef, gitURL string) (*admin.Worker, error) {
	i := tricium.LookupImplForPlatform(f, s.Platform) // If verified, there should be an Impl.
	p := tricium.LookupPlatform(sc, s.Platform)       // If verified, the platform should be known.
	// The separator character for worker names is underscore, so this
	// character shouldn't appear in function or platform names. This
	// is also checked in config validation.
	workerName := fmt.Sprintf("%s_%s", s.Function, s.Platform)
	if strings.Contains(s.Function, "_") || strings.Contains(s.Platform.String(), "_") {
		return nil, errors.Reason("invalid name when making worker %q", workerName).Err()
	}
	w := &admin.Worker{
		Name:                workerName,
		Needs:               f.Needs,
		Provides:            f.Provides,
		NeedsForPlatform:    i.NeedsForPlatform,
		ProvidesForPlatform: i.ProvidesForPlatform,
		RuntimePlatform:     i.RuntimePlatform,
		Dimensions:          p.Dimensions,
		CipdPackages:        i.CipdPackages,
	}
	switch ii := i.Impl.(type) {
	case *tricium.Impl_Recipe:
		w.Impl = &admin.Worker_Recipe{Recipe: ii.Recipe}
	case nil:
		return nil, errors.Reason("missing Impl when constructing worker %s", w.Name).Err()
	default:
		return nil, errors.Reason("Impl.Impl has unexpected type %T", ii).Err()
	}
	return w, nil
}

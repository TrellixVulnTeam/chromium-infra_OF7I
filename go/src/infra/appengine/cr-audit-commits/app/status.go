// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"strconv"

	"context"

	ds "go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/templates"
)

// Status displays recent information regarding the audit app's activity on
// the monitored repositories.
func Status(rctx *router.Context) {
	ctx, req, resp := rctx.Context, rctx.Request, rctx.Writer
	refURL := req.FormValue("refUrl")
	if refURL == "" {
		refURL = "https://chromium.googlesource.com/chromium/src.git/+/master"
	}
	cfg, refState, err := loadConfig(ctx, refURL)
	if err != nil {
		args := templates.Args{
			"RuleMap": RuleMap,
			"Error":   fmt.Sprintf("Unknown repository %s", refURL),
		}
		templates.MustRender(ctx, resp, "pages/status.html", args)
		return
	}
	nCommits := 10
	n := req.FormValue("n")
	if n != "" {
		nCommits, err = strconv.Atoi(n)
		if err != nil {
			// We are swallowing the error on purpose,
			// rather than fail, use default.
			nCommits = 10
		}
	}
	commits := []*RelevantCommit{}
	if refState.LastRelevantCommit != "" {
		rc := &RelevantCommit{
			CommitHash:  refState.LastRelevantCommit,
			RefStateKey: ds.KeyForObj(ctx, refState),
		}

		err = ds.Get(ctx, rc)
		if err != nil {
			handleError(ctx, err, refURL, refState, resp)
			return
		}

		commits, err = lastXRelevantCommits(ctx, rc, nCommits)
		if err != nil {
			handleError(ctx, err, refURL, refState, resp)
			return
		}
	}

	allRefStates := &[]*RefState{}
	err = ds.GetAll(ctx, ds.NewQuery("RefState").Order("-LastRelevantCommitTime").Limit(5), allRefStates)
	if err != nil {
		handleError(ctx, err, refURL, refState, resp)
		return
	}
	args := templates.Args{
		"Commits":          commits,
		"LastRelevant":     refState.LastRelevantCommit,
		"LastRelevantTime": refState.LastRelevantCommitTime,
		"LastScanned":      refState.LastKnownCommit,
		"LastScannedTime":  refState.LastKnownCommitTime,
		"RefUrl":           refURL,
		"RefConfig":        cfg,
		"RefStates":        allRefStates,
	}
	templates.MustRender(ctx, resp, "pages/status.html", args)
}

func handleError(ctx context.Context, err error, refURL string, refState *RefState, resp http.ResponseWriter) {
	logging.WithError(err).Errorf(ctx, "Getting status of repo %s, for revision %s", refURL, refState.LastRelevantCommit)
	http.Error(resp, "Getting status failed. See log for details.", 502)
}

func lastXRelevantCommits(ctx context.Context, rc *RelevantCommit, x int) ([]*RelevantCommit, error) {
	current := rc
	result := []*RelevantCommit{rc}
	for counter := 1; counter < x; counter++ {
		if current.PreviousRelevantCommit == "" {
			break
		}

		current = &RelevantCommit{
			CommitHash:  current.PreviousRelevantCommit,
			RefStateKey: rc.RefStateKey,
		}
		err := ds.Get(ctx, current)
		if err != nil {
			return nil, err
		}
		result = append(result, current)
	}
	return result, nil
}

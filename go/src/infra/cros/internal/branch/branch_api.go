// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package branch

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"time"

	gerritapi "github.com/andygrunwald/go-gerrit"
	"go.chromium.org/luci/common/errors"
	"golang.org/x/sync/errgroup"
)

// GerritProjectBranch contains all the details for creating a new Gerrit branch
// based on an existing one.
type GerritProjectBranch struct {
	GerritURL string
	Project   string
	Branch    string
	SrcRef    string
}

func qpsToPeriod(qps float64) time.Duration {
	if qps <= 0 {
		// some very generous default duration
		LogOut("Got qps %v, <= 0. Using a default duration instead.", qps)
		return time.Second * 10
	}
	periodSec := float64(time.Second) / qps
	return time.Duration(int64(periodSec))
}

func createRemoteBranch(authedClient *http.Client, b GerritProjectBranch, dryRun bool) error {
	agClient, err := gerritapi.NewClient(b.GerritURL, authedClient)
	if err != nil {
		return fmt.Errorf("failed to create Gerrit client: %v", err)
	}
	if dryRun {
		return nil
	}
	bi, resp, err := agClient.Projects.CreateBranch(b.Project, b.Branch, &gerritapi.BranchInput{Revision: b.SrcRef})
	defer resp.Body.Close()
	if err != nil {
		body, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			// shouldn't happen
			return err2
		}
		if resp.StatusCode == http.StatusConflict {
			// Branch already exists, so there's nothing to do.
			return nil
		}
		return errors.Annotate(err, "failed to create branch. Got response %v and branch info %v", string(body), bi).Err()
	}
	return nil
}

// CreateRemoteBranchesAPI creates a bunch of branches on remote Gerrit instances
// for the specified inputs using the Gerrit API.
func CreateRemoteBranchesAPI(authedClient *http.Client, branches []GerritProjectBranch, dryRun bool, gerritQPS float64) error {
	if dryRun {
		LogOut("Dry run (no --push): would create remote branches for %v Gerrit repos", len(branches))
		return nil
	}
	LogOut("Creating remote branches for %v Gerrit repos. This will take a few minutes, since otherwise Gerrit would throttle us.", len(branches))
	var g errgroup.Group
	throttle := time.Tick(qpsToPeriod(gerritQPS))

	var createCount int64
	for _, b := range branches {
		<-throttle
		b := b
		g.Go(func() error {
			err := createRemoteBranch(authedClient, b, dryRun)
			if err != nil {
				return err
			}
			count := atomic.AddInt64(&createCount, 1)
			if count%10 == 0 {
				LogOut("Created %v of %v remote branches", count, len(branches))
			}
			return nil
		})
	}
	err := g.Wait()
	LogOut("Successfully created %v of %v remote branches", atomic.LoadInt64(&createCount), len(branches))
	return err
}

// CheckSelfGroupMembership checks if the authenticated user is in the given
// group on the given Gerrit host. It returns a bool indicating whether or
// not that's the case, or an error if the lookup fails.
func CheckSelfGroupMembership(authedClient *http.Client, gerritURL, expectedGroup string) (bool, error) {
	agClient, err := gerritapi.NewClient(gerritURL, authedClient)
	if err != nil {
		return false, fmt.Errorf("failed to create Gerrit client: %v", err)
	}
	groups, resp, err := agClient.Accounts.ListGroups("self")
	defer resp.Body.Close()
	if err != nil {
		return false, errors.Annotate(err, "failed to get list of Gerrit groups for self").Err()
	}
	for _, g := range *groups {
		if g.Name == expectedGroup {
			return true, nil
		}
	}
	return false, nil
}

// Copyright 2018 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitstore

import (
	"fmt"
	"regexp"
	"strings"

	"infra/appengine/crosskylabadmin/app/config"
	"infra/libs/skylab/inventory"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/common/proto/gitiles"
	"golang.org/x/net/context"
)

// InventoryStore exposes skylab inventory data in git.
//
// Call InventoryStore.Refresh() to obtain the initial inventory data. After making
// modifications to the inventory, call InventoryStore.Commit().
// Call InventoryStore.Refresh() again if you want to use the object beyond a
// InventoryStore.Commit(), to re-validate the store.
type InventoryStore struct {
	*inventory.Lab
	*inventory.Infrastructure

	gerritC     gerrit.GerritClient
	gitilesC    gitiles.GitilesClient
	latestFiles map[string]string
	latestSHA1  string
}

// NewInventoryStore returns a new InventoryStore.
//
// The returned store is not refreshed, hence all inventory data is empty.
func NewInventoryStore(gerritC gerrit.GerritClient, gitilesC gitiles.GitilesClient) *InventoryStore {
	store := &InventoryStore{
		gerritC:  gerritC,
		gitilesC: gitilesC,
	}
	store.clear()
	return store
}

// IsEmptyErr returns true if the given error is because of am empty
// commit.
func IsEmptyErr(e error) bool {
	_, ok := e.(emptyError)
	return ok
}

type emptyError struct {
	err error
}

func (e emptyError) Error() string {
	return e.err.Error()
}

func (g *InventoryStore) commitInfrastructure(ctx context.Context, changed map[string]string, toDelete map[string]bool, path string) error {
	is, err := inventory.WriteInfrastructureToString(g.Infrastructure)
	if err != nil {
		return errors.Annotate(err, "inventory store commit").Err()
	}
	if is != g.latestFiles[path] {
		changed[path] = is
	}
	toDelete[path] = false
	return nil
}

func (g *InventoryStore) commitDuts(ctx context.Context, changed map[string]string, toDelete map[string]bool) error {
	for _, dut := range g.Lab.GetDuts() {
		h := dut.GetCommon().GetHostname()
		p := InvPathForDut(h)
		oneDutLab := &inventory.Lab{
			Duts: []*inventory.DeviceUnderTest{dut},
		}
		ls, err := inventory.WriteLabToString(oneDutLab)
		if err != nil {
			return errors.Annotate(err, "inventory store commit").Err()
		}
		if ls != g.latestFiles[p] {
			changed[p] = ls
		}
		toDelete[p] = false
	}
	return nil
}

// CommitAll commits the current inventory data in the store to git, replacing Commit().
//
// Successful Commit() invalidates the data cached in InventoryStore().
// To continue using the store, call Refresh() again
// TODO(xixuan): rename this to Commit() after per-file inventory is landed and tested.
func (g *InventoryStore) commitAll(ctx context.Context, reason string) (string, error) {
	if g.latestSHA1 == "" {
		return "", errors.New("can not commit invalid store")
	}

	ic := config.Get(ctx).Inventory
	changed := make(map[string]string)

	toDelete := make(map[string]bool, len(g.latestFiles))
	for p := range g.latestFiles {
		toDelete[p] = true
	}

	g.commitDuts(ctx, changed, toDelete)
	g.commitInfrastructure(ctx, changed, toDelete, ic.InfrastructureDataPath)

	// For removal
	for p, delete := range toDelete {
		if delete {
			changed[p] = ""
		}
	}

	if len(changed) == 0 {
		return "", emptyError{errors.New("inventory store commit: nothing to commit")}
	}

	cn, err := commitFileContents(ctx, g.gerritC, ic.Project, ic.Branch, g.latestSHA1, reason, changed)
	if err != nil {
		return "", errors.Annotate(err, "inventory store commit").Err()
	}

	u, err := changeURL(ic.GerritHost, ic.Project, cn)
	if err != nil {
		return "", errors.Annotate(err, "inventory store commit").Err()
	}

	// Successful commit implies our refreshed data is not longer current, so
	// the store cache is invalid.
	g.clear()
	return u, err
}

// regexp for chromeos15/chromeos15-XXX.textpb
const crosPrefix = `chromeos`
const invRegexp = `data/skylab/chromeos.*\/.*-.*\.textpb`

var invRe = regexp.MustCompile(invRegexp)

// InvPathForDut generates the relative path to the inventory git repo for a given DUT.
// e.g. data/skylab/chromeos6/chromeos6-***.textpb
func InvPathForDut(hostname string) string {
	comps := strings.Split(hostname, "-")
	var path string
	if len(comps) == 0 || !strings.HasPrefix(comps[0], crosPrefix) {
		// Keep chromeos as prefix for regular expression.
		path = "chromeos-misc"
	} else {
		path = comps[0]
	}
	return fmt.Sprintf("data/skylab/%s/%s.textpb", path, hostname)
}

func validateInvPathForDut(p string) bool {
	return invRe.MatchString(p)
}

// Commit commits the current inventory data in the store to git.
//
// Successful Commit() invalidates the data cached in InventoryStore().
// To continue using the store, call Refresh() again.
// TODO(xixuan): remove this after per-file inventory is landed and tested.
func (g *InventoryStore) Commit(ctx context.Context, reason string) (string, error) {
	ic := config.Get(ctx).Inventory
	if ic.GetMultifile() {
		return g.commitAll(ctx, reason)
	}

	if g.latestSHA1 == "" {
		return "", errors.New("can not commit invalid store")
	}

	changed := make(map[string]string)

	ls, err := inventory.WriteLabToString(g.Lab)
	if err != nil {
		return "", errors.Annotate(err, "inventory store commit").Err()
	}
	if ls != g.latestFiles[ic.LabDataPath] {
		changed[ic.LabDataPath] = ls
	}

	is, err := inventory.WriteInfrastructureToString(g.Infrastructure)
	if err != nil {
		return "", errors.Annotate(err, "inventory store commit").Err()
	}
	if is != g.latestFiles[ic.InfrastructureDataPath] {
		changed[ic.InfrastructureDataPath] = is
	}

	if len(changed) == 0 {
		return "", emptyError{errors.New("inventory store commit: nothing to commit")}
	}

	cn, err := commitFileContents(ctx, g.gerritC, ic.Project, ic.Branch, g.latestSHA1, reason, changed)
	if err != nil {
		return "", errors.Annotate(err, "inventory store commit").Err()
	}

	u, err := changeURL(ic.GerritHost, ic.Project, cn)
	if err != nil {
		return "", errors.Annotate(err, "inventory store commit").Err()
	}

	// Successful commit implies our refreshed data is not longer current, so
	// the store cache is invalid.
	g.clear()
	return u, err
}

// RefreshAll populates all device data in the store from git, replacing Refresh().
// TODO(xixuan): rename this to Refresh() after per-file inventory is landed and tested.
func (g *InventoryStore) refreshAll(ctx context.Context) (rerr error) {
	defer func() {
		if rerr != nil {
			g.clear()
		}
	}()

	ic := config.Get(ctx).Inventory
	var err error
	g.latestSHA1, err = fetchLatestSHA1(ctx, g.gitilesC, ic.Project, ic.Branch)
	if err != nil {
		return errors.Annotate(err, "inventory store refresh").Err()
	}

	pmap := map[string]bool{
		ic.InfrastructureDataPath: true,
	}

	g.latestFiles, err = fetchAllFromGitiles(ctx, g.gitilesC, ic.Project, g.latestSHA1, pmap)
	if err != nil {
		return errors.Annotate(err, "inventory store refresh").Err()
	}

	// Parse infrastructure
	data, ok := g.latestFiles[ic.InfrastructureDataPath]
	if !ok {
		return errors.New("No infrastructure data in inventory")
	}
	g.Infrastructure = &inventory.Infrastructure{}
	if err := inventory.LoadInfrastructureFromString(data, g.Infrastructure); err != nil {
		return errors.Annotate(err, "inventory store refresh").Err()
	}

	g.Lab = &inventory.Lab{}
	for p, data := range g.latestFiles {
		if pmap[p] {
			continue
		}
		var oneDutLab inventory.Lab
		if err := inventory.LoadLabFromString(data, &oneDutLab); err != nil {
			return errors.Annotate(err, "cannot dump text").Err()
		}
		g.Lab.Duts = append(g.Lab.Duts, oneDutLab.Duts...)
	}
	return nil
}

// Refresh populates inventory data in the store from git.
// TODO(xixuan): remove this after per-file inventory is landed and tested.
func (g *InventoryStore) Refresh(ctx context.Context) (rerr error) {
	ic := config.Get(ctx).Inventory
	if ic.GetMultifile() {
		return g.refreshAll(ctx)
	}

	defer func() {
		if rerr != nil {
			g.clear()
		}
	}()

	// TODO(pprabhu) Replace these checks with config validation.
	if ic.LabDataPath == "" {
		return errors.New("no lab data file path provided in config")
	}
	if ic.InfrastructureDataPath == "" {
		return errors.New("no infrastructure data file path provided in config")
	}

	var err error
	g.latestSHA1, err = fetchLatestSHA1(ctx, g.gitilesC, ic.Project, ic.Branch)
	if err != nil {
		return errors.Annotate(err, "inventory store refresh").Err()
	}

	g.latestFiles, err = fetchFilesFromGitiles(ctx, g.gitilesC, ic.Project, g.latestSHA1, []string{ic.LabDataPath, ic.InfrastructureDataPath})
	if err != nil {
		return errors.Annotate(err, "inventory store refresh").Err()
	}

	data, ok := g.latestFiles[ic.LabDataPath]
	if !ok {
		return errors.New("No lab data in inventory")
	}
	g.Lab = &inventory.Lab{}
	if err := inventory.LoadLabFromString(data, g.Lab); err != nil {
		return errors.Annotate(err, "inventory store refresh").Err()
	}

	data, ok = g.latestFiles[ic.InfrastructureDataPath]
	if !ok {
		return errors.New("No infrastructure data in inventory")
	}
	g.Infrastructure = &inventory.Infrastructure{}
	if err := inventory.LoadInfrastructureFromString(data, g.Infrastructure); err != nil {
		return errors.Annotate(err, "inventory store refresh").Err()
	}

	return nil
}

func (g *InventoryStore) clear() {
	g.Lab = nil
	g.Infrastructure = nil
	g.latestSHA1 = ""
	g.latestFiles = make(map[string]string)
}

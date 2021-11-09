// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"strings"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/common/logging/teelogger"
	configpb "go.chromium.org/luci/common/proto/config"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/config/cfgclient"

	"infra/tools/migrator"
)

// ScanConfig is passed from the main migrator binary to the plugin process.
type ScanConfig struct {
	Squeaky bool
	Clean   bool
	Reapply bool
}

// scanner implements the "scan" command scanning.
type scanner struct {
	factory    migrator.InstantiateAPI
	projectDir ProjectDir
	cfg        ScanConfig
}

// scannedProject is a project being scanned.
type scannedProject struct {
	ctx    context.Context      // has logging and reports sink
	done   func(removeLog bool) // called to finalize the log
	pb     *configpb.Project    // an entry from projects.cfg
	api    migrator.API         // a project-specific instance of the plugin impl
	remote migrator.Project     // an instance of RemoteProject
}

// repoRef is a repo:ref pair.
type repoRef struct {
	repo string
	ref  string
}

// scan calls FindProblems to scan the remote project for errors.
func (p *scannedProject) scan() {
	defer func() {
		if rcov := recover(); rcov != nil {
			// TODO(iannucci): report this better
			logging.Errorf(p.ctx, "fatal error: %s", rcov)
			logging.Errorf(p.ctx, string(debug.Stack()))
			p.remote.Report("FATAL_ERROR", fmt.Sprintf("%s", rcov))
		}
	}()
	p.api.FindProblems(p.ctx, p.remote)
}

// applyFix calls ApplyFix to modify the local configs checkout.
func (p *scannedProject) applyFix(repo *repo) {
	defer func() {
		if rcov := recover(); rcov != nil {
			// TODO(iannucci): report this better
			logging.Errorf(p.ctx, "fatal error: %s", rcov)
			logging.Errorf(p.ctx, string(debug.Stack()))
			p.remote.Report("APPLY_FIX_FAILURE", "Failed to run ApplyFix")
		}
	}()
	p.api.ApplyFix(p.ctx, repo.localProject(p.ctx, p.pb.Id))
}

// hasActionableReports returns true if we need to checkout and fix the project.
func (p *scannedProject) hasActionableReports() bool {
	return HasActionableReports(p.ctx)
}

// repoRef returns repo:ref pair where project configs are hosted.
func (p *scannedProject) repoRef() repoRef {
	return repoRef{
		repo: p.pb.GetGitilesLocation().GetRepo(),
		ref:  p.pb.GetGitilesLocation().GetRef(),
	}
}

// finalize collects the per-project reports and closes the log.
func (p *scannedProject) finalize(dump *migrator.ReportDump) {
	numReports := dump.UpdateFrom(DumpReports(p.ctx))
	if numReports > 0 {
		logging.Warningf(p.ctx, "%d reports", numReports)
	}
	p.done(!p.hasActionableReports())
}

type multiProjectCheckout struct {
	ctx        context.Context   // logs into all relevant projects at once
	checkoutID string            // identifier of this checkout on disk
	repoRef    repoRef           // repo:ref pair
	projs      []*scannedProject // projects that are located there
}

// checkoutID figures out how to name a git checkout given projects there.
func checkoutID(r repoRef, projs []*scannedProject) string {
	if len(projs) == 1 {
		return projs[0].pb.Id + "-project"
	}
	// Convert the repo name into an ID using some heuristic to make it less ugly.
	id := strings.TrimPrefix(r.repo, "https://")
	id = strings.ReplaceAll(id, "infra/config", "")
	id = strings.ReplaceAll(id, ".googlesource.com", "")
	id = strings.ReplaceAll(id, ".", "-")
	id = strings.ReplaceAll(id, "/", "-")
	id = strings.Trim(id, "-")
	return id + "-repo"
}

// multiLoggingContext makes a context that logs into all project contexts.
func multiLoggingContext(ctx context.Context, projs []*scannedProject) context.Context {
	// Clear any default logging in `ctx`, we want to tee only to `projs`.
	ctx = logging.SetFactory(ctx, nil)

	// Collects all per-project logging sinks.
	var out []logging.Factory
	for _, proj := range projs {
		if f := logging.GetFactory(proj.ctx); f != nil {
			out = append(out, f)
		}
	}

	// Log to all of them at once.
	return teelogger.Use(ctx, out...)
}

// report writes the report to all projects at once.
func (co *multiProjectCheckout) report(tag, description string, opts ...migrator.ReportOption) {
	for _, proj := range co.projs {
		proj.remote.Report(tag, description, opts...)
	}
}

// projectPBs returns a list with project protos from projects.cfg.
func (co *multiProjectCheckout) projectPBs() []*configpb.Project {
	out := make([]*configpb.Project, len(co.projs))
	for i, p := range co.projs {
		out[i] = p.pb
	}
	return out
}

// run implements the "scan" command logic.
func (s *scanner) run(ctx context.Context) error {
	// Note: we use this formulation because the GetProjects API excludes vital
	// information on how to check out the project from Git (specifically, the
	// ref and path are omitted).
	projectPB := &configpb.ProjectsCfg{}
	err := cfgclient.Get(ctx, "services/luci-config", "projects.cfg", cfgclient.ProtoText(projectPB), nil)
	if err != nil {
		return errors.Annotate(err, "loading luci-config projects.cfg").Err()
	}

	// Prep all projects for visiting.
	projectsToVisit, err := s.filterProjects(projectPB.Projects)
	if err != nil {
		return errors.Annotate(err, "when applying projects_re filter").Err()
	}
	projs := make([]*scannedProject, len(projectsToVisit))
	for i, projPB := range projectsToVisit {
		projCtx, doneCB := s.perProjectContext(ctx, projPB.Id)
		projs[i] = &scannedProject{
			ctx:    projCtx,
			done:   doneCB,
			pb:     projPB,
			api:    s.factory(),
			remote: RemoteProject(projCtx, projPB.Id),
		}
	}

	// Group projects by git repo:ref they are hosted in, this would allow us to
	// figure out how to name git checkouts on disk.
	projectsByRepo := make(map[repoRef][]*scannedProject)
	for _, proj := range projs {
		projectsByRepo[proj.repoRef()] = append(projectsByRepo[proj.repoRef()], proj)
	}
	checkouts := make([]*multiProjectCheckout, 0, len(projectsByRepo))
	seenIDs := stringset.New(0)
	for repoRef, projs := range projectsByRepo {
		id := checkoutID(repoRef, projs)
		checkouts = append(checkouts, &multiProjectCheckout{
			ctx:        multiLoggingContext(ctx, projs),
			checkoutID: id,
			repoRef:    repoRef,
			projs:      projs,
		})
		if !seenIDs.Add(id) {
			panic(fmt.Sprintf("collision in the checkout IDs on %q vs %v, change the heuristic in checkoutID", id, seenIDs.ToSortedSlice()))
		}
	}

	// Discover if we need to fix anything. This operates on the remote configs
	// and safe to do in parallel. It updates the reports stored in per-project
	// contexts.
	parallel.WorkPool(32, func(ch chan<- func() error) {
		for _, proj := range projs {
			proj := proj
			ch <- func() error {
				proj.scan()
				return nil
			}
		}
	})

	// Visit all checkouts and either fix problems there or clean them up.
	parallel.WorkPool(16, func(ch chan<- func() error) {
		for _, checkout := range checkouts {
			checkout := checkout
			ch <- func() error {
				allGood := true
				for _, proj := range checkout.projs {
					if proj.hasActionableReports() {
						allGood = false
						break
					}
				}
				if allGood {
					s.doCheckoutCleanup(checkout)
				} else {
					s.doCheckoutFixups(checkout)
				}
				return nil
			}
		}
	})

	// Finalize all per-project logs and collect the reports.
	allReports := &migrator.ReportDump{}
	for _, proj := range projs {
		proj.finalize(allReports)
	}

	// Write the reports out as CSV.
	scanOut, err := os.Create(s.projectDir.ScanReportPath())
	if err != nil {
		return err
	}
	defer scanOut.Close()
	return allReports.WriteToCSV(scanOut)
}

// filterProjects returns a list of projects that pass `projects_re` filter.
func (s *scanner) filterProjects(projs []*configpb.Project) ([]*configpb.Project, error) {
	cfg, err := s.projectDir.LoadConfigFile()
	if err != nil {
		return nil, err
	}

	if len(cfg.ProjectsRe) == 0 {
		return projs, nil
	}

	regs := make([]*regexp.Regexp, len(cfg.ProjectsRe))
	for i, str := range cfg.ProjectsRe {
		str = "^(" + str + ")$"
		regs[i], err = regexp.Compile(str)
		if err != nil {
			return nil, errors.Annotate(err, "when compiling %q", str).Err()
		}
	}

	var filtered []*configpb.Project
	for _, proj := range projs {
		match := false
		for _, reg := range regs {
			if reg.MatchString(proj.Id) {
				match = true
				break
			}
		}
		if match {
			filtered = append(filtered, proj)
		}
	}

	return filtered, nil
}

// perProjectContext prepares a context with project logs and reports sink.
func (s *scanner) perProjectContext(ctx context.Context, projID string) (out context.Context, done func(removeLog bool)) {
	ctx = InitReportSink(ctx)
	ctx = (&gologger.LoggerConfig{
		Out: os.Stderr,
		// We pick a more helpful format here which includes the project.
		// The gory details of the filename are recorded to the .log file.
		Format: fmt.Sprintf("%%{color}[%%{level:.1s}|%s]%%{color:reset} %%{message}", projID),
	}).Use(ctx)

	logFile, err := os.Create(s.projectDir.ProjectLog(projID))
	if err != nil {
		// should never happen, let it fly
		panic(errors.Annotate(err, "opening logfile").Err())
	}

	ctx = teelogger.Use(
		ctx,
		(&gologger.LoggerConfig{
			Out:    logFile,
			Format: `[%{level:.1s} %{shortfile}] %{message}`,
		}).NewLogger,
	)

	return ctx, func(removeLog bool) {
		logFile.Close()
		if removeLog {
			os.Remove(logFile.Name())
		}
	}
}

func (s *scanner) doCheckoutCleanup(co *multiProjectCheckout) {
	checkoutDir := s.projectDir.CheckoutDir(co.checkoutID)
	if s.cfg.Clean {
		if _, err := os.Stat(checkoutDir); !os.IsNotExist(err) {
			logging.Warningf(co.ctx, "Cleaning checkout.")
			if err := os.RemoveAll(checkoutDir); err != nil {
				logging.Errorf(co.ctx, "Failed to remove repo: %s", err)
			}
		}
		return
	}

	if _, err := os.Stat(checkoutDir); err == nil {
		logging.Warningf(co.ctx, "No reports found; This checkout can be removed.")
		logging.Warningf(co.ctx, "Pass `-clean` to do this automatically.")
	}
}

func (s *scanner) doCheckoutFixups(co *multiProjectCheckout) {
	r := &repo{
		projectDir: s.projectDir,
		checkoutID: co.checkoutID,
		projects:   co.projectPBs(),
	}

	newCheckout, err := r.initialize(co.ctx, co.repoRef.repo, co.repoRef.ref)
	if err != nil {
		logging.Errorf(co.ctx, "Failed to checkout repo: %s", err)
		co.report("REPO_CREATION_FAILURE", "Failed to checkout/update repo")
		return
	}

	if !newCheckout && s.cfg.Squeaky && s.cfg.Clean {
		if err := r.reset(co.ctx); err != nil {
			logging.Errorf(co.ctx, "Failed to reset the repo: %s", err)
			co.report("REPO_RESET_FAILURE", "Failed to reset the repo")
			return
		}
		newCheckout = true
	}

	for _, proj := range co.projs {
		if newCheckout || s.cfg.Reapply {
			proj.applyFix(r)
		} else if !newCheckout {
			logging.Infof(proj.ctx, "checkout already exists, skipping ApplyFix (pass -re-apply to run anyway).")
		}
	}
}

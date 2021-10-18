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

	projectsToVisit, err := s.filterProjects(projectPB.Projects)
	if err != nil {
		return errors.Annotate(err, "when applying projects_re filter").Err()
	}

	allReports := &migrator.ReportDump{}

	err = parallel.WorkPool(8, func(ch chan<- func() error) {
		for _, projPB := range projectsToVisit {
			projPB := projPB
			ch <- func() (err error) {
				inst := s.factory()

				s.perProjectContext(ctx, projPB, func(ctx context.Context) bool {
					defer func() {
						numReports := allReports.UpdateFrom(DumpReports(ctx))
						if numReports > 0 {
							logging.Warningf(ctx, "%d reports", numReports)
						}
					}()

					proj := RemoteProject(ctx, projPB.Id)
					s.scanProject(ctx, inst, proj)

					if !HasActionableReports(ctx) {
						s.doRepoCleanup(ctx, projPB.Id)
						return true
					}

					// Otherwise create it, and maybe ApplyFix.
					repo, newCheckout := s.doRepoCreation(ctx, inst, projPB, proj)
					if repo != nil {
						if newCheckout || s.cfg.Reapply {
							s.doApplyFix(ctx, inst, proj, repo)
						} else if !newCheckout {
							logging.Infof(ctx, "checkout already exists, skipping ApplyFix (pass -re-apply to run anyway).")
						}
					}
					return false
				})

				return nil
			}
		}
	})
	if err != nil {
		// Nothing above should return an error.
		panic(err)
	}

	scanOut, err := os.Create(s.projectDir.ReportPath())
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

// perProjectContext calls the callback with a per-project context.
//
// The callback should return true to delete the project's log file.
func (s *scanner) perProjectContext(ctx context.Context, projPB *configpb.Project, cb func(ctx context.Context) bool) {
	ctx = InitReportSink(ctx)
	ctx = (&gologger.LoggerConfig{
		Out: os.Stderr,
		// We pick a more helpful format here which includes the project.
		// The gorey details of the filename are recorded to the .log file.
		Format: fmt.Sprintf("%%{color}[%%{level:.1s}|%s]%%{color:reset} %%{message}", projPB.Id),
	}).Use(ctx)

	logFile, err := os.Create(s.projectDir.ProjectLog(projPB.Id))
	if err != nil {
		// should never happen, let it fly
		panic(errors.Annotate(err, "opening logfile").Err())
	}
	defer logFile.Close()
	ctx = teelogger.Use(
		ctx,
		(&gologger.LoggerConfig{
			Out:    logFile,
			Format: `[%{level:.1s} %{shortfile}] %{message}`,
		}).NewLogger,
	)

	if removeLog := cb(ctx); removeLog {
		logFile.Close()
		os.Remove(logFile.Name())
	}
}

func (s *scanner) scanProject(ctx context.Context, inst migrator.API, proj migrator.Project) {
	defer func() {
		if rcov := recover(); rcov != nil {
			// TODO(iannucci): report this better
			logging.Errorf(ctx, "fatal error: %s", rcov)
			logging.Errorf(ctx, string(debug.Stack()))
			proj.Report("FATAL_ERROR", fmt.Sprintf("%s", rcov))
		}
	}()

	inst.FindProblems(ctx, proj)
}

func (r *scanner) doApplyFix(ctx context.Context, inst migrator.API, reporter migrator.Reportable, repo migrator.Repo) {
	defer func() {
		if rcov := recover(); rcov != nil {
			// TODO(iannucci): report this better
			logging.Errorf(ctx, "fatal error: %s", rcov)
			logging.Errorf(ctx, string(debug.Stack()))
			reporter.Report("APPLY_FIX_FAILURE", "Failed to run ApplyFix")
		}
	}()
	inst.ApplyFix(ctx, repo)
}

func (s *scanner) doRepoCreation(ctx context.Context, inst migrator.API, projPB *configpb.Project, reporter migrator.Reportable) (repo migrator.Repo, newCheckout bool) {
	if s.cfg.Squeaky && s.cfg.Clean {
		if err := os.RemoveAll(s.projectDir.ProjectRepo(projPB.Id)); err != nil && !os.IsNotExist(err) {
			logging.Errorf(ctx, "Failed to clean repo, creation may fail: %s", err)
		}
	}
	repo, newCheckout, err := CreateOrLoadRepo(ctx, s.projectDir, projPB.Id, projPB)
	if err != nil {
		logging.Errorf(ctx, "Failed to checkout repo: %s", err)
		reporter.Report("REPO_CREATION_FAILURE", "Failed to checkout/update repo")
		return
	}
	return
}

func (s *scanner) doRepoCleanup(ctx context.Context, projID string) {
	checkoutDir := s.projectDir.ProjectRepo(projID)
	if s.cfg.Clean {
		if _, err := os.Stat(checkoutDir); !os.IsNotExist(err) {
			logging.Warningf(ctx, "Cleaning checkout.")
			if err := os.RemoveAll(checkoutDir); err != nil {
				logging.Errorf(ctx, "Failed to remove repo: %s", err)
			}
		}
		return
	}
	if _, err := os.Stat(checkoutDir); err == nil {
		logging.Warningf(ctx, "No reports found; This checkout can be removed.")
		logging.Warningf(ctx, "Pass `-clean` to do this automatically.")
	}
}

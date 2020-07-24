// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/common/logging/teelogger"
	configpb "go.chromium.org/luci/common/proto/config"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/config/cfgclient"

	"infra/tools/migrator"
	"infra/tools/migrator/plugsupport"
)

func cmdScan(opts cmdBaseOptions) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "scan",
		ShortDesc: "Scans the current state of the world and checks out non-compliant projects.",
		LongDesc: `Scans current state of data in luci-config.

This command must be run within a migrator project. The scan will run all LUCI
projects through the project's plugin. Note that this scans the state of the
files in the luci-config service, NOT the state of the files in the migrator
project.

If the plugin's 'FindProblems' function is undefined OR it makes any Report
calls, this will ensure that the project is checked out locally on disk. If
FindProblems is defined and does NOT make any Report calls, this will inform
you that the checkout can be removed (pass '-clean' to automatically delete
them).

If scan does a new checkout, and the plugin defines ApplyFix, it will be invoked
once on the checked-out project.

If a checkout already exists on disk, this will NOT attempt to update it; It's
recommended to use standard git tooling to pull/rebase/etc. If you really want
a new checkout, you can delete the checked-out project and run 'scan' again
to get a fresh top-of-tree version.
`,

		CommandRun: func() subcommands.CommandRun {
			ret := cmdScanImpl{}
			ret.initFlags(opts)
			ret.Flags.BoolVar(&ret.squeaky, "squeaky", false,
				"If set in conjunction with `clean`, will checkout all repos from scratch.")
			ret.Flags.BoolVar(&ret.clean, "clean", false,
				"If set, will automatically delete project checkouts which have no reported problems.")

			ret.Flags.BoolVar(&ret.reapply, "re-apply", false,
				"If set, will re-run ApplyFix, even if no new checkout was made.")
			return &ret
		},
	}
}

type cmdScanImpl struct {
	cmdBase

	projectDir plugsupport.ProjectDir
	squeaky    bool
	clean      bool
	reapply    bool
}

func (r *cmdScanImpl) positionalRange() (min, max int) { return 0, 0 }

func (r *cmdScanImpl) validateFlags(ctx context.Context, positionals []string, env subcommands.Env) error {
	if r.squeaky && !r.clean {
		return errors.New("you can't be squeaky without being clean! (pass -clean flag)")
	}

	wd, err := os.Getwd()
	if err != nil {
		return errors.Annotate(err, "getting working directory").Err()
	}
	r.projectDir, err = plugsupport.FindProjectRoot(wd)
	return err
}

func (r *cmdScanImpl) scanProject(ctx context.Context, inst migrator.API, proj migrator.Project) {
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

func (r *cmdScanImpl) dumpReport(ctx context.Context, reports *plugsupport.ReportDump) error {
	outFile, err := os.Create(r.projectDir.ReportPath())
	if err != nil {
		return err
	}
	defer outFile.Close()

	cw := csv.NewWriter(outFile)
	defer cw.Flush()

	// Header
	cw.Write([]string{
		"Project", "ConfigFile", "Tag", "Problem", "Metadata",
	})

	reports.Iterate(func(key migrator.ReportID, reports []*migrator.Report) bool {
		for _, report := range reports {
			cw.Write(report.ToCSVRow())
		}
		return false
	})

	cw.Flush()
	return cw.Error()
}

// Callback should return true to delete the project's log file.
func (r *cmdScanImpl) perProjectContext(ctx context.Context, projPB *configpb.Project, cb func(ctx context.Context) bool) {
	ctx = plugsupport.InitReportSink(ctx)
	ctx = (&gologger.LoggerConfig{
		Out: os.Stderr,
		// We pick a more helpful format here which includes the project.
		// The gorey details of the filename are recorded to the .log file.
		Format: fmt.Sprintf("%%{color}[%%{level:.1s}|%s]%%{color:reset} %%{message}", projPB.Id),
	}).Use(ctx)

	logFile, err := os.Create(r.projectDir.ProjectLog(projPB.Id))
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

func (r *cmdScanImpl) doRepoCreation(ctx context.Context, inst migrator.API, projPB *configpb.Project, reporter migrator.Reportable) (repo migrator.Repo, newCheckout bool) {
	if r.squeaky && r.clean {
		if err := os.RemoveAll(r.projectDir.ProjectRepo(projPB.Id)); err != nil && !os.IsNotExist(err) {
			logging.Errorf(ctx, "Failed to clean repo, creation may fail: %s", err)
		}
	}
	repo, newCheckout, err := plugsupport.CreateRepo(ctx, r.projectDir, projPB)
	if err != nil {
		logging.Errorf(ctx, "Failed to checkout repo: %s", err)
		reporter.Report("REPO_CREATION_FAILURE", "Failed to checkout/update repo")
		return
	}
	return
}

func (r *cmdScanImpl) doApplyFix(ctx context.Context, inst migrator.API, reporter migrator.Reportable, repo migrator.Repo) {
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

func (r *cmdScanImpl) doRepoCleanup(ctx context.Context, projID string) {
	checkoutDir := r.projectDir.ProjectRepo(projID)
	if r.clean {
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

func (r *cmdScanImpl) execute(ctx context.Context) error {
	return withPlugin(ctx, r.projectDir, func(factory migrator.InstantiateAPI) error {
		// Note; we use this formulation because the GetProjects API excludes vital
		// information on how to check out the project from Git (specifically, the
		// ref and path are omitted).
		projectPB := &configpb.ProjectsCfg{}
		err := cfgclient.Get(ctx, "services/luci-config", "projects.cfg", cfgclient.ProtoText(projectPB), nil)
		if err != nil {
			panic(errors.Annotate(err, "loading luci-config projects.cfg").Err())
		}

		allReports := &plugsupport.ReportDump{}

		err = parallel.WorkPool(8, func(ch chan<- func() error) {
			for _, projPB := range projectPB.Projects {
				projPB := projPB
				ch <- func() (err error) {
					inst := factory()

					r.perProjectContext(ctx, projPB, func(ctx context.Context) bool {
						defer func() {
							numReports := allReports.Update(plugsupport.DumpReports(ctx))
							if numReports > 0 {
								logging.Warningf(ctx, "%d reports", numReports)
							}
						}()

						proj := plugsupport.RemoteProject(ctx, projPB.Id)
						r.scanProject(ctx, inst, proj)

						// No reports from scan? Clean up the repo if it exists.
						if !plugsupport.HasReports(ctx) {
							r.doRepoCleanup(ctx, projPB.Id)
							return true
						}

						// Otherwise create it, and maybe ApplyFix.
						repo, newCheckout := r.doRepoCreation(ctx, inst, projPB, proj)
						if repo != nil {
							if newCheckout || r.reapply {
								r.doApplyFix(ctx, inst, proj, repo)
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

		return r.dumpReport(ctx, allReports)
	})
}

func (r *cmdScanImpl) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return r.doContextExecute(a, r, args, env)
}

package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"

	igerrit "infra/cros/internal/gerrit"
	"infra/cros/internal/testplan"
	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"

	"go.chromium.org/chromiumos/config/go/test/plan"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	luciflag "go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

var logCfg = gologger.LoggerConfig{
	Out: os.Stderr,
}

func errToCode(a subcommands.Application, err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", a.GetName(), err)
		return 1
	}

	return 0
}

func app(authOpts auth.Options) *cli.Application {
	return &cli.Application{
		Name:    "test_plan",
		Title:   "A tool to work with SourceTestPlan protos in DIR_METADATA files.",
		Context: logCfg.Use,
		Commands: []*subcommands.Command{
			cmdRelevantPlans(authOpts),
			cmdValidate,

			authcli.SubcommandInfo(authOpts, "auth-info", false),
			authcli.SubcommandLogin(authOpts, "auth-login", false),
			authcli.SubcommandLogout(authOpts, "auth-logout", false),

			subcommands.CmdHelp,
		},
	}
}

func cmdRelevantPlans(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "relevant-plans -cl CL1 [-cl CL2] -out OUTPUT",
		ShortDesc: "Find SourceTestPlans relevant to a set of CLs",
		LongDesc: text.Doc(`
		Find SourceTestPlans relevant to a set of CLs.

		Computes SourceTestPlans from "DIR_METADATA" files and returns plans
		relevant to the files changed by a CL.
	`),
		CommandRun: func() subcommands.CommandRun {
			r := &relevantPlansRun{}
			r.authFlags = authcli.Flags{}
			r.authFlags.Register(r.GetFlags(), authOpts)
			r.Flags.Var(luciflag.StringSlice(&r.cls), "cl", text.Doc(`
			CL URL for the patchsets being tested. Must be specified at least once.

			Example: https://chromium-review.googlesource.com/c/chromiumos/platform2/+/123456
		`))
			r.Flags.StringVar(&r.out, "out", "", "Path to the output test plan")

			r.logLevel = logging.Info
			r.Flags.Var(&r.logLevel, "loglevel", text.Doc(`
			Log level, valid options are "debug", "info", "warning", "error". Default is "info".
			`))

			return r
		},
	}
}

type relevantPlansRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	cls       []string
	out       string
	logLevel  logging.Level
}

func (r *relevantPlansRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	return errToCode(a, r.run(ctx))
}

// getChangeRevs parses each of rawCLURLs and returns a ChangeRev.
func getChangeRevs(ctx context.Context, authedClient *http.Client, rawCLURLs []string) ([]*igerrit.ChangeRev, error) {
	changeRevs := make([]*igerrit.ChangeRev, len(rawCLURLs))

	for i, cl := range rawCLURLs {
		changeRevKey, err := igerrit.ParseCLURL(cl)
		if err != nil {
			return nil, err
		}

		changeRev, err := igerrit.GetChangeRev(
			ctx, authedClient, changeRevKey.ChangeNum, changeRevKey.Revision, changeRevKey.Host,
		)
		if err != nil {
			return nil, err
		}

		changeRevs[i] = changeRev
	}

	return changeRevs, nil
}

// writePlans writes each of plans to a textproto file. The first plan is in a
// file named "relevant_plan_1.textpb", the second is in
// "relevant_plan_2.textpb", etc.
//
// TODO(b/182898188): Consider making a message to hold multiple SourceTestPlans
// instead of writing multiple files.
func writePlans(ctx context.Context, plans []*plan.SourceTestPlan, outPath string) error {
	logging.Infof(ctx, "writing output to %s", outPath)

	err := os.MkdirAll(outPath, os.ModePerm)
	if err != nil {
		return err
	}

	for i, plan := range plans {
		outFile, err := os.Create(path.Join(outPath, fmt.Sprintf("relevant_plan_%d.textpb", i)))
		if err != nil {
			return err
		}
		defer outFile.Close()

		err = proto.MarshalText(outFile, plan)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *relevantPlansRun) validateFlags() error {
	if len(r.cls) == 0 {
		return errors.New("-cl must be specified at least once")
	}

	if r.out == "" {
		return errors.New("-out is required")
	}

	return nil
}

func (r *relevantPlansRun) run(ctx context.Context) error {
	if err := r.validateFlags(); err != nil {
		return err
	}

	ctx = logging.SetLevel(ctx, r.logLevel)

	authOpts, err := r.authFlags.Options()
	if err != nil {
		return err
	}

	authedClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts).Client()
	if err != nil {
		return err
	}

	var changeRevs []*igerrit.ChangeRev

	logging.Infof(ctx, "fetching metadata for CLs")

	changeRevs, err = getChangeRevs(ctx, authedClient, r.cls)
	if err != nil {
		return err
	}

	for i, changeRev := range changeRevs {
		logging.Debugf(ctx, "change rev %d: %q", i, changeRev)
	}

	// Use a workdir creation function that returns a tempdir, and removes the
	// entire tempdir on cleanup.
	workdirFn := func() (string, func() error, error) {
		workdir, err := ioutil.TempDir("", "")
		if err != nil {
			return "", nil, err
		}

		return workdir, func() error { return os.RemoveAll((workdir)) }, nil
	}

	plans, err := testplan.FindRelevantPlans(ctx, changeRevs, workdirFn)
	if err != nil {
		return err
	}

	return writePlans(ctx, plans, r.out)
}

var cmdValidate = &subcommands.Command{
	UsageLine: "validate TARGET1 [TARGET2...]",
	ShortDesc: "validate metadata files",
	LongDesc: text.Doc(`
		Validate metadata files.

		Validation logic on "DIR_METADATA" files specific to ChromeOS test planning.
		Note that validation is done on computed metadata, not directly on
		"DIR_METADATA" files; required fields do not need to be explicitly specified in
		all files, as long as they are present on computed targets.

		Each positional argument should be a path to a directory to compute and validate
		metadata for.

		The subcommand returns a non-zero exit code if any of the files is invalid.
	`),
	CommandRun: func() subcommands.CommandRun {
		r := &validateRun{}
		return r
	},
}

type validateRun struct {
	subcommands.CommandRunBase
}

func (r *validateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return errToCode(a, r.run(a, args, env))
}

func (r *validateRun) run(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, r, env)
	mapping, err := dirmd.ReadMapping(ctx, dirmdpb.MappingForm_SPARSE, args...)

	if err != nil {
		return err
	}

	return testplan.ValidateMapping(mapping)
}

func main() {
	opts := chromeinfra.DefaultAuthOptions()
	opts.Scopes = append(opts.Scopes, gerrit.OAuthScope)
	os.Exit(subcommands.Run(app(opts), nil))
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	igerrit "infra/cros/internal/gerrit"
	"infra/cros/internal/testplan"
	"infra/tools/dirmd"
	"net/http"
	"strings"

	"os"

	"github.com/golang/protobuf/jsonpb"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	luciflag "go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/hardcoded/chromeinfra"

	buildpb "go.chromium.org/chromiumos/config/go/build/api"
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
		Title:   "A tool to generate ChromeOS test plans from config in DIR_METADATA files.",
		Context: logCfg.Use,
		Commands: []*subcommands.Command{
			cmdGenerate(authOpts),
			cmdValidate,

			authcli.SubcommandInfo(authOpts, "auth-info", false),
			authcli.SubcommandLogin(authOpts, "auth-login", false),
			authcli.SubcommandLogout(authOpts, "auth-logout", false),

			subcommands.CmdHelp,
		},
	}
}

func cmdGenerate(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "generate -cl CL1 [-cl CL2] -out OUTPUT",
		ShortDesc: "generate a test plan",
		LongDesc: text.Doc(`
		Generate a test plan.

		Computes test config from "DIR_METADATA" files and generates a test plan
		based on the files the patchsets are touching.
	`),
		CommandRun: func() subcommands.CommandRun {
			r := &generateRun{}
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

type generateRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	cls       []string
	out       string
	logLevel  logging.Level
}

func (r *generateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
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

// getBuildSummaryList reads the HEAD BuildSummaryList from
// chromeos/config-internal.
//
// TODO(b/182898188): Return BuildSummaryList specific to given builds, once it
// is available.
func getBuildSummaryList(
	ctx context.Context, authedClient *http.Client,
) (*buildpb.SystemImage_BuildSummaryList, error) {
	buildSummaryListStr, err := igerrit.DownloadFileFromGitiles(
		ctx, authedClient,
		"chrome-internal.googlesource.com",
		"chromeos/config-internal",
		"HEAD",
		"build/generated/build_summary.jsonproto",
	)
	if err != nil {
		return nil, err
	}

	buildSummaryList := &buildpb.SystemImage_BuildSummaryList{}
	if err = jsonpb.Unmarshal(strings.NewReader(buildSummaryListStr), buildSummaryList); err != nil {
		return nil, err
	}

	return buildSummaryList, nil
}

// writeOutputs writes a newline-delimited json file containing outputs to outPath.
func writeOutputs(outputs []*testplan.Output, outPath string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	for _, output := range outputs {
		jsonBytes, err := json.Marshal(output)
		if err != nil {
			return err
		}

		jsonBytes = append(jsonBytes, '\n')

		if _, err = outFile.Write(jsonBytes); err != nil {
			return err
		}
	}

	return nil
}

func (r *generateRun) run(ctx context.Context) error {
	if len(r.cls) == 0 {
		return errors.New("-cl must be specified at least once")
	}

	if r.out == "" {
		return errors.New("-out is required")
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

	logging.Infof(ctx, "fetching metadata for CLs")

	changeRevs, err := getChangeRevs(ctx, authedClient, r.cls)
	if err != nil {
		return err
	}

	logging.Infof(ctx, "fetching build metadata")

	buildSummaryList, err := getBuildSummaryList(ctx, authedClient)
	if err != nil {
		return err
	}

	outputs, err := testplan.Generate(ctx, changeRevs, buildSummaryList)
	if err != nil {
		return err
	}

	return writeOutputs(outputs, r.out)
}

var cmdValidate = &subcommands.Command{
	UsageLine: "validate -root ROOT TARGET1 [TARGET2...]",
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
		r.Flags.StringVar(&r.root, "root", "", "Path to the root directory, typically the root of the ChromeOS checkout")
		return r
	},
}

type validateRun struct {
	subcommands.CommandRunBase
	root string
}

func (r *validateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return errToCode(a, r.run(a, args, env))
}

func (r *validateRun) run(a subcommands.Application, args []string, env subcommands.Env) error {
	if r.root == "" {
		return errors.New("-root is required")
	}

	mapping, err := dirmd.ReadComputed(r.root, args...)
	if err != nil {
		return err
	}

	if err = testplan.ValidateMapping(mapping); err != nil {
		return err
	}

	return nil
}

func main() {
	opts := chromeinfra.DefaultAuthOptions()
	opts.Scopes = append(opts.Scopes, gerrit.OAuthScope)
	os.Exit(subcommands.Run(app(opts), nil))
}

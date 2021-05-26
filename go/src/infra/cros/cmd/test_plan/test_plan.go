package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	protov1 "github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"google.golang.org/protobuf/encoding/protojson"

	buildpb "go.chromium.org/chromiumos/config/go/build/api"
	testpb "go.chromium.org/chromiumos/config/go/test/api"
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
	igerrit "infra/cros/internal/gerrit"
	"infra/cros/internal/testplan"
	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
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
		UsageLine: "generate (-cl CL1 [-cl CL2] | -plan PLAN1 [-plan PLAN2]) -out OUTPUT",
		ShortDesc: "generate a test plan",
		LongDesc: text.Doc(`
		Generate a test plan.

		Computes test config from "DIR_METADATA" files or SourceTestPlan text
		protos and generates a test plan.
	`),
		CommandRun: func() subcommands.CommandRun {
			r := &generateRun{}
			r.authFlags = authcli.Flags{}
			r.authFlags.Register(r.GetFlags(), authOpts)
			r.Flags.Var(luciflag.StringSlice(&r.cls), "cl", text.Doc(`
			CL URL for the patchsets being tested. Must be specified at least once
			if "plan" is not specified.

			Example: https://chromium-review.googlesource.com/c/chromiumos/platform2/+/123456
		`))
			r.Flags.Var(luciflag.StringSlice(&r.planPaths), "plan", text.Doc(`
			Text proto file with a SourceTestPlan to use. Must be specified at least
			once if "cl" is not specified.
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
	planPaths []string
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

// unmarshalProtojson wraps protojson.Unmarshal for v1 protos.
//
// The jsonpb package directly unmarshals v1 protos, but has known issues that
// cause errors when unmarshaling data needed by test_plan (specifically, no
// support for FieldMask, https://github.com/golang/protobuf/issues/745).
//
// Use the protojson package to unmarshal, which fixes this issue. This function
// is a convinience to convert the v1 proto to v2 so it can use protojson.
func unmarshalProtojson(b []byte, m protov1.Message) error {
	return protojson.Unmarshal(b, protov1.MessageV2(m))
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
	if err = unmarshalProtojson([]byte(buildSummaryListStr), buildSummaryList); err != nil {
		return nil, err
	}

	return buildSummaryList, nil
}

func getDutAttributeList(
	ctx context.Context, authedClient *http.Client,
) (*testpb.DutAttributeList, error) {
	dutAttributeListStr, err := igerrit.DownloadFileFromGitiles(
		ctx, authedClient,
		"chrome-internal.googlesource.com",
		"chromeos/config-internal",
		"HEAD",
		"dut_attributes/generated/dut_attributes.jsonproto",
	)
	if err != nil {
		return nil, err
	}

	dutAttributeList := &testpb.DutAttributeList{}
	if err = unmarshalProtojson([]byte(dutAttributeListStr), dutAttributeList); err != nil {
		return nil, err
	}

	return dutAttributeList, nil
}

// writeRules writes a newline-delimited json file containing rules to outPath.
func writeRules(rules []*testpb.CoverageRule, outPath string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	for _, rule := range rules {
		jsonBytes, err := protojson.Marshal(protov1.MessageV2(rule))
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

func (r *generateRun) validateFlags() error {
	if len(r.cls) == 0 && len(r.planPaths) == 0 {
		return errors.New("-cl or -plan must be specified at least once")
	}

	if len(r.cls) > 0 && len(r.planPaths) > 0 {
		return errors.New("-cl and -plan cannot both be specified")
	}

	if r.out == "" {
		return errors.New("-out is required")
	}

	return nil
}

func (r *generateRun) run(ctx context.Context) error {
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

	var plans []*plan.SourceTestPlan

	if len(r.cls) > 0 {
		logging.Infof(ctx, "fetching metadata for CLs")

		changeRevs, err = getChangeRevs(ctx, authedClient, r.cls)
		if err != nil {
			return err
		}
	} else {
		logging.Infof(ctx, "reading plan text protos")

		for _, planPath := range r.planPaths {
			fileBytes, err := os.ReadFile(planPath)
			if err != nil {
				return err
			}

			plan := &plan.SourceTestPlan{}
			if err := protov1.UnmarshalText(string(fileBytes), plan); err != nil {
				return err
			}

			plans = append(plans, plan)
		}
	}

	logging.Infof(ctx, "fetching build metadata")

	buildSummaryList, err := getBuildSummaryList(ctx, authedClient)
	if err != nil {
		return err
	}

	logging.Debugf(ctx, "fetched %d BuildSummaries", len(buildSummaryList.Values))

	logging.Infof(ctx, "fetching dut attributes")

	dutAttributeList, err := getDutAttributeList(ctx, authedClient)
	if err != nil {
		return err
	}

	logging.Debugf(ctx, "fetched dut attributes:\n%s", dutAttributeList)

	rules, err := testplan.Generate(
		ctx, changeRevs, plans, buildSummaryList, dutAttributeList,
	)
	if err != nil {
		return err
	}

	return writeRules(rules, r.out)
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

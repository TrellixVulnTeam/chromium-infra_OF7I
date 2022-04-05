// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"infra/cros/internal/buildplan"
	igerrit "infra/cros/internal/gerrit"
	"infra/cros/internal/manifestutil"
	"infra/cros/internal/shared"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	cros_pb "go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	testplans_pb "go.chromium.org/chromiumos/infra/proto/go/testplans"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	bbproto "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

const (
	buildIrrelevanceConfigPath       = "buildplanconfig/generated/build_irrelevance_config.binaryproto"
	builderConfigsPath               = "generated/builder_configs.binaryproto"
	slimBuildConfigPath              = "buildplanconfig/generated/slim_build_config.binaryproto"
	targetTestRequirementsConfigPath = "testingconfig/generated/target_test_requirements.binaryproto"
)

var (
	unmarshaler = jsonpb.Unmarshaler{AllowUnknownFields: true}
)

func cmdCheckSkip(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "generate-plan --input_json=/path/to/input.json --output_json=/path/to/output.json",
		ShortDesc: "Checks which of the provided BuilderConfigs must be run",
		LongDesc:  "Checks which of the provided BuilderConfigs must be run.",
		CommandRun: func() subcommands.CommandRun {
			c := &checkBuild{}
			c.authFlags = authcli.Flags{}
			c.authFlags.Register(c.GetFlags(), authOpts)
			c.Flags.StringVar(&c.inputJSON, "input_json", "",
				"Path to JSON proto representing a GenerateBuildPlanRequest")
			c.Flags.StringVar(&c.outputJSON, "output_json", "",
				"Path to file to write output GenerateBuildPlanResponse JSON proto")
			c.Flags.StringVar(&c.inputTextPb, "input_text_pb", "",
				"Path to text proto representing a GenerateBuildPlanRequest")
			c.Flags.StringVar(&c.outputTextPb, "output_text_pb", "",
				"Path to file to write output GenerateBuildPlanResponse text proto")
			c.Flags.StringVar(&c.inputBinaryPb, "input_binary_pb", "",
				"Path to binaryproto file representing a GenerateTestPlanRequest")
			c.Flags.StringVar(&c.outputBinaryPb, "output_binary_pb", "",
				"Path to file to write output GenerateTestPlanResponse binaryproto")
			c.Flags.StringVar(&c.manifestFile, "manifest_file", "",
				"Path to local manifest file. If given, will be used instead of default snapshot.xml")
			return c
		}}
}

type fetchConfigResult struct {
	buildIrrelevanceCfg *testplans_pb.BuildIrrelevanceCfg
	slimBuildCfg        *testplans_pb.SlimBuildCfg
	testReqsCfg         *testplans_pb.TargetTestRequirementsCfg
	builderConfigs      *cros_pb.BuilderConfigs
}

func (c *checkBuild) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	flag.Parse()

	req, err := c.readInput()
	if err != nil {
		log.Print(err)
		return 1
	}

	configs, err := c.fetchConfigFromGitiles()
	if err != nil {
		log.Print(err)
		return 2
	}

	changes, err := readGerritChanges(req.GerritChanges)
	if err != nil {
		log.Print(err)
		return 3
	}

	changeRevs, err := c.fetchGerritData(changes)
	if err != nil {
		log.Print(err)
		return 4
	}

	var repoToSrcRoot *map[string]map[string]string
	// If we have a local manifest file provided, use that. Else get it from Gerrit.
	if c.manifestFile == "" {
		gitilesCommit, err := readGitilesCommit(req.GitilesCommit)
		if err != nil {
			log.Print(err)
			return 5
		}

		repoToSrcRoot, err = c.getRepoToSourceRoot(gitilesCommit)
		if err != nil {
			log.Print(err)
			return 6
		}
	} else {
		log.Printf("Reading local manifest from %s", c.manifestFile)
		repoToSrcRootMap, err := manifestutil.GetRepoToRemoteBranchToSourceRootFromFile(c.manifestFile)
		if err != nil {
			log.Print(err)
			return 9
		}
		repoToSrcRoot = &repoToSrcRootMap
	}

	checkBuildersInput := buildplan.CheckBuildersInput{
		Builders:              req.BuilderConfigs,
		Changes:               changes,
		ChangeRevs:            changeRevs,
		RepoToBranchToSrcRoot: *repoToSrcRoot,
		BuildIrrelevanceCfg:   configs.buildIrrelevanceCfg,
		SlimBuildCfg:          configs.slimBuildCfg,
		TestReqsCfg:           configs.testReqsCfg,
		BuilderConfigs:        configs.builderConfigs,
	}
	resp, err := checkBuildersInput.CheckBuilders()
	if err != nil {
		log.Printf("Error checking which builds can be skipped:\n%v", err)
		return 7
	}

	if err = c.writeOutput(resp); err != nil {
		log.Print(err)
		return 8
	}
	return 0
}

type checkBuild struct {
	subcommands.CommandRunBase
	authFlags      authcli.Flags
	inputJSON      string
	outputJSON     string
	inputTextPb    string
	outputTextPb   string
	inputBinaryPb  string
	outputBinaryPb string
	manifestFile   string
}

func nonEmptyCount(strs ...string) int64 {
	var count int64
	for _, str := range strs {
		if len(str) > 0 {
			count++
		}
	}
	return count
}

func (c *checkBuild) readInput() (*cros_pb.GenerateBuildPlanRequest, error) {
	numInputs := nonEmptyCount(c.inputBinaryPb, c.inputTextPb, c.inputJSON)
	if numInputs != 1 {
		return nil, fmt.Errorf("expected 1 input-related flag, found %v", numInputs)
	}
	// use input_binary_pb if it's specified
	if len(c.inputBinaryPb) > 0 {
		inputPb, err := ioutil.ReadFile(c.inputBinaryPb)
		if err != nil {
			return nil, fmt.Errorf("Failed reason input_binary_pb\n%v", err)
		}
		req := &cros_pb.GenerateBuildPlanRequest{}
		if err := proto.Unmarshal(inputPb, req); err != nil {
			return nil, fmt.Errorf("Failed parsing input_binary_pb as proto\n%v", err)
		}
		return req, nil
	}
	if len(c.inputTextPb) > 0 {
		inputBytes, err := ioutil.ReadFile(c.inputTextPb)
		log.Printf("Request is:\n%s", string(inputBytes))
		if err != nil {
			return nil, fmt.Errorf("Failed reading input_text_pb\n%v", err)
		}
		req := &cros_pb.GenerateBuildPlanRequest{}
		if err := proto.UnmarshalText(string(inputBytes), req); err != nil {
			return nil, fmt.Errorf("Couldn't decode %s as a chromiumos.GenerateBuildPlanRequest\n%v", c.inputJSON, err)
		}
		return req, nil
	}
	inputBytes, err := ioutil.ReadFile(c.inputJSON)
	log.Printf("Request is:\n%s", string(inputBytes))
	if err != nil {
		return nil, fmt.Errorf("Failed reading input_json\n%v", err)
	}
	req := &cros_pb.GenerateBuildPlanRequest{}
	if err := unmarshaler.Unmarshal(bytes.NewReader(inputBytes), req); err != nil {
		return nil, fmt.Errorf("Couldn't decode %s as a chromiumos.GenerateBuildPlanRequest\n%v", c.inputJSON, err)
	}
	return req, nil
}

func (c *checkBuild) fetchConfigFromGitiles() (*fetchConfigResult, error) {
	// Create an authenticated client for Gerrit RPCs, then fetch all required CL data from Gerrit.
	ctx := context.Background()
	authOpts, err := c.authFlags.Options()
	if err != nil {
		return nil, err
	}
	authedClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts).Client()
	if err != nil {
		return nil, err
	}
	gerritClient, err := igerrit.NewClient(authedClient)
	if err != nil {
		return nil, err
	}
	m, err := gerritClient.FetchFilesFromGitiles(ctx,
		"chrome-internal.googlesource.com",
		"chromeos/infra/config",
		"main",
		[]string{buildIrrelevanceConfigPath, slimBuildConfigPath, targetTestRequirementsConfigPath, builderConfigsPath})
	if err != nil {
		return nil, err
	}
	buildIrrelevanceConfig := &testplans_pb.BuildIrrelevanceCfg{}
	if err := proto.Unmarshal([]byte((*m)[buildIrrelevanceConfigPath]), buildIrrelevanceConfig); err != nil {
		return nil, fmt.Errorf("Couldn't decode %s as a BuildIrrelevanceCfg\n%v", (*m)[buildIrrelevanceConfigPath], err)
	}
	slimBuildConfig := &testplans_pb.SlimBuildCfg{}
	if err := proto.Unmarshal([]byte((*m)[slimBuildConfigPath]), slimBuildConfig); err != nil {
		return nil, fmt.Errorf("Couldn't decode %s as a SlimBuildCfg\n%v", (*m)[slimBuildConfigPath], err)
	}
	targetTestRequirementsConfig := &testplans_pb.TargetTestRequirementsCfg{}
	if err := proto.Unmarshal([]byte((*m)[targetTestRequirementsConfigPath]), targetTestRequirementsConfig); err != nil {
		return nil, fmt.Errorf("Couldn't decode %s as a TargetTestRequirementsCfg\n%v", (*m)[targetTestRequirementsConfigPath], err)
	}
	builderConfigs := &cros_pb.BuilderConfigs{}
	if err := proto.Unmarshal([]byte((*m)[builderConfigsPath]), builderConfigs); err != nil {
		return nil, fmt.Errorf("Couldn't decode %s as a BuilderConfigs\n%v", (*m)[builderConfigsPath], err)
	}
	// It can be useful to log these when debugging, but they currently add >6 MB of text to the logging
	// log.Printf("Fetched config from Gitiles:\n%s\n\n%s\n\n%s",
	//	proto.MarshalTextString(buildIrrelevanceConfig), proto.MarshalTextString(targetTestRequirementsConfig), proto.MarshalTextString(builderConfigs))
	return &fetchConfigResult{
		buildIrrelevanceCfg: buildIrrelevanceConfig,
		slimBuildCfg:        slimBuildConfig,
		testReqsCfg:         targetTestRequirementsConfig,
		builderConfigs:      builderConfigs,
	}, nil
}

func readGerritChanges(changeBytes []*cros_pb.ProtoBytes) ([]*bbproto.GerritChange, error) {
	changes := make([]*bbproto.GerritChange, 0)
	for i, c := range changeBytes {
		gc := &bbproto.GerritChange{}
		if err := proto.Unmarshal(c.SerializedProto, gc); err != nil {
			return nil, fmt.Errorf("Couldn't decode %s as a GerritChange\n%v", c.String(), err)
		}
		log.Printf("Got GerritChange %d proto:\n%s", i, proto.MarshalTextString(gc))
		changes = append(changes, gc)
	}
	return changes, nil
}

func (c *checkBuild) fetchGerritData(changes []*bbproto.GerritChange) (*igerrit.ChangeRevData, error) {
	// Create an authenticated client for Gerrit RPCs, then fetch all required CL data from Gerrit.
	ctx := context.Background()
	authOpts, err := c.authFlags.Options()
	if err != nil {
		return nil, err
	}
	authedClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts).Client()
	if err != nil {
		return nil, err
	}
	changeIds := make([]igerrit.ChangeRevKey, 0)
	for _, ch := range changes {
		changeIds = append(changeIds, igerrit.ChangeRevKey{Host: ch.Host, ChangeNum: ch.Change, Revision: int32(ch.Patchset)})
	}
	chRevData, err := igerrit.GetChangeRevData(ctx, authedClient, changeIds)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch CL data from Gerrit. "+
			"Note that a NotFound error may indicate authorization issues.\n%v", err)
	}
	return chRevData, nil
}

func readGitilesCommit(gitilesBytes *cros_pb.ProtoBytes) (*bbproto.GitilesCommit, error) {
	gc := &bbproto.GitilesCommit{}
	if err := proto.Unmarshal(gitilesBytes.SerializedProto, gc); err != nil {
		return nil, fmt.Errorf("Couldn't decode %s as a GitilesCommit\n%v", gitilesBytes.String(), err)
	}
	log.Printf("Got GitilesCommit proto:\n%s", proto.MarshalTextString(gc))
	return gc, nil
}

func (c *checkBuild) getRepoToSourceRoot(gc *bbproto.GitilesCommit) (*map[string]map[string]string, error) {
	ctx := context.Background()
	authOpts, err := c.authFlags.Options()
	if err != nil {
		return nil, err
	}
	authedClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts).Client()
	if err != nil {
		return nil, err
	}
	gerritClient, err := igerrit.NewClient(authedClient)
	if err != nil {
		return nil, err
	}
	if gc.Id == "" {
		log.Print("No manifest commit provided. Using 'snapshot' instead.")
		gc.Id = "snapshot"
	}
	ch := make(chan map[string]map[string]string, 1)
	err = shared.DoWithRetry(ctx, shared.LongerOpts, func() error {
		repoToRemoteBranchToSrcRoot, err := manifestutil.GetRepoToRemoteBranchToSourceRootFromGitiles(ctx, gerritClient, gc)
		if err != nil {
			return err
		}
		ch <- repoToRemoteBranchToSrcRoot
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Error with GetRepoToRemoteBranchToSourceRootFromGitiles\n%v", err)
	}
	repoToRemoteBranchToSrcRoot := <-ch

	return &repoToRemoteBranchToSrcRoot, nil
}

func (c *checkBuild) writeOutput(resp *cros_pb.GenerateBuildPlanResponse) error {
	log.Printf("Full output =\n%s", proto.MarshalTextString(resp))

	if len(c.outputJSON) > 0 {
		marshal := &jsonpb.Marshaler{EmitDefaults: true, Indent: "  "}
		jsonOutput, err := marshal.MarshalToString(resp)
		if err != nil {
			return fmt.Errorf("Failed to marshal JSON %v\n%v", resp, err)
		}
		if err = ioutil.WriteFile(c.outputJSON, []byte(jsonOutput), 0644); err != nil {
			return fmt.Errorf("Failed to write output JSON!\n%v", err)
		}
		log.Printf("Wrote JSON output to %s", c.outputJSON)
	}

	if len(c.outputTextPb) > 0 {
		if err := ioutil.WriteFile(c.outputTextPb, []byte(proto.MarshalTextString(resp)), 0644); err != nil {
			return fmt.Errorf("Failed to write output text proto!\n%v", err)
		}
	}

	if len(c.outputBinaryPb) > 0 {
		binaryOutput, err := proto.Marshal(resp)
		if err != nil {
			return fmt.Errorf("Failed to marshal binaryproto %v\n%v", resp, err)
		}
		if err = ioutil.WriteFile(c.outputBinaryPb, binaryOutput, 0644); err != nil {
			return fmt.Errorf("Failed to write output binary proto!\n%v", err)
		}
		log.Printf("Wrote output binary proto to %s", c.outputBinaryPb)
	}

	return nil
}

// GetApplication returns an instance of the build_plan_generator application.
func GetApplication(authOpts auth.Options) *cli.Application {
	return &cli.Application{
		Name: "build_plan_generator",

		Context: func(ctx context.Context) context.Context {
			return ctx
		},

		Commands: []*subcommands.Command{
			authcli.SubcommandInfo(authOpts, "auth-info", false),
			authcli.SubcommandLogin(authOpts, "auth-login", false),
			authcli.SubcommandLogout(authOpts, "auth-logout", false),
			cmdCheckSkip(authOpts),
		},
	}
}

func main() {
	opts := chromeinfra.DefaultAuthOptions()
	opts.Scopes = []string{gerrit.OAuthScope, auth.OAuthScopeEmail}
	app := GetApplication(opts)
	os.Exit(subcommands.Run(app, nil))
}

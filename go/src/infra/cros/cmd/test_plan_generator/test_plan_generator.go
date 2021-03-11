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
	"path"

	"infra/cros/internal/generator"
	igerrit "infra/cros/internal/gerrit"
	"infra/cros/internal/repo"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/testplans"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	bbproto "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

const (
	boardPriorityConfigPath    = "testingconfig/generated/board_priority.binaryproto"
	sourceTreeTestConfigPath   = "testingconfig/generated/source_tree_test_config.binaryproto"
	targetTestRequirementsPath = "testingconfig/generated/target_test_requirements.binaryproto"
)

var (
	unmarshaler = jsonpb.Unmarshaler{AllowUnknownFields: true}
)

func cmdGenTestPlan(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "gen-test-plan --input_json=/path/to/input.json --output_json=/path/to/output.json",
		ShortDesc: "Generates a test plan",
		LongDesc:  "Generates a test plan",
		CommandRun: func() subcommands.CommandRun {
			c := &getTestPlanRun{}
			c.authFlags = authcli.Flags{}
			c.authFlags.Register(c.GetFlags(), authOpts)
			c.Flags.StringVar(&c.inputJSON, "input_json", "", "Path to JSON proto representing a GenerateTestPlanRequest")
			c.Flags.StringVar(&c.outputJSON, "output_json", "", "Path to file to write output GenerateTestPlanResponse JSON proto")
			c.Flags.StringVar(&c.inputBinaryPb, "input_binary_pb", "", "Path to binaryproto file representing a GenerateTestPlanRequest")
			c.Flags.StringVar(&c.outputBinaryPb, "output_binary_pb", "", "Path to file to write output GenerateTestPlanResponse binaryproto")
			c.Flags.StringVar(&c.localConfigDir, "local_config_dir", "", "Path to an infra/config checkout, to be used rather than origin HEAD")
			c.Flags.StringVar(&c.manifestFile, "manifest_file", "", "Path to local manifest file. If given, will be used instead of default snapshot.xml")
			return c
		}}
}

func (c *getTestPlanRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	flag.Parse()

	req, err := c.readInput()
	if err != nil {
		log.Print(err)
		return 1
	}

	var boardPriorityList *testplans.BoardPriorityList
	var sourceTreeConfig *testplans.SourceTreeTestCfg
	var testReqsConfig *testplans.TargetTestRequirementsCfg
	if c.localConfigDir == "" {
		boardPriorityList, sourceTreeConfig, testReqsConfig, err = c.fetchConfigFromGitiles()
	} else {
		boardPriorityList, sourceTreeConfig, testReqsConfig, err = c.readLocalConfigFiles()
	}
	if err != nil {
		log.Print(err)
		return 2
	}

	bbBuilds, err := readBuildbucketBuilds(req.BuildbucketProtos)
	if err != nil {
		log.Print(err)
		return 3
	}

	gerritChanges, err := readGerritChanges(req.GerritChanges)
	if err != nil {
		log.Print(err)
		return 8
	}

	changeRevs, err := c.fetchGerritData(gerritChanges)
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
		repoToSrcRootMap, err := repo.GetRepoToRemoteBranchToSourceRootFromManifestFile(c.manifestFile)
		if err != nil {
			log.Print(err)
			return 9
		}
		repoToSrcRoot = &repoToSrcRootMap
	}

	testPlan, err := generator.CreateTestPlan(testReqsConfig, sourceTreeConfig, boardPriorityList, bbBuilds, gerritChanges, changeRevs, *repoToSrcRoot)
	if err != nil {
		log.Printf("Error creating test plan:\n%v", err)
		return 7
	}

	if err = c.writeOutput(testPlan); err != nil {
		log.Print(err)
		return 8
	}
	return 0
}

type getTestPlanRun struct {
	subcommands.CommandRunBase
	authFlags      authcli.Flags
	inputJSON      string
	outputJSON     string
	inputBinaryPb  string
	outputBinaryPb string
	localConfigDir string
	manifestFile   string
}

func (c *getTestPlanRun) readInput() (*testplans.GenerateTestPlanRequest, error) {
	// use input_binary_pb if it's specified
	if len(c.inputBinaryPb) > 0 {
		inputPb, err := ioutil.ReadFile(c.inputBinaryPb)
		if err != nil {
			return nil, fmt.Errorf("Failed reason input_binary_pb\n%v", err)
		}
		req := &testplans.GenerateTestPlanRequest{}
		if err := proto.Unmarshal(inputPb, req); err != nil {
			return nil, fmt.Errorf("Failed parsing input_binary_pb as proto\n%v", err)
		}
		return req, nil
		// otherwise use input_json
	}
	inputBytes, err := ioutil.ReadFile(c.inputJSON)
	if err != nil {
		return nil, fmt.Errorf("Failed reading input_json\n%v", err)
	}
	req := &testplans.GenerateTestPlanRequest{}
	if err := unmarshaler.Unmarshal(bytes.NewReader(inputBytes), req); err != nil {
		return nil, fmt.Errorf("Couldn't decode %s as a GenerateTestPlanRequest\n%v", c.inputJSON, err)
	}
	return req, nil
}

func (c *getTestPlanRun) fetchConfigFromGitiles() (*testplans.BoardPriorityList, *testplans.SourceTreeTestCfg, *testplans.TargetTestRequirementsCfg, error) {
	// Create an authenticated client for Gerrit RPCs, then fetch all required CL data from Gerrit.
	ctx := context.Background()
	authOpts, err := c.authFlags.Options()
	if err != nil {
		return nil, nil, nil, err
	}
	authedClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts).Client()
	if err != nil {
		return nil, nil, nil, err
	}

	m, err := igerrit.FetchFilesFromGitiles(ctx, authedClient,
		"chrome-internal.googlesource.com",
		"chromeos/infra/config",
		"main",
		[]string{boardPriorityConfigPath, sourceTreeTestConfigPath, targetTestRequirementsPath})
	if err != nil {
		return nil, nil, nil, err
	}

	boardPriorityList := &testplans.BoardPriorityList{}
	if err := proto.Unmarshal([]byte((*m)[boardPriorityConfigPath]), boardPriorityList); err != nil {
		return nil, nil, nil, fmt.Errorf("Couldn't decode %s as a BoardPriorityList\n%v", (*m)[boardPriorityConfigPath], err)
	}
	sourceTreeConfig := &testplans.SourceTreeTestCfg{}
	if err := proto.Unmarshal([]byte((*m)[sourceTreeTestConfigPath]), sourceTreeConfig); err != nil {
		return nil, nil, nil, fmt.Errorf("Couldn't decode %s as a SourceTreeTestCfg\n%v", (*m)[sourceTreeTestConfigPath], err)
	}
	testReqsConfig := &testplans.TargetTestRequirementsCfg{}
	if err := proto.Unmarshal([]byte((*m)[targetTestRequirementsPath]), testReqsConfig); err != nil {
		return nil, nil, nil, fmt.Errorf("Couldn't decode %s as a TargetTestRequirementsCfg\n%v", (*m)[targetTestRequirementsPath], err)
	}
	log.Printf("Fetched config from Gitiles:\n%s\n\n%s\n\n%s", proto.MarshalTextString(boardPriorityList),
		proto.MarshalTextString(sourceTreeConfig), proto.MarshalTextString(testReqsConfig))
	return boardPriorityList, sourceTreeConfig, testReqsConfig, nil
}

func (c *getTestPlanRun) readLocalConfigFiles() (*testplans.BoardPriorityList, *testplans.SourceTreeTestCfg, *testplans.TargetTestRequirementsCfg, error) {
	log.Print("--------------------------------------------")
	log.Print("WARNING: Reading config from local dir.")
	log.Print("Be sure that you've run `./regenerate_configs.sh -b` first to generate binaryproto files")
	log.Print("--------------------------------------------")

	bplBytes, err := ioutil.ReadFile(path.Join(c.localConfigDir, boardPriorityConfigPath))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read BoardPriorityList file: %v", err)
	}
	boardPriorityList := &testplans.BoardPriorityList{}
	if err := proto.Unmarshal(bplBytes, boardPriorityList); err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't decode file as BoardPriorityList: %v", err)
	}

	stcBytes, err := ioutil.ReadFile(path.Join(c.localConfigDir, sourceTreeTestConfigPath))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read SourceTreeTestCfg file: %v", err)
	}
	sourceTreeConfig := &testplans.SourceTreeTestCfg{}
	if err := proto.Unmarshal(stcBytes, sourceTreeConfig); err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't decode file as SourceTreeTestCfg: %v", err)
	}

	ttrBytes, err := ioutil.ReadFile(path.Join(c.localConfigDir, targetTestRequirementsPath))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read TargetTestRequirementsCfg file: %v", err)
	}
	testReqsConfig := &testplans.TargetTestRequirementsCfg{}
	if err := proto.Unmarshal(ttrBytes, testReqsConfig); err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't decode file as TargetTestRequirementsCfg: %v", err)
	}
	log.Printf("Read local config:\n%s\n\n%s\n\n%s", proto.MarshalTextString(boardPriorityList),
		proto.MarshalTextString(sourceTreeConfig), proto.MarshalTextString(testReqsConfig))
	return boardPriorityList, sourceTreeConfig, testReqsConfig, nil
}

func readBuildbucketBuilds(bbBuildsBytes []*testplans.ProtoBytes) ([]*bbproto.Build, error) {
	bbBuilds := make([]*bbproto.Build, 0)
	for _, bbBuildBytes := range bbBuildsBytes {
		bbBuild := &bbproto.Build{}
		if err := proto.Unmarshal(bbBuildBytes.SerializedProto, bbBuild); err != nil {
			return bbBuilds, fmt.Errorf("Couldn't decode %s as a Buildbucket Build\n%v", bbBuildBytes.String(), err)
		}
		bbBuilds = append(bbBuilds, bbBuild)
	}
	if len(bbBuilds) > 0 {
		log.Printf("Sample buildbucket proto:\n%s", proto.MarshalTextString(bbBuilds[0]))
	}
	return bbBuilds, nil
}

func readGerritChanges(changesBytes []*testplans.ProtoBytes) ([]*bbproto.GerritChange, error) {
	changes := make([]*bbproto.GerritChange, 0)
	for _, changeBytes := range changesBytes {
		change := &bbproto.GerritChange{}
		if err := proto.Unmarshal(changeBytes.SerializedProto, change); err != nil {
			return changes, fmt.Errorf("Couldn't decode %s as a GerritChange\n%v", changeBytes.String(), err)
		}
		changes = append(changes, change)
	}
	if len(changes) > 0 {
		log.Printf("Sample GerritChange proto:\n%s", proto.MarshalTextString(changes[0]))
	}
	return changes, nil
}

func (c *getTestPlanRun) fetchGerritData(changes []*bbproto.GerritChange) (*igerrit.ChangeRevData, error) {
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
	for _, c := range changes {
		changeIds = append(changeIds, igerrit.ChangeRevKey{Host: c.Host, ChangeNum: c.Change, Revision: int32(c.Patchset)})
	}
	chRevData, err := igerrit.GetChangeRevData(ctx, authedClient, changeIds)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch CL data from Gerrit. "+
			"Note that a NotFound error may indicate authorization issues.\n%v", err)
	}
	return chRevData, nil
}

func readGitilesCommit(gitilesBytes *testplans.ProtoBytes) (*bbproto.GitilesCommit, error) {
	gc := &bbproto.GitilesCommit{}
	if err := proto.Unmarshal(gitilesBytes.SerializedProto, gc); err != nil {
		return nil, fmt.Errorf("Couldn't decode %s as a GitilesCommit\n%v", gitilesBytes.String(), err)
	}
	log.Printf("Got GitilesCommit proto:\n%s", proto.MarshalTextString(gc))
	return gc, nil
}

func (c *getTestPlanRun) getRepoToSourceRoot(gc *bbproto.GitilesCommit) (*map[string]map[string]string, error) {
	ctx := context.Background()
	authOpts, err := c.authFlags.Options()
	if err != nil {
		return nil, err
	}
	authedClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts).Client()
	if err != nil {
		return nil, err
	}
	if gc.Id == "" {
		log.Print("No manifest commit provided. Using 'snapshot' instead.")
		gc.Id = "snapshot"
	}
	repoToRemoteBranchToSrcRoot, err := repo.GetRepoToRemoteBranchToSourceRootFromManifests(ctx, authedClient, gc)
	if err != nil {
		return nil, fmt.Errorf("Error with repo tool call\n%v", err)
	}
	return &repoToRemoteBranchToSrcRoot, nil
}
func (c *getTestPlanRun) writeOutput(tp *testplans.GenerateTestPlanResponse) error {
	if len(c.outputJSON) > 0 {
		marshal := &jsonpb.Marshaler{EmitDefaults: true, Indent: "  "}
		jsonOutput, err := marshal.MarshalToString(tp)
		if err != nil {
			return fmt.Errorf("Failed to marshal JSON %v\n%v", tp, err)
		}
		if err = ioutil.WriteFile(c.outputJSON, []byte(jsonOutput), 0644); err != nil {
			return fmt.Errorf("Failed to write output JSON!\n%v", err)
		}
		log.Printf("Wrote output JSON to %s", c.outputJSON)
	}

	if len(c.outputBinaryPb) > 0 {
		binaryOutput, err := proto.Marshal(tp)
		if err != nil {
			return fmt.Errorf("Failed to marshal binaryproto %v\n%v", tp, err)
		}
		if err = ioutil.WriteFile(c.outputBinaryPb, binaryOutput, 0644); err != nil {
			return fmt.Errorf("Failed to write output binary proto!\n%v", err)
		}
		log.Printf("Wrote output binary proto to %s", c.outputBinaryPb)
	}

	return nil
}

// GetApplication returns an instance of the test_planner application.
func GetApplication(authOpts auth.Options) *cli.Application {
	return &cli.Application{
		Name: "test_planner",

		Context: func(ctx context.Context) context.Context {
			return ctx
		},

		Commands: []*subcommands.Command{
			authcli.SubcommandInfo(authOpts, "auth-info", false),
			authcli.SubcommandLogin(authOpts, "auth-login", false),
			authcli.SubcommandLogout(authOpts, "auth-logout", false),
			cmdGenTestPlan(authOpts),
		},
	}
}

func main() {
	opts := chromeinfra.DefaultAuthOptions()
	opts.Scopes = []string{gerrit.OAuthScope, auth.OAuthScopeEmail}
	app := GetApplication(opts)
	os.Exit(subcommands.Run(app, nil))
}

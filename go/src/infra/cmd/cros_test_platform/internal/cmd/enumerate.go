// Copyright 2098 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"

	"io/ioutil"
	"os"
	"path/filepath"

	"infra/cmd/cros_test_platform/internal/autotest/artifacts"
	"infra/cmd/cros_test_platform/internal/autotest/testspec"
	"infra/cmd/cros_test_platform/internal/enumeration"
	"infra/cmd/cros_test_platform/internal/site"

	"github.com/kr/pretty"
	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/gcloud/gs"
	"go.chromium.org/luci/common/logging"
)

// Enumerate is the `enumerate` subcommand implementation.
var Enumerate = &subcommands.Command{
	UsageLine: "enumerate -input_json /path/to/input.json -output_json /path/to/output.json",
	ShortDesc: "Enumerate tasks to execute for given requests.",
	LongDesc: `Enumerate tasks to execute for given requests.

Step input and output is JSON encoded protobuf defined at
https://chromium.googlesource.com/chromiumos/infra/proto/+/master/src/test_platform/steps/enumeration.proto`,
	CommandRun: func() subcommands.CommandRun {
		c := &enumerateRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path that contains JSON encoded test_platform.steps.EnumerationRequests")
		c.Flags.StringVar(&c.outputPath, "output_json", "", "Path where JSON encoded test_platform.steps.EnumerationResponses should be written.")
		c.Flags.BoolVar(&c.debug, "debug", false, "Print debugging information to stderr.")
		c.Flags.BoolVar(&c.tagged, "tagged", true, "Transitional flag to enable tagged requests and responses.")
		return c
	},
}

type enumerateRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath  string
	outputPath string
	debug      bool

	// TODO(crbug.com/1002941) Completely transition to tagged requests only, once
	// - recipe has transitioned to using tagged requests
	tagged      bool
	orderedTags []string
}

func (c *enumerateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	err := c.innerRun(a, args, env)
	if err != nil {
		fmt.Fprintf(a.GetErr(), "%s\n", err)
	}
	return exitCode(err)
}

func (c *enumerateRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.processCLIArgs(args); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = setupLogging(ctx)

	requests, err := c.readRequests()
	if err != nil {
		return err
	}
	if len(requests) == 0 {
		return errors.Reason("zero requests").Err()
	}

	workspace, err := ioutil.TempDir("", "enumerate")
	if err != nil {
		return err
	}
	defer func() {
		os.RemoveAll(workspace)
	}()

	// TODO(crbug.com/1012863) Properly handle recoverable error in some
	// requests. Currently a catastrophic error in any request immediately
	// aborts all requests.
	tms := make([]*api.TestMetadataResponse, len(requests))
	merr := errors.NewMultiError()
	for i, r := range requests {
		m := r.GetMetadata().GetTestMetadataUrl()
		if m == "" {
			return errors.Reason("empty request.metadata.test_metadata_url in %s", r).Err()
		}
		gsPath := gs.Path(m)

		w, err := ioutil.TempDir(workspace, "request")
		if err != nil {
			return err
		}

		lp, err := c.downloadArtifacts(ctx, gsPath, w)
		if err != nil {
			return err
		}

		tm, writableErr := computeMetadata(lp, w)
		if writableErr != nil && tm == nil {
			// Catastrophic error. There is no reasonable response to write.
			return writableErr
		}
		tms[i] = tm
		merr = append(merr, writableErr)
	}
	var writableErr error
	if merr.First() != nil {
		writableErr = merr
	}

	resps := make([]*steps.EnumerationResponse, len(requests))
	merr = errors.NewMultiError()
	for i := range requests {
		if ts, err := c.enumerate(tms[i], requests[i]); err != nil {
			merr = append(merr, err)
		} else {
			resps[i] = &steps.EnumerationResponse{AutotestInvocations: ts}
		}
	}

	if c.debug {
		c.debugDump(ctx, requests, tms, resps, merr)
	}
	if merr.First() != nil {
		return merr
	}
	return c.writeResponsesWithError(resps, writableErr)
}

func (c *enumerateRun) debugDump(ctx context.Context, reqs []*steps.EnumerationRequest, tms []*api.TestMetadataResponse, resps []*steps.EnumerationResponse, merr errors.MultiError) {
	logging.Infof(ctx, "## Begin debug dump")
	if len(reqs) != len(tms) {
		panic(fmt.Sprintf("%d metadata for %d requests", len(tms), len(reqs)))
	}
	if len(reqs) != len(resps) {
		panic(fmt.Sprintf("%d responses for %d requests", len(resps), len(reqs)))
	}

	logging.Infof(ctx, "Errors encountered...")
	if merr.First() != nil {
		for _, e := range merr {
			if e != nil {
				logging.Infof(ctx, "%s", e)
			}
		}
	}
	logging.Infof(ctx, "###")
	logging.Infof(ctx, "")

	for i := range reqs {
		logging.Infof(ctx, "Request: %s", pretty.Sprint(reqs[i]))
		logging.Infof(ctx, "Response: %s", pretty.Sprint(resps[i]))
		logging.Infof(ctx, "Test Metadata: %s", pretty.Sprint(tms[i]))
		logging.Infof(ctx, "")
	}

	logging.Infof(ctx, "## End debug dump")
}

func (c *enumerateRun) processCLIArgs(args []string) error {
	if len(args) > 0 {
		return errors.Reason("have %d positional args, want 0", len(args)).Err()
	}
	if c.inputPath == "" {
		return errors.Reason("-input_json not specified").Err()
	}
	if c.outputPath == "" {
		return errors.Reason("-output_json not specified").Err()
	}
	return nil
}

func (c *enumerateRun) readRequests() ([]*steps.EnumerationRequest, error) {
	var rs steps.EnumerationRequests
	if err := readRequest(c.inputPath, &rs); err != nil {
		return nil, err
	}
	if !c.tagged {
		return rs.Requests, nil
	}
	ts, reqs := c.unzipTaggedRequests(rs.TaggedRequests)
	c.orderedTags = ts
	return reqs, nil
}

func (c *enumerateRun) unzipTaggedRequests(trs map[string]*steps.EnumerationRequest) ([]string, []*steps.EnumerationRequest) {
	var ts []string
	var rs []*steps.EnumerationRequest
	for t, r := range trs {
		ts = append(ts, t)
		rs = append(rs, r)
	}
	return ts, rs
}

func (c *enumerateRun) writeResponsesWithError(resps []*steps.EnumerationResponse, err error) error {
	r := &steps.EnumerationResponses{
		Responses: resps,
	}
	if c.tagged {
		r.TaggedResponses = c.zipTaggedResponses(c.orderedTags, resps)
	}
	return writeResponseWithError(c.outputPath, r, err)
}

func (c *enumerateRun) zipTaggedResponses(ts []string, rs []*steps.EnumerationResponse) map[string]*steps.EnumerationResponse {
	if len(ts) != len(rs) {
		panic(fmt.Sprintf("got %d responses for %d tags (%s)", len(rs), len(ts), ts))
	}
	m := make(map[string]*steps.EnumerationResponse)
	for i := range ts {
		m[ts[i]] = rs[i]
	}
	return m
}

func (c *enumerateRun) gsPath(requests []*steps.EnumerationRequest) (gs.Path, error) {
	if len(requests) == 0 {
		panic("zero requests")
	}

	m := requests[0].GetMetadata().GetTestMetadataUrl()
	if m == "" {
		return "", errors.Reason("empty request.metadata.test_metadata_url in %s", requests[0]).Err()
	}
	for _, r := range requests[1:] {
		o := r.GetMetadata().GetTestMetadataUrl()
		if o != m {
			return "", errors.Reason("mismatched test metadata URLs: %s vs %s", m, o).Err()
		}
	}
	return gs.Path(m), nil
}

func (c *enumerateRun) downloadArtifacts(ctx context.Context, gsDir gs.Path, workspace string) (artifacts.LocalPaths, error) {
	outDir := filepath.Join(workspace, "artifacts")
	if err := os.Mkdir(outDir, 0750); err != nil {
		return artifacts.LocalPaths{}, errors.Annotate(err, "download artifacts").Err()
	}
	client, err := c.newGSClient(ctx)
	if err != nil {
		return artifacts.LocalPaths{}, errors.Annotate(err, "download artifacts").Err()
	}
	lp, err := artifacts.DownloadFromGoogleStorage(ctx, client, gsDir, outDir)
	if err != nil {
		return artifacts.LocalPaths{}, errors.Annotate(err, "download artifacts").Err()
	}
	return lp, err
}

func (c *enumerateRun) newGSClient(ctx context.Context) (gs.Client, error) {
	t, err := newAuthenticatedTransport(ctx, &c.authFlags)
	if err != nil {
		return nil, errors.Annotate(err, "create GS client").Err()
	}
	return gs.NewProdClient(ctx, t)
}

func (c *enumerateRun) enumerate(tm *api.TestMetadataResponse, request *steps.EnumerationRequest) ([]*steps.EnumerationResponse_AutotestInvocation, error) {
	var ts []*steps.EnumerationResponse_AutotestInvocation

	g, err := enumeration.GetForTests(tm.Autotest, request.TestPlan.Test)
	if err != nil {
		return nil, err
	}
	ts = append(ts, g...)

	ts = append(ts, enumeration.GetForSuites(tm.Autotest, request.TestPlan.Suite)...)
	ts = append(ts, enumeration.GetForEnumeration(request.TestPlan.GetEnumeration())...)
	return ts, nil
}

func computeMetadata(localPaths artifacts.LocalPaths, workspace string) (*api.TestMetadataResponse, error) {
	extracted := filepath.Join(workspace, "extracted")
	if err := os.Mkdir(extracted, 0750); err != nil {
		return nil, errors.Annotate(err, "compute metadata").Err()
	}
	if err := artifacts.ExtractControlFiles(localPaths, extracted); err != nil {
		return nil, errors.Annotate(err, "compute metadata").Err()
	}
	return testspec.Get(extracted)
}

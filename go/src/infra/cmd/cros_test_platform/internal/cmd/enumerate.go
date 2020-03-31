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
	"strings"
	"sync"

	"infra/cmd/cros_test_platform/internal/autotest/artifacts"
	"infra/cmd/cros_test_platform/internal/autotest/testspec"
	"infra/cmd/cros_test_platform/internal/enumeration"
	"infra/cmd/cros_test_platform/internal/site"
	"infra/cmd/cros_test_platform/internal/utils"

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
		c.Flags.BoolVar(&c.debugLogger.enabled, "debug", false, "Print debugging information to stderr.")
		return c
	},
}

type enumerateRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath   string
	outputPath  string
	debugLogger debugLogger
}

func (c *enumerateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.validateArgs(args); err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		c.Flags.Usage()
		return exitCode(err)
	}

	ctx := cli.GetContext(a, c, env)
	ctx = setupLogging(ctx)
	err := c.innerRun(ctx, args)
	if err != nil {
		logApplicationError(ctx, a, err)
	}
	return exitCode(err)
}

func (c *enumerateRun) innerRun(ctx context.Context, args []string) error {
	taggedRequests, err := c.readRequests()
	if err != nil {
		return err
	}
	if len(taggedRequests) == 0 {
		return errors.Reason("zero requests").Err()
	}

	workspace, err := ioutil.TempDir("", "enumerate")
	if err != nil {
		return err
	}
	defer func() {
		os.RemoveAll(workspace)
	}()

	resps := make(map[string]*steps.EnumerationResponse)
	for t, r := range taggedRequests {
		c.debugLogger.LogRequest(ctx, t, r)
		resp, err := c.enumerateOne(ctx, workspace, t, r)
		if err != nil {
			return nil
		}
		resps[t] = resp
		c.debugLogger.LogResponse(ctx, t, resp)
	}

	return writeResponse(c.outputPath, &steps.EnumerationResponses{
		TaggedResponses: resps,
	})
}

// Enumerates one request.
//
// All errors from this function should be treated as infrastructure failure in
// the enumerate step.
// User/caller errors will receive an empty enumeration with errors described in
// its body, rather than an infrastructure error.
func (c *enumerateRun) enumerateOne(ctx context.Context, workspace string, tag string, r *steps.EnumerationRequest) (*steps.EnumerationResponse, error) {
	m := r.GetMetadata().GetTestMetadataUrl()
	if m == "" {
		return &steps.EnumerationResponse{
			AutotestInvocations: []*steps.EnumerationResponse_AutotestInvocation{},
			ErrorSummary:        fmt.Sprintf("empty request.metadata.test_metadata_url for %s", tag),
		}, nil
	}
	gsPath := gs.Path(m)

	w, err := ioutil.TempDir(workspace, "request")
	if err != nil {
		return nil, err
	}

	lp, err := c.downloadArtifacts(ctx, gsPath, w)
	if err != nil {
		if matchesUserErrorPatterns(err) {
			return &steps.EnumerationResponse{
				AutotestInvocations: []*steps.EnumerationResponse_AutotestInvocation{},
				ErrorSummary:        fmt.Sprintf("%s: %s", tag, err.Error()),
			}, nil
		}
		return nil, err
	}

	tm, err := c.computeMetadata(ctx, tag, lp, w)
	if err != nil {
		return nil, err
	}

	ts, errs := c.getEnumeration(ctx, tag, tm, r)
	resp := &steps.EnumerationResponse{}
	if len(ts) > 0 {
		resp.AutotestInvocations = ts
	}
	if errs != nil {
		resp.ErrorSummary = errs.Error()
	}
	return resp, nil
}

func matchesUserErrorPatterns(err error) bool {
	if strings.Contains(err.Error(), "object doesn't exist") {
		return true
	}
	return false
}

func validateEnumeration(ts []*steps.EnumerationResponse_AutotestInvocation) ([]*steps.EnumerationResponse_AutotestInvocation, errors.MultiError) {
	if len(ts) == 0 {
		return ts, errors.NewMultiError(errors.Reason("empty enumeration").Err())
	}

	vts := make([]*steps.EnumerationResponse_AutotestInvocation, 0, len(ts))
	var merr errors.MultiError
	for _, t := range ts {
		if err := validateInvocation(t); err != nil {
			merr = append(merr, errors.Annotate(err, "validate %s", t).Err())
		} else {
			vts = append(vts, t)
		}
	}
	return vts, errorsOrNil(merr)
}

func errorsOrNil(merr errors.MultiError) errors.MultiError {
	if merr.First() != nil {
		return merr
	}
	return nil
}

func validateInvocation(t *steps.EnumerationResponse_AutotestInvocation) error {
	if t.GetTest().GetName() == "" {
		return errors.Reason("empty name").Err()
	}
	if t.GetTest().GetExecutionEnvironment() == api.AutotestTest_EXECUTION_ENVIRONMENT_UNSPECIFIED {
		return errors.Reason("unspecified execution environment").Err()
	}
	return nil
}

func (c *enumerateRun) validateArgs(args []string) error {
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

func (c *enumerateRun) readRequests() (map[string]*steps.EnumerationRequest, error) {
	var rs steps.EnumerationRequests
	if err := readRequest(c.inputPath, &rs); err != nil {
		return nil, err
	}
	return rs.TaggedRequests, nil
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

func (c *enumerateRun) getEnumeration(ctx context.Context, tag string, tm *api.TestMetadataResponse, request *steps.EnumerationRequest) (ts []*steps.EnumerationResponse_AutotestInvocation, errs errors.MultiError) {
	defer func() {
		if errs.First() != nil {
			c.debugLogger.LogErrors(ctx, utils.AnnotateEach(errs, "enumerate %s", tag))
		}
	}()

	g, err := enumeration.GetForTests(tm.Autotest, request.TestPlan.Test)
	if err != nil {
		return nil, errors.NewMultiError(err)
	}
	ts = append(ts, g...)
	ts = append(ts, enumeration.GetForSuites(tm.Autotest, request.TestPlan.Suite)...)
	ts = append(ts, enumeration.GetForEnumeration(request.TestPlan.GetEnumeration())...)
	ts, errs = validateEnumeration(ts)
	return
}

func (c *enumerateRun) computeMetadata(ctx context.Context, tag string, localPaths artifacts.LocalPaths, workspace string) (*api.TestMetadataResponse, error) {
	extracted := filepath.Join(workspace, "extracted")
	if err := os.Mkdir(extracted, 0750); err != nil {
		return nil, errors.Annotate(err, "compute metadata for %s", tag).Err()
	}
	if err := artifacts.ExtractControlFiles(localPaths, extracted); err != nil {
		return nil, errors.Annotate(err, "compute metadata for %s", tag).Err()
	}

	tm, warnings := testspec.Get(extracted)
	if tm == nil {
		panic(fmt.Sprintf("testspec.Get() should always return valid metadata. Got nil for %s", tag))
	}
	c.debugLogger.LogTestMetadata(ctx, tag, tm)
	if warnings.First() != nil {
		c.debugLogger.LogWarnings(ctx, utils.AnnotateEach(warnings, "compute metadata for %s", tag))
	}
	return tm, nil
}

// debugLogger logs various intermediate PODs, only when enabled.
//
// All public methods of debugLogger are atomic: Logs generated by concurrent
// calls to debugLogger methods are guaranteed to not be interspersed.
type debugLogger struct {
	enabled bool
	m       sync.Mutex
}

func (l *debugLogger) LogRequest(ctx context.Context, tag string, req *steps.EnumerationRequest) {
	if !l.enabled {
		return
	}
	l.m.Lock()
	defer l.m.Unlock()

	defer l.debugBlock(ctx, "requests")()
	logging.Infof(ctx, "Request[%s]: %s", tag, pretty.Sprint(req))
}

func (l *debugLogger) LogTestMetadata(ctx context.Context, tag string, tm *api.TestMetadataResponse) {
	if !l.enabled {
		return
	}
	l.m.Lock()
	defer l.m.Unlock()

	defer l.debugBlock(ctx, "metadata")()
	logging.Infof(ctx, "Test Metadata[%s]: %s", tag, pretty.Sprint(tm))
}

func (l *debugLogger) LogResponse(ctx context.Context, tag string, resp *steps.EnumerationResponse) {
	if !l.enabled {
		return
	}
	l.m.Lock()
	defer l.m.Unlock()

	defer l.debugBlock(ctx, "responses")()
	logging.Infof(ctx, "Response[%s]: %s", tag, pretty.Sprint(resp))
}

func (l *debugLogger) LogErrors(ctx context.Context, merr errors.MultiError) {
	// Unlike other debug data, errors are logged regardless of l.enabled
	if merr.First() == nil {
		return
	}
	l.m.Lock()
	defer l.m.Unlock()

	defer l.debugBlock(ctx, "errors")()
	for _, err := range merr {
		logging.Errorf(ctx, "%s", err)
	}
}

func (l *debugLogger) LogWarnings(ctx context.Context, merr errors.MultiError) {
	if !l.enabled {
		return
	}
	if merr.First() == nil {
		return
	}
	l.m.Lock()
	defer l.m.Unlock()

	defer l.debugBlock(ctx, "warnings")()
	for _, err := range merr {
		logging.Errorf(ctx, "%s", err)
	}
}

// Begins a block of debug log.
//
// The returned function should be deferred to close the block.
func (l *debugLogger) debugBlock(ctx context.Context, title string) func() {
	logging.Infof(ctx, "## BEGIN DEBUG LOG [%s]", title)
	return func() {
		logging.Infof(ctx, "## END DEBUG LOG [%s]", title)
	}
}

// Copyright 2098 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
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
	"go.chromium.org/luci/common/data/stringset"
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
		return c
	},
}

type enumerateRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath  string
	outputPath string
	debug      bool
}

func (c *enumerateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	ctx = setupLogging(ctx)
	err := c.innerRun(ctx, args)
	if err != nil {
		logApplicationError(ctx, a, err)
	}
	return exitCode(err)
}

func (c *enumerateRun) innerRun(ctx context.Context, args []string) error {
	if err := c.processCLIArgs(args); err != nil {
		return err
	}

	dl := debugLogger{enabled: c.debug}

	taggedRequests, err := c.readRequests()
	if err != nil {
		return err
	}
	if len(taggedRequests) == 0 {
		return errors.Reason("zero requests").Err()
	}
	dl.LogRequests(ctx, taggedRequests)

	workspace, err := ioutil.TempDir("", "enumerate")
	if err != nil {
		return err
	}
	defer func() {
		os.RemoveAll(workspace)
	}()

	tms := make(map[string]*api.TestMetadataResponse)
	merr := errors.NewMultiError()
	resps := make(map[string]*steps.EnumerationResponse)
	for t, r := range taggedRequests {
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

		tm, errs := computeMetadata(lp, w)
		if errs != nil && tm == nil {
			// Catastrophic error. There is no reasonable response to write.
			return utils.AnnotateEach(errs, "compute metadata for %s", t)
		}
		tms[t] = tm
		merr = append(merr, utils.AnnotateEach(errs, "compute metadata for %s", t)...)

		// TODO(pprabhu) Simplify error handling so we don't have to reset
		// errors variable.
		errs = errors.MultiError{}
		ts, err := c.enumerate(tms[t], r)
		if err != nil {
			errs = append(errs, err)
		}
		ts, ierrs := validateEnumeration(ts)
		if ierrs != nil {
			errs = append(errs, ierrs...)
		}

		resps[t] = &steps.EnumerationResponse{}
		if len(ts) > 0 {
			resps[t].AutotestInvocations = ts
		}
		if errs != nil {
			resps[t].ErrorSummary = errs.Error()
			dl.LogErrors(ctx, utils.AnnotateEach(errs, "enumerate %s", t))
		}
	}

	dl.LogTestMetadata(ctx, tms)
	if merr.First() != nil {
		dl.LogWarnings(ctx, merr)
	}
	dl.LogResponses(ctx, resps)

	return writeResponse(c.outputPath, &steps.EnumerationResponses{
		TaggedResponses: resps,
	})
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

func computeMetadata(localPaths artifacts.LocalPaths, workspace string) (*api.TestMetadataResponse, errors.MultiError) {
	extracted := filepath.Join(workspace, "extracted")
	if err := os.Mkdir(extracted, 0750); err != nil {
		return nil, errors.NewMultiError(errors.Annotate(err, "compute metadata").Err())
	}
	if err := artifacts.ExtractControlFiles(localPaths, extracted); err != nil {
		return nil, errors.NewMultiError(errors.Annotate(err, "compute metadata").Err())
	}
	return testspec.Get(extracted)
}

// debugLogger logs various intermiedate PODs, only when enabled.
//
// All public methods of debugLogger are atomic: Logs generated by concurrent
// calls to debugLogger methods are guaranteed to not be interspersed.
type debugLogger struct {
	enabled bool
	m       sync.Mutex

	requestTags stringset.Set
}

func (l *debugLogger) LogRequests(ctx context.Context, reqs map[string]*steps.EnumerationRequest) {
	if !l.enabled {
		return
	}
	l.m.Lock()
	defer l.m.Unlock()

	defer l.debugBlock(ctx, "requests")()
	l.requestTags = stringset.New(len(reqs))
	for t := range reqs {
		l.requestTags.Add(t)
		logging.Infof(ctx, "Request[%s]: %s", t, pretty.Sprint(reqs[t]))
	}
}

func (l *debugLogger) LogTestMetadata(ctx context.Context, tms map[string]*api.TestMetadataResponse) {
	if !l.enabled {
		return
	}
	l.m.Lock()
	defer l.m.Unlock()

	defer l.debugBlock(ctx, "metadata")()
	ts := stringset.New(len(tms))
	for t := range tms {
		ts.Add(t)
		logging.Infof(ctx, "Test Metadata[%s]: %s", t, pretty.Sprint(tms[t]))
	}
	if ms := l.requestTags.Difference(ts); len(ms) > 0 {
		logging.Warningf(ctx, "No metadata for requests %s", ms)
	}
}

func (l *debugLogger) LogResponses(ctx context.Context, resps map[string]*steps.EnumerationResponse) {
	if !l.enabled {
		return
	}
	l.m.Lock()
	defer l.m.Unlock()

	defer l.debugBlock(ctx, "responses")()
	ts := stringset.New(len(resps))
	for t := range resps {
		ts.Add(t)
		logging.Infof(ctx, "Response[%s]: %s", t, pretty.Sprint(resps[t]))
	}
	if ms := l.requestTags.Difference(ts); len(ms) > 0 {
		logging.Warningf(ctx, "No response for requests %s", ms)
	}
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
		logging.Infof(ctx, "## END DEBUG LOG")
	}
}

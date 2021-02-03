// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maruel/subcommands"
	tlsapi "go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/phosphorus"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/google"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/cros/cmd/phosphorus/internal/autotest/atutil"
	"infra/cros/cmd/phosphorus/internal/tls"
	"infra/libs/lro"
)

// Prejob subcommand: Run a prejob (e.g. provision) against a DUT.
var Prejob = &subcommands.Command{
	UsageLine: "prejob -input_json /path/to/input.json",
	ShortDesc: "Run a prejob against a DUT.",
	LongDesc: `Run a prejob against a DUT.

Provision the DUT via 'autoserv --provision' if desired provisionable labels
do not match the existing ones. Otherwise, reset the DUT via
'autosev --reset'`,
	CommandRun: func() subcommands.CommandRun {
		c := &prejobRun{}
		c.Flags.StringVar(&c.InputPath, "input_json", "", "Path that contains JSON encoded test_platform.phosphorus.PrejobRequest")
		c.Flags.StringVar(&c.OutputPath, "output_json", "", "Path to write JSON encoded test_platform.phosphorus.PrejobResponse to")
		return c
	},
}

type prejobRun struct {
	CommonRun
}

func (c *prejobRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.ValidateArgs(); err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		c.Flags.Usage()
		return 1
	}

	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, args, env); err != nil {
		logApplicationError(ctx, a, err)
		return 1
	}
	return 0
}

const (
	droneTLWPort = 7151
)

func (c *prejobRun) innerRun(ctx context.Context, args []string, env subcommands.Env) error {
	var r phosphorus.PrejobRequest
	if err := ReadJSONPB(c.InputPath, &r); err != nil {
		return err
	}
	if err := validatePrejobRequest(&r); err != nil {
		return err
	}

	if d := google.TimeFromProto(r.Deadline); !d.IsZero() {
		var c context.CancelFunc
		log.Printf("Running with deadline %s (current time: %s)", d, time.Now().UTC())
		ctx, c = context.WithDeadline(ctx, d)
		defer c()
	}

	resp, err := runProvisionSteps(ctx, &r)
	if err != nil {
		return err
	}
	return WriteJSONPB(c.OutputPath, resp)
}

func runProvisionSteps(ctx context.Context, r *phosphorus.PrejobRequest) (*phosphorus.PrejobResponse, error) {
	bt, err := newBackgroundTLS()
	if err != nil {
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_ABORTED}, err
	}
	defer bt.Close()

	resp, err := provisionChromeOSBuild(ctx, bt, r)
	if skipRemainingSteps(resp, err) {
		return resp, err
	}
	resp, err = provisionFirmwareLegacy(ctx, r)
	if skipRemainingSteps(resp, err) {
		return resp, err
	}
	return provisionLacros(ctx, bt, r)
}

func skipRemainingSteps(r *phosphorus.PrejobResponse, err error) bool {
	return err != nil || r.GetState() != phosphorus.PrejobResponse_SUCCEEDED
}

func provisionChromeOSBuild(ctx context.Context, bt *backgroundTLS, r *phosphorus.PrejobRequest) (*phosphorus.PrejobResponse, error) {
	if shouldProvisionChromeOSViaTLS(r) {
		return provisionChromeOSBuildViaTLS(ctx, bt, r)
	}
	return provisionChromeOSBuildLegacy(ctx, r)
}

func shouldProvisionChromeOSViaTLS(r *phosphorus.PrejobRequest) bool {
	e := r.GetConfig().GetPrejobStep().GetProvisionDutExperiment()
	if e == nil {
		return false
	}
	if !e.Enabled {
		return false
	}

	v := chromeOSBuildDependencyOrEmpty(r.SoftwareDependencies)
	switch fs := e.GetCrosVersionSelector().(type) {
	case *phosphorus.ProvisionDutExperiment_CrosVersionAllowList:
		for _, p := range fs.CrosVersionAllowList.Prefixes {
			if strings.HasPrefix(v, p) {
				return true
			}
		}
		return false
	case *phosphorus.ProvisionDutExperiment_CrosVersionDisallowList:
		for _, p := range fs.CrosVersionDisallowList.Prefixes {
			if strings.HasPrefix(v, p) {
				return false
			}
		}
		return true
	default:
		// Arbitrary default. We don't actually want the config to ever leave
		// this oneof unset.
		return false
	}
}

func provisionChromeOSBuildLegacy(ctx context.Context, r *phosphorus.PrejobRequest) (*phosphorus.PrejobResponse, error) {
	desired := chromeOSBuildDependencyOrEmpty(r.SoftwareDependencies)
	exists := r.ExistingProvisionableLabels[chromeOSBuildKey]
	if desired != "" && desired != exists {
		ar, err := provisionViaAutoserv(ctx, "os", r, []string{fmt.Sprintf("%s:%s", chromeOSBuildKey, desired)})
		return toPrejobResponse(ar), err
	}
	ar, err := resetViaAutoserv(ctx, r)
	return toPrejobResponse(ar), err
}

func provisionFirmwareLegacy(ctx context.Context, r *phosphorus.PrejobRequest) (*phosphorus.PrejobResponse, error) {
	labels := []string{}
	desired := roFirmwareBuildDependencyOrEmpty(r.SoftwareDependencies)
	exists := r.ExistingProvisionableLabels[roFirmwareBuildKey]
	if desired != "" && desired != exists {
		labels = append(labels, fmt.Sprintf("%s:%s", roFirmwareBuildKey, desired))
	}
	desired = rwFirmwareBuildDependencyOrEmpty(r.SoftwareDependencies)
	exists = r.ExistingProvisionableLabels[rwFirmwareBuildKey]
	if desired != "" && desired != exists {
		labels = append(labels, fmt.Sprintf("%s:%s", rwFirmwareBuildKey, desired))
	}

	if len(labels) > 0 {
		ar, err := provisionViaAutoserv(ctx, "firmware", r, labels)
		return toPrejobResponse(ar), err
	}
	// Nothing to be done, so succeed trivially.
	return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_SUCCEEDED}, nil
}

func toPrejobResponse(r *atutil.Result) *phosphorus.PrejobResponse {
	switch {
	case r == nil:
		return nil
	case r.Success():
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_SUCCEEDED}
	case r.RunResult.Aborted:
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_ABORTED}
	default:
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_FAILED}
	}
}

const (
	chromeOSBuildKey   = "cros-version"
	roFirmwareBuildKey = "fwro-version"
	rwFirmwareBuildKey = "fwrw-version"

	gsImageArchivePrefix = "gs://chromeos-image-archive"
)

func validatePrejobRequest(r *phosphorus.PrejobRequest) error {
	missingArgs := getCommonMissingArgs(r.Config)

	if r.DutHostname == "" {
		missingArgs = append(missingArgs, "DUT hostname")
	}

	if len(missingArgs) > 0 {
		return fmt.Errorf("no %s provided", strings.Join(missingArgs, ", "))
	}

	return nil
}

// provisionViaAutoserv provisions a single host. It is a wrapper around
// `autoserv --provision`. It cannot modify its point arguments.
//
// tag is a human-readable name for the provision operation being attempted.
// labels are the list of Tauto labels to be provisioned.
//
// This function will be obsoleted once runTLSProvision is globally enabled.
func provisionViaAutoserv(ctx context.Context, tag string, r *phosphorus.PrejobRequest, labels []string) (*atutil.Result, error) {
	j := getMainJob(r.Config)
	subDir := fmt.Sprintf("provision_%s_%s", r.DutHostname, tag)
	fullPath := filepath.Join(r.Config.Task.ResultsDir, subDir)
	p := &atutil.Provision{
		Host:       r.DutHostname,
		Labels:     labels,
		ResultsDir: fullPath,
	}
	ar, err := atutil.RunAutoserv(ctx, j, p, os.Stdout)
	if err != nil {
		return nil, errors.Annotate(err, "run provision").Err()
	}
	return ar, nil
}

// resetViaAutoserv resets a single host. It is a wrapper around
// `autoserv --reset`.
//
// This function will be obsoleted once runTLSProvision is globally enabled.
func resetViaAutoserv(ctx context.Context, r *phosphorus.PrejobRequest) (*atutil.Result, error) {
	j := getMainJob(r.Config)
	subDir := fmt.Sprintf("reset_%s", r.DutHostname)
	fullPath := filepath.Join(r.Config.Task.ResultsDir, subDir)
	a := &atutil.AdminTask{
		Host:       r.DutHostname,
		ResultsDir: fullPath,
		Type:       atutil.Reset,
	}
	ar, err := atutil.RunAutoserv(ctx, j, a, os.Stdout)
	if err != nil {
		return nil, errors.Annotate(err, "run reset").Err()
	}
	return ar, nil
}

// provisionChromeOSBuildViaTLS provisions a DUT via the TLS API.
// See go/cros-tls go/cros-prover
//
// Errors returned from the actual provision operation are interpreted into
// the response. An error is returned for failure modes that can not be mapped
// to a response.
func provisionChromeOSBuildViaTLS(ctx context.Context, bt *backgroundTLS, r *phosphorus.PrejobRequest) (*phosphorus.PrejobResponse, error) {
	desired := chromeOSBuildDependencyOrEmpty(r.SoftwareDependencies)
	if desired == "" {
		logging.Infof(ctx, "Skipped Chrome OS Build provision, because no specific version was requested.")
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_SUCCEEDED}, nil
	}
	if exists := r.ExistingProvisionableLabels[chromeOSBuildKey]; exists == desired {
		logging.Infof(ctx, "Skipped Chrome OS Build provision, because requested version (%s) is already installed", desired)
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_SUCCEEDED}, nil
	}

	logging.Infof(ctx, "Starting to provision Chrome OS via TLS.")
	c := tlsapi.NewCommonClient(bt.Client)
	op, err := c.ProvisionDut(
		ctx,
		&tlsapi.ProvisionDutRequest{
			Name: r.DutHostname,
			Image: &tlsapi.ProvisionDutRequest_ChromeOSImage{
				PathOneof: &tlsapi.ProvisionDutRequest_ChromeOSImage_GsPathPrefix{
					GsPathPrefix: fmt.Sprintf("%s/%s", gsImageArchivePrefix, desired),
				},
			},
		},
	)
	if err != nil {
		// Errors here indicate a failure in starting the operation, not failure
		// in the provision attempt.
		return nil, errors.Annotate(err, "provision chromeos via tls").Err()
	}

	resp, err := waitForOp(ctx, bt, op)
	if err != nil {
		return nil, errors.Annotate(err, "provision chromeos via tls").Err()
	}
	return resp, nil
}

func waitForOp(ctx context.Context, bt *backgroundTLS, op *longrunning.Operation) (*phosphorus.PrejobResponse, error) {
	op, err := lro.Wait(ctx, longrunning.NewOperationsClient(bt.Client), op.GetName())
	if err != nil {
		// TODO(pprabhu) Cancel operation.
		// - Create 60 second headroom before deadline for cancellation.
		// - Cancel operation and wait up to deadline for cancellation to complete.
		// - Return multi-error with failure to cancel, if cancellation fails.
		s, isGRPCErr := status.FromError(err)
		if err == context.DeadlineExceeded || (isGRPCErr && s.Code() == codes.InvalidArgument) {
			return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_ABORTED}, nil
		}
		return nil, err
	}
	if s := op.GetError(); s != nil {
		// Error here is a failure in the provision attempt.
		// TODO(pprabhu) Surface detailed errors up.
		// See https://docs.google.com/document/d/12w5pPntorUY1cgDHHxT3Nu6wdhVox288g5_BnyKCPOE/edit#heading=h.fj6zbs6kop08
		logging.Errorf(ctx, "Operation failed: (code: %s, message: %s, details: %s", s.Code, s.Message, s.Details)
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_FAILED}, nil
	}
	return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_SUCCEEDED}, nil
}

func provisionLacros(ctx context.Context, bt *backgroundTLS, r *phosphorus.PrejobRequest) (*phosphorus.PrejobResponse, error) {
	src := lacrosGCSPathOrEmpty(r.SoftwareDependencies)
	if src == "" {
		logging.Infof(ctx, "Skipped LaCrOS provision, because no specific version was requested.")
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_SUCCEEDED}, nil
	}

	logging.Infof(ctx, "Starting to provision LaCrOS.")
	c := tlsapi.NewCommonClient(bt.Client)
	op, err := c.ProvisionLacros(
		ctx,
		&tlsapi.ProvisionLacrosRequest{
			Name: r.DutHostname,
			Image: &tlsapi.ProvisionLacrosRequest_LacrosImage{
				PathOneof: &tlsapi.ProvisionLacrosRequest_LacrosImage_GsPathPrefix{
					GsPathPrefix: src,
				},
			},
		},
	)
	if err != nil {
		// Errors here indicate a failure in starting the operation, not failure
		// in the provision attempt.
		return nil, errors.Annotate(err, "provision lacros").Err()
	}

	resp, err := waitForOp(ctx, bt, op)
	if err != nil {
		return nil, errors.Annotate(err, "provision lacros").Err()
	}
	return resp, nil
}

func chromeOSBuildDependencyOrEmpty(deps []*test_platform.Request_Params_SoftwareDependency) string {
	for _, d := range deps {
		if b, ok := d.Dep.(*test_platform.Request_Params_SoftwareDependency_ChromeosBuild); ok {
			return b.ChromeosBuild
		}
	}
	return ""
}

func roFirmwareBuildDependencyOrEmpty(deps []*test_platform.Request_Params_SoftwareDependency) string {
	for _, d := range deps {
		if b, ok := d.Dep.(*test_platform.Request_Params_SoftwareDependency_RoFirmwareBuild); ok {
			return b.RoFirmwareBuild
		}
	}
	return ""
}

func rwFirmwareBuildDependencyOrEmpty(deps []*test_platform.Request_Params_SoftwareDependency) string {
	for _, d := range deps {
		if b, ok := d.Dep.(*test_platform.Request_Params_SoftwareDependency_RwFirmwareBuild); ok {
			return b.RwFirmwareBuild
		}
	}
	return ""
}

func lacrosGCSPathOrEmpty(deps []*test_platform.Request_Params_SoftwareDependency) string {
	for _, d := range deps {
		if b, ok := d.Dep.(*test_platform.Request_Params_SoftwareDependency_LacrosGcsPath); ok {
			return b.LacrosGcsPath
		}
	}
	return ""
}

type backgroundTLS struct {
	server *tls.Server
	Client *grpc.ClientConn
}

func (b *backgroundTLS) Close() error {
	// Make it safe to Close() more than once.
	if b.server == nil {
		return nil
	}
	err := b.Client.Close()
	b.server.Stop()
	b.server = nil
	return err
}

// Run a TLS server in the background and crate a gRPC client to it.
//
// On success, the caller must call backgroundTLS.Close() to clean up resources.
func newBackgroundTLS() (*backgroundTLS, error) {
	s, err := tls.StartBackground(fmt.Sprintf("0.0.0.0:%d", droneTLWPort))
	if err != nil {
		return nil, errors.Annotate(err, "start background TLS").Err()
	}
	c, err := grpc.Dial(s.Address(), grpc.WithInsecure())
	if err != nil {
		s.Stop()
		return nil, errors.Annotate(err, "start background TLS").Err()
	}
	return &backgroundTLS{
		server: s,
		Client: c,
	}, nil
}

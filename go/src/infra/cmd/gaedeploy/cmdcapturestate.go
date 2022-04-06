// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	appengine "cloud.google.com/go/appengine/apiv1"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	appenginepb "google.golang.org/genproto/googleapis/appengine/v1"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/deploy/api/modelpb"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

var cmdCaptureState = &subcommands.Command{
	UsageLine: "capture-state [...]",
	ShortDesc: "captures the current state of the GAE app",
	LongDesc: `Captures the current state of the GAE app by calling GAE Admin API.

Writes it as JSONPB serialized deploy.model.AssetState message to stdout or
a path specified by -json-output.

On failures, writes the error details into the "status" field of the JSON
output, logs the error to stderr, and exits with non-zero exit code.
`,

	CommandRun: func() subcommands.CommandRun {
		c := &cmdCaptureStateRun{}
		c.init()
		return c
	},
}

type cmdCaptureStateRun struct {
	commandBase
}

func (c *cmdCaptureStateRun) init() {
	c.commandBase.init(c.exec, extraFlags{
		appID:      true,
		jsonOutput: true,
	})
}

func (c *cmdCaptureStateRun) exec(ctx context.Context) error {
	gaeState, err := captureState(ctx, c.appID)
	assetState := &modelpb.AssetState{
		Timestamp: timestamppb.Now(),
		Status:    errorToStatus(err),
	}
	// Report only fully captured GAE state, LUCI Deploy backend doesn't like
	// incomplete states.
	if err == nil {
		assetState.State = &modelpb.AssetState_Appengine{
			Appengine: gaeState,
		}
	}
	writeErr := writeOutput(c.jsonOutput, assetState)
	if err != nil {
		return err
	}
	return writeErr
}

func errorToStatus(err error) *statuspb.Status {
	if err == nil {
		return nil
	}

	// Fish out the original gRPC status (if any) to get the code and details.
	var st *status.Status
	errors.WalkLeaves(err, func(err error) bool {
		var ok bool
		st, ok = status.FromError(err)
		return !ok
	})

	// Use the original code and details, but use the annotated error message.
	if st != nil {
		stpb := st.Proto()
		stpb.Message = err.Error()
		return stpb
	}

	// Report non-gRPC errors as Internal.
	return &statuspb.Status{
		Code:    int32(codes.Internal),
		Message: err.Error(),
	}
}

func captureState(ctx context.Context, appID string) (*modelpb.AppengineState, error) {
	state := &modelpb.AppengineState{}

	creds, err := auth.NewAuthenticator(ctx, auth.SilentLogin, chromeinfra.SetDefaultAuthOptions(auth.Options{
		Scopes: []string{
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/userinfo.email",
		},
	})).TokenSource()
	if err != nil {
		return state, err
	}

	retryOnErrors := func(method string) gax.CallOption {
		return gax.WithRetry(func() gax.Retryer {
			return gax.OnErrorFunc(gax.Backoff{
				Initial:    100 * time.Millisecond,
				Max:        30000 * time.Millisecond,
				Multiplier: 1.30,
			}, func(err error) bool {
				switch status.Code(err) {
				case codes.DeadlineExceeded, codes.Internal, codes.Unavailable, codes.Unknown:
					logging.Errorf(ctx, "%s: %s, retrying", method, err)
					return true
				default:
					return false
				}
			})
		})
	}

	appsClient, err := appengine.NewApplicationsClient(ctx, option.WithTokenSource(creds))
	if err != nil {
		return state, err
	}
	defer appsClient.Close()

	servicesClient, err := appengine.NewServicesClient(ctx, option.WithTokenSource(creds))
	if err != nil {
		return state, err
	}
	defer servicesClient.Close()

	versionsClient, err := appengine.NewVersionsClient(ctx, option.WithTokenSource(creds))
	if err != nil {
		return state, err
	}
	defer versionsClient.Close()

	// Ask the Cloud library to do retries for us. It doesn't do so by default.
	appsClient.CallOptions.GetApplication = append(appsClient.CallOptions.GetApplication, retryOnErrors("GetApplication"))
	servicesClient.CallOptions.ListServices = append(servicesClient.CallOptions.ListServices, retryOnErrors("ListServices"))
	versionsClient.CallOptions.ListVersions = append(versionsClient.CallOptions.ListVersions, retryOnErrors("ListVersions"))

	// Note: we deliberately do not parallelize any calls below to avoid hitting
	// (very low) limits on QPM to Appengine Admin API. Some parallelization
	// happens on recipes level (each app is handled independently).

	logging.Infof(ctx, "Getting application info")
	app, err := appsClient.GetApplication(ctx, &appenginepb.GetApplicationRequest{
		Name: fmt.Sprintf("apps/%s", appID),
	})
	if err != nil {
		return state, errors.Annotate(err, "in GetApplication").Err()
	}
	state.CapturedState = &modelpb.AppengineState_CapturedState{
		LocationId:      app.LocationId,
		DefaultHostname: app.DefaultHostname,
		DatabaseType:    modelpb.AppengineState_CapturedState_DatabaseType(app.DatabaseType),
	}

	logging.Infof(ctx, "Listing services")
	services, err := listServices(ctx, servicesClient, &appenginepb.ListServicesRequest{
		Parent: app.Name,
	})
	if err != nil {
		return state, errors.Annotate(err, "in ListServices").Err()
	}

	for _, service := range services {
		logging.Infof(ctx, "Listing versions of %q", service.Id)
		vers, err := listVersions(ctx, versionsClient, &appenginepb.ListVersionsRequest{
			Parent: service.Name,
		})
		if err != nil {
			return state, errors.Annotate(err, "in ListVersions(%q)", service.Id).Err()
		}

		versions := make([]*modelpb.AppengineState_Service_Version, len(vers))
		for idx, ver := range vers {
			versions[idx] = &modelpb.AppengineState_Service_Version{
				Name: ver.Id,
				CapturedState: &modelpb.AppengineState_Service_Version_CapturedState{
					InstanceClass:     ver.InstanceClass,
					Env:               ver.Env,
					Runtime:           ver.Runtime,
					RuntimeChannel:    ver.RuntimeChannel,
					RuntimeApiVersion: ver.RuntimeApiVersion,
					CreatedBy:         ver.CreatedBy,
					CreateTime:        ver.CreateTime,
					VersionUrl:        ver.VersionUrl,
				},
			}
		}
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].Name < versions[j].Name
		})

		state.Services = append(state.Services, &modelpb.AppengineState_Service{
			Name:              service.Id,
			TrafficSplitting:  modelpb.AppengineState_Service_TrafficSplitting(service.Split.GetShardBy()),
			TrafficAllocation: rescaleTrafficAlloc(service.Split.GetAllocations()),
			Versions:          versions,
		})
	}

	sort.Slice(state.Services, func(i, j int) bool {
		return state.Services[i].Name < state.Services[j].Name
	})

	return state, nil
}

func rescaleTrafficAlloc(m map[string]float64) map[string]int32 {
	out := make(map[string]int32, len(m))
	for k, v := range m {
		out[k] = int32(v * 1000)
	}
	return out
}

func listServices(ctx context.Context, client *appengine.ServicesClient, req *appenginepb.ListServicesRequest) (listing []*appenginepb.Service, err error) {
	it := client.ListServices(ctx, req)
	for {
		switch item, err := it.Next(); {
		case err == iterator.Done:
			return listing, nil
		case err != nil:
			return nil, err
		default:
			listing = append(listing, item)
		}
	}
}

func listVersions(ctx context.Context, client *appengine.VersionsClient, req *appenginepb.ListVersionsRequest) (listing []*appenginepb.Version, err error) {
	it := client.ListVersions(ctx, req)
	for {
		switch item, err := it.Next(); {
		case err == iterator.Done:
			return listing, nil
		case err != nil:
			return nil, err
		default:
			listing = append(listing, item)
		}
	}
}

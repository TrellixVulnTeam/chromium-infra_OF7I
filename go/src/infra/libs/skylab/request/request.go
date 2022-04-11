// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package request provides a library to create swarming requests based on
// skylab test or task parameters.
package request

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	swarming "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/data/strpair"
	"go.chromium.org/luci/common/errors"

	"infra/libs/skylab/inventory"
	swarming_inventory "infra/libs/skylab/inventory/swarming"
	"infra/libs/skylab/worker"
)

// Args defines the set of arguments for creating a request.
type Args struct {
	// Cmd specifies the payload command to run for the request.
	Cmd worker.Command
	// TODO(crbug.com/1033291): Rename to Skylab tags.
	SwarmingTags []string
	// ProvisionableDimensions specifies the provisionable dimensions in raw
	// string form; e.g. {"provisionable-cros-version:foo-cq-R75-1.2.3.4"}
	ProvisionableDimensions []string
	// ProvisionableDimensionExpiration specifies the interval of time
	// during which Swarming will attempt to find a bot matching optional
	// (i.e. provisionable) dimensions. After the expiration time Swarming
	// will only use required dimensions for finding the bot.
	ProvisionableDimensionExpiration time.Duration
	// Dimensions specifies swarming dimensions in raw string form.
	//
	// It is preferable to specify dimensions via the SchedulableLabels
	// argument. This argument should only be used for user-supplied freeform
	// dimensions; e.g. {"label-power:battery"}
	Dimensions []string
	// SchedulableLabels specifies schedulable label requirements that will
	// be translated to dimensions.
	SchedulableLabels *inventory.SchedulableLabels
	// Contain a slice of SchedulableLabels instance, which each of them
	// represents scheduleable label requirements of a secondary devices.
	SecondaryDevicesLabels []*inventory.SchedulableLabels
	Timeout                time.Duration
	Priority               int64
	ParentTaskID           string
	ParentRequestUID       string
	// Pubsub Topic for status updates on the tests run for the request
	StatusTopic string
	// If CFT is enabled.
	CFTIsEnabled bool
	// Test describes the test to be run.
	TestRunnerRequest *skylab_test_runner.Request
	// Describes the test to be run via CFT workflow.
	CFTTestRunnerRequest *skylab_test_runner.CFTTestRequest
	// Experiments to pass on to test_runner builders.
	Experiments []string
	// The Gerrit Changes associated with the test_runner invocation.
	GerritChanges []*buildbucket_pb.GerritChange
}

// MessagePayload contains the information for Pubsub subscribers.
type MessagePayload struct {
	ParentRequestUID string `json:"parent_request_uid,omitempty"`
}

// NewBBRequest returns the Buildbucket request to create the test_runner build
// with these arguments.
func (a *Args) NewBBRequest(b *buildbucket_pb.BuilderID) (*buildbucket_pb.ScheduleBuildRequest, error) {
	bbDims, err := a.getBBDimensions()
	if err != nil {
		return nil, errors.Annotate(err, "create bb request").Err()
	}

	props := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"cft_is_enabled": {
				Kind: &structpb.Value_BoolValue{BoolValue: a.CFTIsEnabled},
			},
		},
	}

	if a.CFTIsEnabled {
		cftReq, err := cftRequestToStructPB(a.CFTTestRunnerRequest)
		if err != nil {
			return nil, errors.Annotate(err, "create bb request: cftRequestToStructPB").Err()
		}
		props.Fields["cft_test_request"] = cftReq
	} else {
		// TODO(crbug.com/1036559#c1): Add timeouts.
		req, err := requestToStructPB(a.TestRunnerRequest)
		if err != nil {
			return nil, errors.Annotate(err, "create bb request: requestToStructPB").Err()
		}
		props.Fields["request"] = req
	}

	exps := make(map[string]bool)
	for _, k := range a.Experiments {
		exps[k] = true
	}

	tags, err := parseBBStringPairs(a.SwarmingTags)
	if err != nil {
		return nil, errors.Annotate(err, "create bb request").Err()
	}

	// Add the hide-in-gerrit tag to CQ runs so it does not show up on Gerrit.
	if len(a.GerritChanges) > 0 {
		tags = append(tags, &buildbucket_pb.StringPair{
			Key:   "hide-in-gerrit",
			Value: "test_runner",
		})
	}

	br := &buildbucket_pb.ScheduleBuildRequest{
		Builder:       b,
		GerritChanges: a.GerritChanges,
		Properties:    props,
		Experiments:   exps,
		Tags:          tags,
		Dimensions:    bbDims,
		Priority:      int32(a.Priority),
		Swarming: &buildbucket_pb.ScheduleBuildRequest_Swarming{
			ParentRunId: a.ParentTaskID,
		},
	}
	if a.StatusTopic != "" {
		br.Notify = &buildbucket_pb.NotificationConfig{
			PubsubTopic: a.StatusTopic,
		}
	}
	return br, nil
}

// getBBDimensions returns both required and optional dimensions that will be
// used to match this request with a Swarming bot.
func (a *Args) getBBDimensions() ([]*buildbucket_pb.RequestedDimension, error) {
	ret := inventoryDimensionsToBBDimensions(a.getInventoryDimensions())

	pd, err := dims(a.ProvisionableDimensions).BBDimensions()
	if err != nil {
		return nil, errors.Annotate(err, "get BB dimensions").Err()
	}

	if a.ProvisionableDimensionExpiration != 0 {
		setDimensionExpiration(pd, a.ProvisionableDimensionExpiration)
	}

	ret = append(ret, pd...)

	extraDims, err := dims(a.Dimensions).BBDimensions()
	if err != nil {
		return nil, errors.Annotate(err, "get BB dimensions").Err()
	}
	ret = append(ret, extraDims...)
	return ret, nil
}

// getInventoryDimensions converts SchedulableLabels to inventory dimensions.
// If secondary devices exist, it will also convert SchedulableLabels of secondary
// devices to inventory dimensions, and then join them into one.
func (a *Args) getInventoryDimensions() swarming_inventory.Dimensions {
	dims := swarming_inventory.Convert(a.SchedulableLabels)
	if len(a.SecondaryDevicesLabels) == 0 {
		return dims
	}
	var secondaryDims []swarming_inventory.Dimensions
	for _, sl := range a.SecondaryDevicesLabels {
		secondaryDims = append(secondaryDims, swarming_inventory.Convert(sl))
	}
	for _, sd := range secondaryDims {
		for key, values := range sd {
			if _, ok := dims[key]; ok {
				dims[key] = append(dims[key], values...)
			} else {
				dims[key] = values
			}
		}
	}
	return trimJoinedDimensions(dims)
}

// trimJoinedDimensions remove duplicate values from dimensions, for special dimensions
// ("label-board" and "label-model") it adding an numeric suffix to make it unique.
func trimJoinedDimensions(dims swarming_inventory.Dimensions) swarming_inventory.Dimensions {
	trimmedDims := make(swarming_inventory.Dimensions)
	for key, values := range dims {
		counter := make(map[string]int)
		for _, value := range values {
			if counter[value] == 0 {
				trimmedDims[key] = append(trimmedDims[key], value)
				counter[value]++
			} else if key == "label-board" || key == "label-model" {
				// If a test want more than one DUTs in a same boards or model, we need to add
				// a suffix to make all of them unique, otherwise it will be treated as a single
				// dimension match from swarming.
				suffix := counter[value] + 1
				trimmedDims[key] = append(trimmedDims[key], fmt.Sprintf("%s_%d", value, suffix))
				counter[value]++
			}
		}
	}
	return trimmedDims
}

func inventoryDimensionsToBBDimensions(id swarming_inventory.Dimensions) []*buildbucket_pb.RequestedDimension {
	var ret []*buildbucket_pb.RequestedDimension
	for key, values := range id {
		for _, value := range values {
			ret = append(ret, &buildbucket_pb.RequestedDimension{
				Key:   key,
				Value: value,
			})
		}
	}
	return ret
}

// TODO(zamorzaev): make the type public and refactor the clients to use it.
type dims []string

// BBDimensions converts a dims of the form "foo:bar" to BB rpc requested
// dimensions.
func (d dims) BBDimensions() ([]*buildbucket_pb.RequestedDimension, error) {
	ret := make([]*buildbucket_pb.RequestedDimension, len(d))
	for i, dim := range d {
		k, v := strpair.Parse(dim)
		if v == "" {
			return nil, fmt.Errorf("malformed dimension with key '%s' has no value", k)
		}
		ret[i] = &buildbucket_pb.RequestedDimension{
			Key:   k,
			Value: v,
		}
	}
	return ret, nil
}

// setDimensionExpiration adds an expiration to each requested dimension.
func setDimensionExpiration(d []*buildbucket_pb.RequestedDimension, expiration time.Duration) {
	for _, dim := range d {
		dim.Expiration = ptypes.DurationProto(expiration)
	}
}

type provisionDims dims

// StrippedDims removes "provisionable-" prefix.
func (p provisionDims) StrippedDims() []string {
	ret := make([]string, len(p))
	for i, l := range p {
		ret[i] = strings.TrimPrefix(l, "provisionable-")
	}
	return ret
}

// parseBBStringPairs converts strings of the form "foo:bar" to BB rpc string
// pairs.
func parseBBStringPairs(tags []string) ([]*buildbucket_pb.StringPair, error) {
	ret := make([]*buildbucket_pb.StringPair, len(tags))
	for i, t := range tags {
		k, v := strpair.Parse(t)
		if v == "" {
			return nil, fmt.Errorf("malformed tag with key '%s' has no value", k)
		}
		ret[i] = &buildbucket_pb.StringPair{
			Key:   k,
			Value: v,
		}
	}
	return ret, nil
}

// requestToStructPB converts a skylab_test_runner.Request into a Struct
// with the same JSON presentation.
func requestToStructPB(from *skylab_test_runner.Request) (*structpb.Value, error) {
	m := jsonpb.Marshaler{}
	jsonStr, err := m.MarshalToString(from)
	if err != nil {
		return nil, err
	}
	reqStruct := &structpb.Struct{}
	if err := jsonpb.UnmarshalString(jsonStr, reqStruct); err != nil {
		return nil, err
	}
	return &structpb.Value{
		Kind: &structpb.Value_StructValue{StructValue: reqStruct},
	}, nil
}

// cftRequestToStructPB converts a skylab_test_runner.CFTTestRequest into a Struct
// with the same JSON presentation.
func cftRequestToStructPB(from *skylab_test_runner.CFTTestRequest) (*structpb.Value, error) {
	m := jsonpb.Marshaler{}
	jsonStr, err := m.MarshalToString(from)
	if err != nil {
		return nil, err
	}
	reqStruct := &structpb.Struct{}
	if err := jsonpb.UnmarshalString(jsonStr, reqStruct); err != nil {
		return nil, err
	}
	return &structpb.Value{
		Kind: &structpb.Value_StructValue{StructValue: reqStruct},
	}, nil
}

// newNotificationConfig constructs a valid NotificationConfig.
func newNotificationConfig(topic string, payload []byte) *buildbucket_pb.NotificationConfig {
	if topic == "" {
		// BB will crash if it encounters a non-nil NotificationConfig with an
		// empty PubsubTopic.
		return nil
	}
	return &buildbucket_pb.NotificationConfig{
		PubsubTopic: topic,
		UserData:    payload,
	}
}

// SwarmingNewTaskRequest returns the Swarming request to create the Skylab
// task with these arguments.
func (a *Args) SwarmingNewTaskRequest() (*swarming.SwarmingRpcsNewTaskRequest, error) {
	dims, err := a.StaticDimensions()
	if err != nil {
		return nil, errors.Annotate(err, "create request").Err()
	}
	slices, err := getSlices(a.Cmd, dims, a.ProvisionableDimensions, a.Timeout)
	if err != nil {
		return nil, errors.Annotate(err, "create request").Err()
	}

	req := &swarming.SwarmingRpcsNewTaskRequest{
		Name:         a.Cmd.TaskName,
		Tags:         a.SwarmingTags,
		TaskSlices:   slices,
		Priority:     a.Priority,
		ParentTaskId: a.ParentTaskID,
		PubsubTopic:  a.StatusTopic,
	}
	return req, nil
}

// StaticDimensions returns the dimensions required on a Swarming bot that can
// service this request.
//
// StaticDimensions() do not include dimensions used to optimize task
// scheduling.
func (a *Args) StaticDimensions() ([]*swarming.SwarmingRpcsStringPair, error) {
	ret := inventoryDimensionsToPairs(a.getInventoryDimensions())
	d, err := stringToPairs(a.Dimensions...)
	if err != nil {
		return nil, errors.Annotate(err, "get static dimensions").Err()
	}
	ret = append(ret, d...)
	ret = append(ret, &swarming.SwarmingRpcsStringPair{
		Key:   "pool",
		Value: "ChromeOSSkylab",
	})
	return ret, nil
}

// getSlices generates and returns the set of swarming task slices for the given test task.
func getSlices(cmd worker.Command, staticDimensions []*swarming.SwarmingRpcsStringPair, provisionableDimensions []string, timeout time.Duration) ([]*swarming.SwarmingRpcsTaskSlice, error) {
	slices := make([]*swarming.SwarmingRpcsTaskSlice, 1, 2)

	dims, _ := stringToPairs("dut_state:ready")
	dims = append(dims, staticDimensions...)

	provisionablePairs, err := stringToPairs(provisionableDimensions...)
	if err != nil {
		return nil, errors.Annotate(err, "create slices").Err()
	}

	s0Dims := append(dims, provisionablePairs...)
	slices[0] = taskSlice(cmd.Args(), s0Dims, timeout)

	if len(provisionableDimensions) != 0 {
		cmd.ProvisionLabels = provisionDims(provisionableDimensions).StrippedDims()
		s1Dims := dims
		slices = append(slices, taskSlice(cmd.Args(), s1Dims, timeout))
	}

	finalSlice := slices[len(slices)-1]
	finalSlice.ExpirationSecs = int64(timeout.Seconds())

	return slices, nil
}

func taskSlice(command []string, dimensions []*swarming.SwarmingRpcsStringPair, timeout time.Duration) *swarming.SwarmingRpcsTaskSlice {
	return &swarming.SwarmingRpcsTaskSlice{
		// We want all slices to wait, at least a little while, for bots with
		// metching dimensions.
		// For slice 0: This allows the task to try to re-use provisionable
		// labels that get set by previous tasks with the same label that are
		// about to finish.
		// For slice 1: This allows the task to wait for devices to get
		// repaired, if there are no devices with dut_state:ready.
		WaitForCapacity: true,
		// Slice 0 should have a fairly short expiration time, to reduce
		// overhead for tasks that are the first ones enqueue with a particular
		// provisionable label. This value will be overwritten for the final
		// slice of a task.
		ExpirationSecs: 30,
		Properties: &swarming.SwarmingRpcsTaskProperties{
			Command:              command,
			Dimensions:           dimensions,
			ExecutionTimeoutSecs: int64(timeout.Seconds()),
		},
	}
}

// stringToPairs converts a slice of strings in foo:bar form to a slice of swarming
// rpc string pairs.
func stringToPairs(dimensions ...string) ([]*swarming.SwarmingRpcsStringPair, error) {
	pairs := make([]*swarming.SwarmingRpcsStringPair, len(dimensions))
	for i, d := range dimensions {
		k, v := strpair.Parse(d)
		if v == "" {
			return nil, fmt.Errorf("malformed dimension with key '%s' has no value", k)
		}
		pairs[i] = &swarming.SwarmingRpcsStringPair{Key: k, Value: v}
	}
	return pairs, nil
}

func inventoryDimensionsToPairs(dimensions swarming_inventory.Dimensions) []*swarming.SwarmingRpcsStringPair {
	pairs := make([]*swarming.SwarmingRpcsStringPair, 0, len(dimensions))
	for key, values := range dimensions {
		for _, value := range values {
			pairs = append(pairs, &swarming.SwarmingRpcsStringPair{Key: key, Value: value})
		}
	}
	return pairs
}

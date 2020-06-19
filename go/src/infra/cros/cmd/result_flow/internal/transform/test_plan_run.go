// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package transform

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/analytics"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// Build presents a CTP build containing multiple TestPlanRuns.
type Build interface {
	ToTestPlanRuns(context.Context) []*analytics.TestPlanRun
}

type ctpBuild struct {
	bb         *result_flow.BuildbucketConfig
	id         int64
	status     bbpb.Status
	createTime *timestamp.Timestamp
	startTime  *timestamp.Timestamp
	endTime    *timestamp.Timestamp
	reqs       map[string]*test_platform.Request
	resps      map[string]*steps.ExecuteResponse
}

// LoadRawBuildBucketResp loads a CTP build from Buildbucket response.
func LoadRawBuildBucketResp(ctx context.Context, b *bbpb.Build, bb *result_flow.BuildbucketConfig) (Build, error) {
	if b == nil {
		return nil, fmt.Errorf("empty build")
	}
	c := &ctpBuild{
		bb:         bb,
		id:         b.GetId(),
		status:     b.GetStatus(),
		createTime: b.GetCreateTime(),
		startTime:  b.GetStartTime(),
		endTime:    b.GetEndTime(),
	}
	var err error
	prop := b.GetInput().GetProperties()
	if prop == nil {
		return nil, fmt.Errorf("Build has no input properties found, %d", b.GetId())
	}
	setDefaultStructValues(prop)
	op := prop.GetFields()
	if rValue, ok := op["request"]; ok {
		request, err := structPBToRequest(rValue)
		if err != nil {
			return nil, errors.Annotate(err, "failed to extract CTP request").Err()
		}
		c.reqs = map[string]*test_platform.Request{
			"default": request,
		}
	}
	if raw, ok := op["requests"]; ok {
		if c.reqs, err = structPBToRequests(raw); err != nil {
			return nil, errors.Annotate(err, "failed to extract CTP requests").Err()
		}
	}
	v, ok := getCTPResponseValue(b)
	if !ok {
		logging.Infof(ctx, "Build has no output properties yet, %d", b.GetId())
		return c, nil
	}
	setDefaultStructValues(b.GetOutput().GetProperties())
	c.resps, err = extractCTPBuildResponses(v)
	return c, err
}

// ToTestPlanRuns derives TestPlanRun entities from a CTP build.
func (c *ctpBuild) ToTestPlanRuns(ctx context.Context) []*analytics.TestPlanRun {
	var res []*analytics.TestPlanRun
	for k := range c.reqs {
		res = append(res, c.genTestPlanRun(k))
	}
	return res
}

func (c *ctpBuild) genTestPlanRun(key string) *analytics.TestPlanRun {
	return &analytics.TestPlanRun{
		Uid:           c.createUID(key),
		Suite:         c.getSuiteName(key),
		ExecutionUrl:  inferExecutionURL(c.bb, c.id),
		DutPool:       c.getDutPool(key),
		BuildTarget:   c.getBuildTarget(key),
		ChromeosBuild: c.getChromeosBuild(key),
		Status:        c.inferTestPlanStatus(key),
		Timeline:      c.getTimeline(),
	}
}

func (c *ctpBuild) createUID(key string) string {
	return fmt.Sprintf("TestPlanRuns/%d/%s", c.id, key)
}

func (c *ctpBuild) getSuiteName(key string) string {
	suites := c.reqs[key].GetTestPlan().Suite
	return suites[0].GetName()
}

func (c *ctpBuild) getDutPool(key string) string {
	scheduling := c.reqs[key].GetParams().GetScheduling()
	if pool := scheduling.GetUnmanagedPool(); pool != "" {
		return pool
	}
	return scheduling.GetManagedPool().String()
}

func (c *ctpBuild) getBuildTarget(key string) string {
	return c.reqs[key].GetParams().GetSoftwareAttributes().GetBuildTarget().GetName()
}

func (c *ctpBuild) getChromeosBuild(key string) string {
	for _, dep := range c.reqs[key].GetParams().GetSoftwareDependencies() {
		if b := dep.GetChromeosBuild(); b != "" {
			return b
		}
	}
	return ""
}

func (c *ctpBuild) inferTestPlanStatus(key string) *analytics.Status {
	if c.resps == nil {
		return nil
	}
	resp, ok := c.resps[key]
	if resp == nil || !ok {
		return nil
	}
	return &analytics.Status{
		Value: resp.GetState().GetLifeCycle().String(),
	}
}

func (c *ctpBuild) getTimeline() *analytics.Timeline {
	return &analytics.Timeline{
		CreateTime: c.createTime,
		StartTime:  c.startTime,
		EndTime:    c.endTime,
	}
}

// TODO(linxinan): Original code is at "infra/cmd/skylab/internal/bb/bb.go". Consider
// moving them to shared libs.
func extractCTPBuildResponses(rs *structpb.Value) (map[string]*steps.ExecuteResponse, error) {
	if b64 := rs.GetStringValue(); b64 != "" {
		responses, err := compressedPBToExecuteResponses(b64)
		if err != nil {
			return nil, errors.Annotate(err, "extractBuildData").Err()
		}
		return responses.TaggedResponses, nil
	}
	return nil, fmt.Errorf("Failed to find response field")
}

func getCTPResponseValue(b *bbpb.Build) (*structpb.Value, bool) {
	op := b.GetOutput().GetProperties().GetFields()
	if op == nil {
		return nil, false
	}
	v, ok := op["compressed_responses"]
	return v, ok
}

func structPBToRequest(from *structpb.Value) (*test_platform.Request, error) {
	m := jsonpb.Marshaler{}
	json, err := m.MarshalToString(from)
	if err != nil {
		return nil, errors.Annotate(err, "structPBToExecuteRequest").Err()
	}
	request := &test_platform.Request{}
	if err := jsonpb.UnmarshalString(json, request); err != nil {
		return nil, errors.Annotate(err, "structPBToExecuteRequest").Err()
	}
	return request, nil
}

func structPBToRequests(from *structpb.Value) (map[string]*test_platform.Request, error) {
	requests := make(map[string]*test_platform.Request)
	m, err := structPBStructToMap(from)
	if err != nil {
		return nil, errors.Annotate(err, "struct PB to requests").Err()
	}
	for k, v := range m {
		r, err := structPBToRequest(v)
		if err != nil {
			return nil, errors.Annotate(err, "struct PB to requests").Err()
		}
		requests[k] = r
	}
	return requests, nil
}

func structPBStructToMap(from *structpb.Value) (map[string]*structpb.Value, error) {
	switch m := from.Kind.(type) {
	case *structpb.Value_StructValue:
		if m.StructValue == nil {
			return nil, errors.Reason("struct has no fields").Err()
		}
		return m.StructValue.Fields, nil
	default:
		return nil, errors.Reason("not a Struct type").Err()
	}
}

func compressedPBToExecuteResponses(from string) (*steps.ExecuteResponses, error) {
	if from == "" {
		return nil, nil
	}
	bs, err := base64.StdEncoding.DecodeString(from)
	if err != nil {
		return nil, errors.Annotate(err, "compressedPBToExecuteResponses").Err()
	}
	reader, err := zlib.NewReader(bytes.NewReader(bs))
	if err != nil {
		return nil, errors.Annotate(err, "compressedPBToExecuteResponses").Err()
	}
	bs, err = ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.Annotate(err, "compressedPBToExecuteResponses").Err()
	}
	resp, err := binPBToExecuteResponses(bs)
	if err != nil {
		return nil, errors.Annotate(err, "compressedPBToExecuteResponses").Err()
	}
	return resp, nil
}

func binPBToExecuteResponses(from []byte) (*steps.ExecuteResponses, error) {
	response := &steps.ExecuteResponses{}
	if err := proto.Unmarshal(from, response); err != nil {
		return nil, errors.Annotate(err, "binPBToExecuteResponses").Err()
	}
	return response, nil
}

// setDefaultStructValues defaults nil or empty values inside the given
// structpb.Struct. Needed because structpb.Value cannot be marshaled to JSON
// unless there is a kind set. More details are in crbug/1093683.
func setDefaultStructValues(s *structpb.Struct) {
	for k, v := range s.GetFields() {
		switch {
		case v == nil:
			s.Fields[k] = &structpb.Value{
				Kind: &structpb.Value_NullValue{},
			}
		case v.Kind == nil:
			v.Kind = &structpb.Value_NullValue{}
		case v.GetStructValue() != nil:
			setDefaultStructValues(v.GetStructValue())
		}
	}
}

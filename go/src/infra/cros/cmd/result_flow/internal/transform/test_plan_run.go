// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package transform

import (
	"context"
	"fmt"

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

// CTPBuildResults presents a CTP build containing multiple TestPlanRuns.
type CTPBuildResults interface {
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

// LoadCTPBuildBucketResp loads a CTP build from Buildbucket response.
func LoadCTPBuildBucketResp(ctx context.Context, b *bbpb.Build, bb *result_flow.BuildbucketConfig) (CTPBuildResults, error) {
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
		request, err := structPBToCTPRequest(rValue)
		if err != nil {
			return nil, errors.Annotate(err, "failed to extract CTP request").Err()
		}
		c.reqs = map[string]*test_platform.Request{
			"default": request,
		}
	}
	if raw, ok := op["requests"]; ok {
		if c.reqs, err = structPBToCTPRequests(raw); err != nil {
			return nil, errors.Annotate(err, "failed to extract CTP requests").Err()
		}
	}
	v, ok := getOutputPropertiesValue(b, "compressed_responses")
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
		Uid:           c.createTestPlanRunUID(key),
		Suite:         c.getSuiteName(key),
		ExecutionUrl:  inferExecutionURL(c.bb, c.id),
		DutPool:       c.getDutPool(key),
		BuildTarget:   c.getBuildTarget(key),
		ChromeosBuild: c.getChromeosBuild(key),
		Status:        c.inferTestPlanStatus(key),
		Timeline:      c.getTimeline(),
	}
}

func (c *ctpBuild) createTestPlanRunUID(key string) string {
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

func extractCTPBuildResponses(rs *structpb.Value) (map[string]*steps.ExecuteResponse, error) {
	if b64 := rs.GetStringValue(); b64 != "" {
		pb, err := unmarshalCompressedString(b64, &steps.ExecuteResponses{})
		if err != nil {
			return nil, errors.Annotate(err, "extract CTP Responses").Err()
		}
		responses, ok := pb.(*steps.ExecuteResponses)
		if !ok {
			return nil, fmt.Errorf("failed to extract ExecuteResponses")
		}
		return responses.TaggedResponses, nil
	}
	return nil, fmt.Errorf("Failed to find response field")
}

func structPBToCTPRequests(from *structpb.Value) (map[string]*test_platform.Request, error) {
	requests := make(map[string]*test_platform.Request)
	m, err := structPBStructToMap(from)
	if err != nil {
		return nil, errors.Annotate(err, "struct PB to CTP requests").Err()
	}
	for k, v := range m {
		if requests[k], err = structPBToCTPRequest(v); err != nil {
			return nil, err
		}
	}
	return requests, nil
}

func structPBToCTPRequest(from *structpb.Value) (*test_platform.Request, error) {
	pb, err := unmarshalStructPB(from, &test_platform.Request{})
	if err != nil {
		return nil, errors.Annotate(err, "struct PB to CTP request").Err()
	}
	request, ok := pb.(*test_platform.Request)
	if !ok {
		return nil, fmt.Errorf("failed to extract CTP request")
	}
	return request, nil
}

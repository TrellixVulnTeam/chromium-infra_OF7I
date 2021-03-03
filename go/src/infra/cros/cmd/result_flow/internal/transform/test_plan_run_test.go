// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package transform_test

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"testing"

	"infra/cros/cmd/result_flow/internal/transform"

	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/analytics"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	. "github.com/smartystreets/goconvey/convey"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
)

var (
	fakeBuildID    int64 = 8878535213888021808
	fakeCreateTime       = &timestamp.Timestamp{Seconds: 1600000000}
	fakeStartTime        = &timestamp.Timestamp{Seconds: 1600000000 + 3600}
	fakeEndTime          = &timestamp.Timestamp{Seconds: 1600000000 + 3600*2}
	fakeSource           = &result_flow.Source{
		Bb: &result_flow.BuildbucketConfig{
			Host:    "cr-buildbucket.appspot.com",
			Project: "chromeos",
			Bucket:  "testplatform",
			Builder: "cros_test_platform",
		},
	}
)

type ctp struct {
	tag         string
	suite       string
	pool        string
	buildTarget string
	crosBuild   string
	status      test_platform.TaskState_LifeCycle
}

type testPlanRun struct {
	tag         string
	suite       string
	pool        string
	buildTarget string
	crosBuild   string
	status      string
}

func TestBuildToTestPlanRuns(t *testing.T) {
	cases := []struct {
		description string
		in          []*ctp
		out         []*testPlanRun
	}{
		{
			"Transform an ongoing CTP build to analytics.TestPlanRun",
			[]*ctp{
				{
					tag:         "foo-1",
					suite:       "fake-suite",
					pool:        "fake-pool",
					buildTarget: "foo",
					crosBuild:   "fake-release",
					status:      test_platform.TaskState_LIFE_CYCLE_RUNNING,
				},
			},
			[]*testPlanRun{
				{
					tag:         "foo-1",
					suite:       "fake-suite",
					pool:        "fake-pool",
					buildTarget: "foo",
					crosBuild:   "fake-release",
					status:      "",
				},
			},
		},
		{
			"Transform a completed CTP build to analytics.TestPlanRun",
			[]*ctp{
				{
					tag:         "hoo-1",
					suite:       "fake-suite",
					pool:        "fake-pool",
					buildTarget: "hoo",
					crosBuild:   "fake-release",
					status:      test_platform.TaskState_LIFE_CYCLE_COMPLETED,
				},
			},
			[]*testPlanRun{
				{
					tag:         "hoo-1",
					suite:       "fake-suite",
					pool:        "fake-pool",
					buildTarget: "hoo",
					crosBuild:   "fake-release",
					status:      "COMPLETED",
				},
			},
		},
		{
			"Transform a completed CTP build with multiple requests",
			[]*ctp{
				{
					tag:         "foo",
					suite:       "fake-suite",
					pool:        "fake-pool",
					buildTarget: "foo",
					crosBuild:   "fake-release-1",
					status:      test_platform.TaskState_LIFE_CYCLE_COMPLETED,
				},
				{
					tag:         "hoo",
					suite:       "fake-suite",
					pool:        "fake-pool",
					buildTarget: "hoo",
					crosBuild:   "fake-release-2",
					status:      test_platform.TaskState_LIFE_CYCLE_CANCELLED,
				},
			},
			[]*testPlanRun{
				{
					tag:         "foo",
					suite:       "fake-suite",
					pool:        "fake-pool",
					buildTarget: "foo",
					crosBuild:   "fake-release-1",
					status:      "COMPLETED",
				},
				{
					tag:         "hoo",
					suite:       "fake-suite",
					pool:        "fake-pool",
					buildTarget: "hoo",
					crosBuild:   "fake-release-2",
					status:      "CANCELLED",
				},
			},
		},
	}
	ctx := context.Background()
	for _, c := range cases {
		Convey(c.description, t, func() {
			Convey("then CTP build is correctly converted to TestPlanRun.", func() {
				build, _ := transform.LoadCTPBuildBucketResp(ctx, genFakeBuild(c.in), fakeSource.Bb)
				got := build.ToTestPlanRuns(ctx)
				sort.Slice(got, func(i, j int) bool { return got[i].Uid < got[j].Uid })
				So(got, ShouldNotBeNil)
				for i := 0; i < len(got); i++ {
					checkTestPlanRunEquality(genFakeTestPlanRun(c.out[i]), got[i])
				}

			})
		})
	}
}

func genFakeTestPlanRun(out *testPlanRun) *analytics.TestPlanRun {
	return &analytics.TestPlanRun{
		Uid:           genFakeUID(out.tag),
		BuildId:       fakeBuildID,
		Suite:         out.suite,
		ExecutionUrl:  genFakeExecutionURL(fakeBuildID),
		DutPool:       out.pool,
		BuildTarget:   out.buildTarget,
		ChromeosBuild: out.crosBuild,
		CreateTime:    fakeCreateTime,
		StartTime:     fakeStartTime,
		EndTime:       fakeEndTime,
		Status: &analytics.Status{
			Value: out.status,
		},
	}
}

func genFakeBuild(inputs []*ctp) *bbpb.Build {
	requests := make(map[string]*test_platform.Request, len(inputs))
	responses := make(map[string]*steps.ExecuteResponse, len(inputs))
	for _, in := range inputs {
		ctpReq := genFakeTestPlatformRequest(in.buildTarget, in.pool, in.crosBuild)
		if in.suite != "" {
			setSuite(ctpReq, in.suite)
		}
		requests[in.tag] = ctpReq
		if int(in.status)&int(test_platform.TaskState_LIFE_CYCLE_MASK_FINAL) != 0 {
			responses[in.tag] = &steps.ExecuteResponse{
				State: &test_platform.TaskState{
					LifeCycle: in.status,
				},
			}
		}
	}
	return &bbpb.Build{
		Id:         fakeBuildID,
		CreateTime: fakeCreateTime,
		StartTime:  fakeStartTime,
		EndTime:    fakeEndTime,
		Input:      ctpRequestsToInputField(requests),
		Output: pbToOutputField(
			&steps.ExecuteResponses{
				TaggedResponses: responses,
			},
			"compressed_responses",
		),
	}
}

func setSuite(req *test_platform.Request, suite string) {
	req.TestPlan = &test_platform.Request_TestPlan{
		Suite: []*test_platform.Request_Suite{
			{
				Name: suite,
			},
		},
	}
}

func genFakeUID(b string) string {
	return fmt.Sprintf("TestPlanRuns/%d/%s", fakeBuildID, b)
}

func genFakeExecutionURL(id int64) string {
	return fmt.Sprintf("https://ci.chromium.org/p/chromeos/builders/testplatform/cros_test_platform/b%d", id)
}

func genFakeTestPlatformRequest(board, pool, crosBuild string) *test_platform.Request {
	return &test_platform.Request{
		Params: &test_platform.Request_Params{
			SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
				BuildTarget: &chromiumos.BuildTarget{
					Name: board,
				},
			},
			Scheduling: &test_platform.Request_Params_Scheduling{
				Pool: &test_platform.Request_Params_Scheduling_UnmanagedPool{
					UnmanagedPool: pool,
				},
			},
			SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
				{
					Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{
						ChromeosBuild: crosBuild,
					},
				},
			},
		},
	}
}

func checkTestPlanRunEquality(want, got *analytics.TestPlanRun) {
	So(want.Uid, ShouldEqual, got.Uid)
	So(want.BuildId, ShouldEqual, got.BuildId)
	So(want.Suite, ShouldEqual, got.Suite)
	So(want.ExecutionUrl, ShouldEqual, got.ExecutionUrl)
	So(want.DutPool, ShouldEqual, got.DutPool)
	So(want.BuildTarget, ShouldEqual, got.BuildTarget)
	So(want.ChromeosBuild, ShouldEqual, got.ChromeosBuild)
	So(want.GetStatus().GetValue(), ShouldEqual, got.GetStatus().GetValue())
	So(got.CreateTime, ShouldEqual, want.CreateTime)
	So(got.StartTime, ShouldEqual, want.StartTime)
	So(got.EndTime, ShouldEqual, want.EndTime)
}

func ctpRequestsToInputField(requests map[string]*test_platform.Request) *bbpb.Build_Input {
	rs, _ := ctpRequestsToStructPB(requests)
	return &bbpb.Build_Input{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"requests": rs,
			},
		},
	}
}

func ctpRequestsToStructPB(from map[string]*test_platform.Request) (*structpb.Value, error) {
	fs := make(map[string]*structpb.Value)
	for k, r := range from {
		v, err := marshalPB(r)
		if err != nil {
			return nil, errors.Annotate(err, "requests to struct pb (%s)", k).Err()
		}
		fs[k] = v
	}
	return &structpb.Value{
		Kind: &structpb.Value_StructValue{
			StructValue: &structpb.Struct{
				Fields: fs,
			},
		},
	}, nil
}

func pbToOutputField(from proto.Message, field string) *bbpb.Build_Output {
	return &bbpb.Build_Output{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				field: {
					Kind: &structpb.Value_StringValue{
						StringValue: compressPBToString(from),
					},
				},
			},
		},
	}
}

func compressPBToString(from proto.Message) string {
	wire, _ := proto.Marshal(from)
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(wire)
	w.Close()
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

func marshalPB(from proto.Message) (*structpb.Value, error) {
	m := jsonpb.Marshaler{}
	jsonStr, err := m.MarshalToString(from)
	if err != nil {
		return nil, err
	}
	to := &structpb.Struct{}
	if err := jsonpb.UnmarshalString(jsonStr, to); err != nil {
		return nil, err
	}
	return &structpb.Value{
		Kind: &structpb.Value_StructValue{StructValue: to},
	}, nil
}

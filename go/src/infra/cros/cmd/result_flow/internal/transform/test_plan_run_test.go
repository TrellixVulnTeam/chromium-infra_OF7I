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
	"infra/cros/cmd/result_flow/internal/transform"
	"sort"
	"testing"

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
	fakeCrosBuild = "release/R83-13020.67.0"
)

func TestBuildToTestPlanRuns(t *testing.T) {
	cases := []struct {
		description string
		in          *bbpb.Build
		want        []*analytics.TestPlanRun
	}{
		{
			"Transform an ongoing CTP build to analytics.TestPlanRun",
			genFakeBuild("foo", false),
			genFakeTestPlanRuns("foo", false),
		},
		{
			"Transform a completed CTP build to analytics.TestPlanRun",
			genFakeBuild("hoo", true),
			genFakeTestPlanRuns("hoo", true),
		},
	}
	ctx := context.Background()
	for _, c := range cases {
		Convey(c.description, t, func() {
			Convey("then CTP build is correctly converted to TestPlanRun.", func() {
				build, _ := transform.LoadCTPBuildBucketResp(ctx, c.in, fakeSource.Bb)
				got := build.ToTestPlanRuns(ctx)
				sort.Slice(got, func(i, j int) bool { return got[i].Uid < got[j].Uid })
				So(got, ShouldNotBeNil)
				for i := 0; i < len(got); i++ {
					checkTestPlanRunEquality(c.want[i], got[i])
				}

			})
		})
	}
}

func genFakeTestPlanRuns(label string, finished bool) []*analytics.TestPlanRun {
	runs := []*analytics.TestPlanRun{
		{
			Uid:           genFakeUID(label),
			BuildId:       fakeBuildID,
			Suite:         genFakeSuite(label),
			ExecutionUrl:  genFakeExecutionURL(),
			DutPool:       genFakePool(label),
			BuildTarget:   label,
			ChromeosBuild: genFakeCrOSBuild(label),
			Timeline:      genAnalyticTimeline(),
		},
	}
	if finished {
		runs[0].Status = &analytics.Status{
			Value: "LIFE_CYCLE_COMPLETED",
		}
	}
	return runs
}

func genFakeBuild(label string, finished bool) *bbpb.Build {
	res := &bbpb.Build{
		Id:         fakeBuildID,
		CreateTime: fakeCreateTime,
		StartTime:  fakeStartTime,
		EndTime:    fakeEndTime,
		Input: ctpRequestsToInputField(
			map[string]*test_platform.Request{
				label: genFakeTestPlatformRequest(label),
			},
		),
	}
	if finished {
		res.Output = pbToOutputField(
			genFakeTestPlatformResponses(
				label,
				test_platform.TaskState_LIFE_CYCLE_COMPLETED,
			),
			"compressed_responses",
		)
	}
	return res
}

func genAnalyticTimeline() *analytics.Timeline {
	return &analytics.Timeline{
		CreateTime: fakeCreateTime,
		StartTime:  fakeStartTime,
		EndTime:    fakeEndTime,
	}
}

func genFakeUID(b string) string {
	return fmt.Sprintf("TestPlanRuns/%d/%s", fakeBuildID, b)
}

func genFakeExecutionURL() string {
	return fmt.Sprintf("https://ci.chromium.org/p/chromeos/builders/testplatform/cros_test_platform/b%d", fakeBuildID)
}

func genFakePool(b string) string {
	return b + "-pool"
}

func genFakeCrOSBuild(b string) string {
	return fmt.Sprintf("%s-%s", b, fakeCrosBuild)
}

func genFakeSuite(b string) string {
	return b + "-suite"
}

func genFakeTestPlatformRequest(board string) *test_platform.Request {
	return &test_platform.Request{
		Params: &test_platform.Request_Params{
			SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
				BuildTarget: &chromiumos.BuildTarget{
					Name: board,
				},
			},
			Scheduling: &test_platform.Request_Params_Scheduling{
				Pool: &test_platform.Request_Params_Scheduling_UnmanagedPool{
					UnmanagedPool: genFakePool(board),
				},
			},
			SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
				{
					Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{
						ChromeosBuild: genFakeCrOSBuild(board),
					},
				},
			},
		},
		TestPlan: &test_platform.Request_TestPlan{

			Suite: []*test_platform.Request_Suite{
				{
					Name: genFakeSuite(board),
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
	So(want.Status, ShouldResemble, got.Status)
	So(want.Timeline, ShouldResemble, got.Timeline)
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

func genFakeTestPlatformResponses(key string, lifeCycle test_platform.TaskState_LifeCycle) *steps.ExecuteResponses {
	return &steps.ExecuteResponses{
		TaggedResponses: map[string]*steps.ExecuteResponse{
			key: {
				State: &test_platform.TaskState{
					LifeCycle: lifeCycle,
				},
			},
		},
	}
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

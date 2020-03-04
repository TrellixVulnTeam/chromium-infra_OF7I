// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package request_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	swarming "go.chromium.org/luci/common/api/swarming/swarming/v1"

	"infra/libs/skylab/inventory"
	"infra/libs/skylab/request"
)

func TestBuilderID(t *testing.T) {
	Convey("Given request arguments that specify a builder ID", t, func() {
		id := buildbucket_pb.BuilderID{
			Project: "foo-project",
			Bucket:  "foo-bucket",
			Builder: "foo-builder",
		}
		args := request.Args{
			BuilderID:         &id,
			TestRunnerRequest: &skylab_test_runner.Request{},
		}
		Convey("when a request is formed", func() {
			req, err := args.NewBBRequest()
			So(err, ShouldBeNil)
			So(req, ShouldNotBeNil)
			Convey("then request should have a builder ID.", func() {
				So(req.Builder, ShouldNotBeNil)
				diff := pretty.Compare(req.Builder, id)
				So(diff, ShouldBeEmpty)
			})
		})
	})
}

func TestDimensionsBB(t *testing.T) {
	Convey("Given request arguments that specify provisionable and regular dimenisons and inventory labels", t, func() {
		model := "foo-model"
		args := request.Args{
			Dimensions:                       []string{"k1:v1"},
			ProvisionableDimensions:          []string{"provisionable-k2:v2", "provisionable-k3:v3"},
			ProvisionableDimensionExpiration: 30 * time.Second,
			SchedulableLabels:                inventory.SchedulableLabels{Model: &model},
			TestRunnerRequest:                &skylab_test_runner.Request{},
		}
		Convey("when a request is formed", func() {
			req, err := args.NewBBRequest()
			So(err, ShouldBeNil)
			So(req, ShouldNotBeNil)
			Convey("then request should have correct dimensions.", func() {
				So(req.Dimensions, ShouldHaveLength, 4)

				want := []*buildbucket_pb.RequestedDimension{
					{
						Key:        "provisionable-k2",
						Value:      "v2",
						Expiration: ptypes.DurationProto(30 * time.Second),
					},
					{
						Key:        "provisionable-k3",
						Value:      "v3",
						Expiration: ptypes.DurationProto(30 * time.Second),
					},
					{
						Key:   "k1",
						Value: "v1",
					},
					{
						Key:   "label-model",
						Value: "foo-model",
					},
				}

				diff := pretty.Compare(sortBBDimensions(req.Dimensions), sortBBDimensions(want))
				So(diff, ShouldBeEmpty)
			})
		})
	})
}

func TestPropertiesBB(t *testing.T) {
	Convey("Given request arguments that specify a test runner request", t, func() {
		want := skylab_test_runner.Request{
			Prejob: &skylab_test_runner.Request_Prejob{
				SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
					{
						Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{
							ChromeosBuild: "foo-build",
						},
					},
				},
				ProvisionableLabels: map[string]string{
					"key": "value",
				},
			},
			Test: &skylab_test_runner.Request_Test{
				Harness: &skylab_test_runner.Request_Test_Autotest_{
					Autotest: &skylab_test_runner.Request_Test_Autotest{
						Name:     "foo-test",
						TestArgs: "a1=v1 a2=v2",
						Keyvals: map[string]string{
							"k1": "v1",
							"k2": "v2",
						},
						IsClientTest: true,
						DisplayName:  "fancy-name",
					},
				},
			},
		}
		args := request.Args{
			TestRunnerRequest: &want,
		}
		Convey("when a BB request is formed", func() {
			req, err := args.NewBBRequest()
			So(err, ShouldBeNil)
			So(req, ShouldNotBeNil)
			Convey("it should contain the test runner request.", func() {
				So(req.Properties, ShouldNotBeNil)

				reqStruct, ok := req.Properties.Fields["request"]
				So(ok, ShouldBeTrue)

				m := jsonpb.Marshaler{}
				s, err := m.MarshalToString(reqStruct)
				So(err, ShouldBeNil)

				var got skylab_test_runner.Request
				err = jsonpb.UnmarshalString(s, &got)
				So(err, ShouldBeNil)

				diff := pretty.Compare(got, want)
				So(diff, ShouldBeEmpty)
			})
		})
	})
}

func TestTagsBB(t *testing.T) {
	Convey("Given request arguments that specify tags", t, func() {
		args := request.Args{
			SwarmingTags:      []string{"k1:v1", "k2:v2"},
			TestRunnerRequest: &skylab_test_runner.Request{},
		}
		Convey("when a request is formed", func() {
			req, err := args.NewBBRequest()
			So(err, ShouldBeNil)
			So(req, ShouldNotBeNil)
			Convey("then request should have correct tags.", func() {
				So(req.Tags, ShouldHaveLength, 2)

				want := []*buildbucket_pb.StringPair{
					{
						Key:   "k1",
						Value: "v1",
					},
					{
						Key:   "k2",
						Value: "v2",
					},
				}

				diff := pretty.Compare(sortBBStringPairs(req.Tags), sortBBStringPairs(want))
				So(diff, ShouldBeEmpty)
			})
		})
	})
}

func TestPriorityBB(t *testing.T) {
	Convey("Given request arguments that specify tags", t, func() {
		args := request.Args{
			Priority:          42,
			TestRunnerRequest: &skylab_test_runner.Request{},
		}
		Convey("when a request is formed", func() {
			req, err := args.NewBBRequest()
			So(err, ShouldBeNil)
			So(req, ShouldNotBeNil)
			Convey("then request should have correct priority.", func() {
				So(req.Priority, ShouldEqual, 42)
			})
		})
	})
}

func TestStatusTopicBB(t *testing.T) {
	Convey("Given request arguments that specify a Pubsub topic for status updates", t, func() {
		args := request.Args{
			StatusTopic:       "a topic name",
			TestRunnerRequest: &skylab_test_runner.Request{},
		}
		Convey("when a request is formed", func() {
			req, err := args.NewBBRequest()
			So(err, ShouldBeNil)
			So(req, ShouldNotBeNil)
			Convey("then request should have the Pubsub topic assigned.", func() {
				So(req.Notify, ShouldNotBeNil)
				So(req.Notify.PubsubTopic, ShouldEqual, "a topic name")
			})
		})
	})
}

func TestNoStatusTopicBB(t *testing.T) {
	Convey("Given request arguments that specify a Pubsub topic for status updates", t, func() {
		args := request.Args{
			TestRunnerRequest: &skylab_test_runner.Request{},
		}
		Convey("when a request is formed", func() {
			req, err := args.NewBBRequest()
			So(err, ShouldBeNil)
			So(req, ShouldNotBeNil)
			Convey("then request should have no notify field.", func() {
				So(req.Notify, ShouldBeNil)
			})
		})
	})
}

func TestProvisionableDimensions(t *testing.T) {
	Convey("Given request arguments that specify provisionable and regular dimenisons and inventory labels", t, func() {
		model := "foo-model"
		args := request.Args{
			Dimensions:              []string{"k1:v1"},
			ProvisionableDimensions: []string{"k2:v2", "k3:v3"},
			SchedulableLabels:       inventory.SchedulableLabels{Model: &model},
		}
		Convey("when a request is formed", func() {
			req, err := args.SwarmingNewTaskRequest()
			So(err, ShouldBeNil)
			So(req, ShouldNotBeNil)
			Convey("then request should have correct slice structure.", func() {
				So(req.TaskSlices, ShouldHaveLength, 2)

				// First slice requires all dimensions.
				// Second slice (fallback) requires only non-provisionable dimensions.
				s0 := req.TaskSlices[0]
				s1 := req.TaskSlices[1]
				So(s0.Properties.Dimensions, ShouldHaveLength, 6)
				So(s1.Properties.Dimensions, ShouldHaveLength, 4)

				s1Expect := toStringPairs([]string{
					"pool:ChromeOSSkylab",
					"dut_state:ready",
					fmt.Sprintf("label-model:%s", model),
					"k1:v1",
				})
				diff := pretty.Compare(sortDimensions(s1.Properties.Dimensions), sortDimensions(s1Expect))
				So(diff, ShouldBeEmpty)

				s0Expect := append(s1Expect, toStringPairs([]string{"k2:v2", "k3:v3"})...)
				diff = pretty.Compare(sortDimensions(s0.Properties.Dimensions), sortDimensions(s0Expect))
				So(diff, ShouldBeEmpty)

				// First slice command doesn't include provisioning.
				// Second slice (fallback) does.
				s0FlatCmd := strings.Join(s0.Properties.Command, " ")
				s1FlatCmd := strings.Join(s1.Properties.Command, " ")
				provString := "-provision-labels k2:v2,k3:v3"
				So(s0FlatCmd, ShouldNotContainSubstring, provString)
				So(s1FlatCmd, ShouldContainSubstring, provString)
			})
		})
	})
}

func TestStatusTopic(t *testing.T) {
	Convey("Given request arguments that specify a Pubsub topic for status updates", t, func() {
		args := request.Args{
			StatusTopic: "a topic name",
		}
		Convey("when a request is formed", func() {
			req, err := args.SwarmingNewTaskRequest()
			So(err, ShouldBeNil)
			So(req, ShouldNotBeNil)
			Convey("then request should have the Pubsub topic assigned.", func() {
				So(req.PubsubTopic, ShouldEqual, "a topic name")
			})
		})
	})
}

func TestSliceExpiration(t *testing.T) {
	timeout := 11 * time.Minute
	Convey("Given a request arguments with no provisionable dimensions", t, func() {
		args := request.Args{
			Timeout: timeout,
		}
		req, err := args.SwarmingNewTaskRequest()
		So(req, ShouldNotBeNil)
		So(err, ShouldBeNil)
		Convey("request should have a single slice with provided timeout.", func() {
			So(req.TaskSlices, ShouldHaveLength, 1)
			So(req.TaskSlices[0].ExpirationSecs, ShouldEqual, 60*11)
		})
	})
	Convey("Given a request arguments with provisionable dimensions", t, func() {
		args := request.Args{
			Timeout:                 timeout,
			ProvisionableDimensions: []string{"k1:v1"},
		}
		req, err := args.SwarmingNewTaskRequest()
		So(req, ShouldNotBeNil)
		So(err, ShouldBeNil)
		Convey("request should have 2 slices, with provided timeout on only the second.", func() {
			So(req.TaskSlices, ShouldHaveLength, 2)
			So(req.TaskSlices[0].ExpirationSecs, ShouldBeLessThan, 60*5)
			So(req.TaskSlices[1].ExpirationSecs, ShouldEqual, 60*11)
		})
	})
}

func TestStaticDimensions(t *testing.T) {
	cases := []struct {
		Tag  string
		Args request.Args
		Want []*swarming.SwarmingRpcsStringPair
	}{
		{
			Tag:  "empty args",
			Args: request.Args{},
			Want: toStringPairs([]string{"pool:ChromeOSSkylab"}),
		},
		{
			Tag: "one schedulable label",
			Args: request.Args{
				SchedulableLabels: inventory.SchedulableLabels{
					Model: stringPtr("some_model"),
				},
			},
			Want: toStringPairs([]string{"pool:ChromeOSSkylab", "label-model:some_model"}),
		},
		{
			Tag: "one dimension",
			Args: request.Args{
				Dimensions: []string{"some:value"},
			},
			Want: toStringPairs([]string{"pool:ChromeOSSkylab", "some:value"}),
		},
		{
			Tag: "one provisionable dimension",
			Args: request.Args{
				ProvisionableDimensions: []string{"cros-version:value"},
			},
			Want: toStringPairs([]string{"pool:ChromeOSSkylab"}),
		},
		{
			Tag: "one of each",
			Args: request.Args{
				SchedulableLabels: inventory.SchedulableLabels{
					Model: stringPtr("some_model"),
				},
				Dimensions:              []string{"some:value"},
				ProvisionableDimensions: []string{"cros-version:value"},
			},
			Want: toStringPairs([]string{"pool:ChromeOSSkylab", "label-model:some_model", "some:value"}),
		},
	}

	for _, c := range cases {
		t.Run(c.Tag, func(t *testing.T) {
			got, err := c.Args.StaticDimensions()
			if err != nil {
				t.Fatalf("error in StaticDimensions(): %s", err)
			}
			want := sortDimensions(c.Want)
			got = sortDimensions(got)
			if diff := pretty.Compare(want, got); diff != "" {
				t.Errorf("Incorrect static dimensions, -want +got: %s", diff)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func toStringPairs(ss []string) []*swarming.SwarmingRpcsStringPair {
	ret := make([]*swarming.SwarmingRpcsStringPair, len(ss))
	for i, s := range ss {
		p := strings.Split(s, ":")
		if len(p) != 2 {
			panic(fmt.Sprintf("Invalid dimension %s", s))
		}
		ret[i] = &swarming.SwarmingRpcsStringPair{
			Key:   p[0],
			Value: p[1],
		}
	}
	return ret
}

func sortBBStringPairs(dims []*buildbucket_pb.StringPair) []*buildbucket_pb.StringPair {
	sort.SliceStable(dims, func(i, j int) bool {
		return dims[i].Key < dims[j].Key || (dims[i].Key == dims[j].Key && dims[i].Value < dims[j].Value)
	})
	return dims
}

func sortBBDimensions(dims []*buildbucket_pb.RequestedDimension) []*buildbucket_pb.RequestedDimension {
	sort.SliceStable(dims, func(i, j int) bool {
		return dims[i].Key < dims[j].Key || (dims[i].Key == dims[j].Key && dims[i].Value < dims[j].Value)
	})
	return dims
}

func sortDimensions(dims []*swarming.SwarmingRpcsStringPair) []*swarming.SwarmingRpcsStringPair {
	sort.SliceStable(dims, func(i, j int) bool {
		return dims[i].Key < dims[j].Key || (dims[i].Key == dims[j].Key && dims[i].Value < dims[j].Value)
	})
	return dims
}

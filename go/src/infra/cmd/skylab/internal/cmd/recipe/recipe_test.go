// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recipe

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/kylelemons/godebug/pretty"
	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
)

func TestRequest(t *testing.T) {
	a := Args{
		Board:                      "foo-board",
		Image:                      "foo-image",
		Model:                      "foo-model",
		Pool:                       "foo-pool",
		TestPlan:                   NewTestPlanForAutotestTests("foo-arg1=val1 foo-arg2=val2", "foo-test-1", "foo-test-2"),
		Timeout:                    30 * time.Minute,
		Keyvals:                    map[string]string{"k1": "v1"},
		FreeformSwarmingDimensions: []string{"freeform-key:freeform-value"},
		MaxRetries:                 5,
		ProvisionLabels:            []string{"fwrw-version:foo-firmware"},
		LegacySuite:                "legacy-suite",
	}
	got, err := a.TestPlatformRequest()
	want := &test_platform.Request{
		Params: &test_platform.Request_Params{
			HardwareAttributes: &test_platform.Request_Params_HardwareAttributes{
				Model: "foo-model",
			},
			SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
				BuildTarget: &chromiumos.BuildTarget{
					Name: "foo-board",
				},
			},
			FreeformAttributes: &test_platform.Request_Params_FreeformAttributes{
				SwarmingDimensions: []string{"freeform-key:freeform-value"},
			},
			SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
				{
					Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "foo-image"},
				},
				{
					Dep: &test_platform.Request_Params_SoftwareDependency_RwFirmwareBuild{RwFirmwareBuild: "foo-firmware"},
				},
			},
			Scheduling: &test_platform.Request_Params_Scheduling{
				Pool: &test_platform.Request_Params_Scheduling_UnmanagedPool{
					UnmanagedPool: "foo-pool",
				},
			},
			Metadata: &test_platform.Request_Params_Metadata{
				TestMetadataUrl:        "gs://chromeos-image-archive/foo-image",
				DebugSymbolsArchiveUrl: "gs://chromeos-image-archive/foo-image",
			},
			Time: &test_platform.Request_Params_Time{
				MaximumDuration: &duration.Duration{
					Nanos:   0,
					Seconds: 1800,
				},
			},
			Decorations: &test_platform.Request_Params_Decorations{
				AutotestKeyvals: map[string]string{"k1": "v1"},
			},
			Retry: &test_platform.Request_Params_Retry{
				Allow: true,
				Max:   5,
			},
			Legacy: &test_platform.Request_Params_Legacy{
				AutotestSuite: "legacy-suite",
			},
		},
		TestPlan: &test_platform.Request_TestPlan{
			Test: []*test_platform.Request_Test{
				{Harness: &test_platform.Request_Test_Autotest_{Autotest: &test_platform.Request_Test_Autotest{Name: "foo-test-1", TestArgs: "foo-arg1=val1 foo-arg2=val2"}}},
				{Harness: &test_platform.Request_Test_Autotest_{Autotest: &test_platform.Request_Test_Autotest{Name: "foo-test-2", TestArgs: "foo-arg1=val1 foo-arg2=val2"}}},
			},
		},
	}
	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("Unexpected diff (-got +want): %s", diff)
	}
	if err != nil {
		t.Errorf("Unexpected error %s", err)
	}
}

func TestSchedulingParam(t *testing.T) {
	Convey("Given a", t, func() {
		cases := []struct {
			name                  string
			inputPool             string
			inputAccount          string
			inputPriority         int64
			expectedAccount       string
			expectedManagedPool   test_platform.Request_Params_Scheduling_ManagedPool
			expectedUnmanagedPool string
			expectedPriority      int64
		}{
			{
				name:                  "unmanaged pool with Quota Account",
				inputAccount:          "foo account",
				inputPriority:         142,
				inputPool:             "foo-pool",
				expectedAccount:       "foo account",
				expectedPriority:      0,
				expectedUnmanagedPool: "foo-pool",
			},
			{
				name:                  "unmanaged pool without Quota Account",
				inputPriority:         142,
				inputPool:             "foo-pool",
				expectedPriority:      142,
				expectedUnmanagedPool: "foo-pool",
			},
			{
				name:                "long-named managed pool",
				inputPool:           "MANAGED_POOL_CQ",
				expectedManagedPool: test_platform.Request_Params_Scheduling_MANAGED_POOL_CQ,
			},
			{
				name:                "skylab-named managed pool",
				inputPool:           "DUT_POOL_CQ",
				expectedManagedPool: test_platform.Request_Params_Scheduling_MANAGED_POOL_CQ,
			},
			{
				name:                "short-named managed pool",
				inputPool:           "cq",
				expectedManagedPool: test_platform.Request_Params_Scheduling_MANAGED_POOL_CQ,
			},
			{
				// CTS is a managed pool but not enabled Quota Scheduler.
				name:                "CTS pool with priority",
				inputPool:           "cts",
				inputPriority:       142,
				expectedManagedPool: test_platform.Request_Params_Scheduling_MANAGED_POOL_CTS,
				expectedPriority:    142,
			},
		}
		for _, c := range cases {
			Convey(c.name, func() {
				s := toScheduling(c.inputPool, c.inputAccount, c.inputPriority)
				Convey("then scheduling parameters are correct.", func() {
					So(s.GetManagedPool(), ShouldResemble, c.expectedManagedPool)
					So(s.GetUnmanagedPool(), ShouldResemble, c.expectedUnmanagedPool)
					So(s.Priority, ShouldEqual, c.expectedPriority)
					So(s.QsAccount, ShouldEqual, c.expectedAccount)
				})
			})
		}
	})
}

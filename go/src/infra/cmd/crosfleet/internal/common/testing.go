package common

import (
	"github.com/golang/protobuf/ptypes/duration"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
)

// CmpOpts enables comparisons of the listed protos with unexported fields.
var CmpOpts = cmpopts.IgnoreUnexported(
	buildbucketpb.Build{},
	buildbucketpb.RequestedDimension{},
	buildbucketpb.StringPair{},
	chromiumos.BuildTarget{},
	duration.Duration{},
	test_platform.Request{},
	test_platform.Request_TestPlan{},
	test_platform.Request_Params{},
	test_platform.Request_Params_HardwareAttributes{},
	test_platform.Request_Params_SoftwareAttributes{},
	test_platform.Request_Params_SoftwareDependency{},
	test_platform.Request_Params_FreeformAttributes{},
	test_platform.Request_Params_Scheduling{},
	test_platform.Request_Params_Decorations{},
	test_platform.Request_Params_Retry{},
	test_platform.Request_Params_Metadata{},
	test_platform.Request_Params_Time{},
	test_platform.ServiceVersion{},
	structpb.ListValue{},
	structpb.Struct{},
	structpb.Value{})

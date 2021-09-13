package testorchestrator_test

import (
	"context"
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/testorchestrator"

	"github.com/google/go-cmp/cmp"
	testpb "go.chromium.org/chromiumos/config/go/test/api"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestGetRequestedDimensions(t *testing.T) {
	ctx := context.Background()

	dutCriteria := []*testpb.DutCriterion{
		{
			AttributeId: &testpb.DutAttribute_Id{
				Value: "dutattr1",
			},
			Values: []string{"valA", "valB"},
		}, {
			AttributeId: &testpb.DutAttribute_Id{
				Value: "dutattr2",
			},
			Values: []string{"valC"},
		},
	}

	expectedDims := []*bbpb.RequestedDimension{
		{
			Key:   "dutattr1",
			Value: "valA",
		},
		{
			Key:   "dutattr2",
			Value: "valC",
		},
	}

	dims, err := testorchestrator.GetRequestedDimensions(ctx, dutCriteria)

	assert.NilError(t, err)

	if diff := cmp.Diff(expectedDims, dims, protocmp.Transform()); diff != "" {
		t.Errorf("GetRequestedDimensions(%s) returned unexpected diff (-want +got):\n%s", dutCriteria, diff)
	}
}

func TestGetRequestedDimensionsErrors(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		dutCriteria  []*testpb.DutCriterion
		errorMessage string
	}{
		{
			"no id",
			[]*testpb.DutCriterion{
				{
					Values: []string{"valA"},
				},
			},
			"criteria must have DutAttribute id set",
		},
		{
			"no values",
			[]*testpb.DutCriterion{
				{
					AttributeId: &testpb.DutAttribute_Id{
						Value: "dutattr1",
					},
					Values: []string{},
				},
			},
			"at least one value must be set on DutAttributes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := testorchestrator.GetRequestedDimensions(ctx, tc.dutCriteria)
			assert.ErrorContains(t, err, tc.errorMessage)
		})
	}
}

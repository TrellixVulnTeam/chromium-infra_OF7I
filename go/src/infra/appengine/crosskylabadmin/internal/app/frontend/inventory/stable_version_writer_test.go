package inventory

import (
	"testing"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"

	"github.com/google/go-cmp/cmp"
)

// TestSetSatlabStableVersion tests that SetSatlabStableVersion returns a not-yet-implemented response.
func TestSetSatlabStableVersion(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = withSplitInventory(ctx)
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	expected := "rpc error: code = Unimplemented desc = SetSatlabStableVersion not yet implemented"
	_, err := tf.Inventory.SetSatlabStableVersion(ctx, &fleet.SetSatlabStableVersionRequest{
		Strategy: &fleet.SetSatlabStableVersionRequest_SatlabHostnameStrategy{
			SatlabHostnameStrategy: &fleet.SatlabHostnameStrategy{
				Hostname: "satlab-host1",
			},
		},
	})

	actual := ""
	if err != nil {
		actual = err.Error()
	}

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("unexpected diff (-want +got): %s", diff)
	}
}

// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/gae/service/datastore"

	// See https://bugs.chromium.org/p/chromium/issues/detail?id=1242998 for details.
	// TODO(gregorynisbet): Remove this once new behavior is default.
	_ "go.chromium.org/luci/gae/service/datastore/crbug1242998safeget"

	kartepb "infra/cros/karte/api"
	"infra/cros/karte/internal/idstrategy"
	"infra/cros/karte/internal/scalars"
)

// TestModifyingSealedActionShouldFail tests that updating a record after the seal time fails.
func TestModifyingSealedActionShouldFail(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	ctx = idstrategy.Use(ctx, idstrategy.NewDefault())
	testClock := testclock.New(time.Unix(3, 4))
	ctx = clock.Set(ctx, testClock)

	k := NewKarteFrontend()

	k.CreateAction(ctx, &kartepb.CreateActionRequest{
		Action: &kartepb.Action{
			Kind: "w",
		},
	})

	resp, err := k.ListActions(ctx, &kartepb.ListActionsRequest{
		Filter: `kind == "w"`,
	})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if l := len(resp.GetActions()); l != 1 {
		t.Errorf("unexpected number of actions %d", l)
	}

	name := resp.GetActions()[0].GetName()

	sealTime := scalars.ConvertTimestampPtrToString(resp.GetActions()[0].GetSealTime())
	if diff := cmp.Diff(fmt.Sprintf("%d:%d", 3+12*60*60, 0), sealTime); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}

	testClock = testclock.New(time.Unix(13*60*60, 0))
	ctx = clock.Set(ctx, testClock)

	_, err = k.UpdateAction(ctx, &kartepb.UpdateActionRequest{
		Action: &kartepb.Action{
			Name: name,
			Kind: "e",
		},
	},
	)
	if err == nil {
		t.Errorf("update should have failed but didn't")
	}
}

// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/api"
	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
)

func mockConfigBundle(id string, programId string, name string) *payload.ConfigBundle {
	return &payload.ConfigBundle{
		DesignList: []*api.Design{
			{
				Id: &api.DesignId{
					Value: id,
				},
				ProgramId: &api.ProgramId{
					Value: programId,
				},
				Name: name,
			},
		},
	}
}

func TestUpdateConfigBundle(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	t.Run("update non-existent ConfigBundle", func(t *testing.T) {
		want := mockConfigBundle("design1", "program1", "name1")
		got, err := UpdateConfigBundle(ctx, want)
		if err != nil {
			t.Fatalf("UpdateConfigBundle failed: %s", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("UpdateConfigBundle returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("update existent ConfigBundle", func(t *testing.T) {
		cb2 := mockConfigBundle("design2", "program2", "name2")
		cb2update := mockConfigBundle("design2", "program2", "name2update")

		// Insert cb2 into datastore
		_, _ = UpdateConfigBundle(ctx, cb2)

		// Update cb2
		got, err := UpdateConfigBundle(ctx, cb2update)
		if err != nil {
			t.Fatalf("UpdateConfigBundle failed: %s", err)
		}
		if diff := cmp.Diff(cb2update, got); diff != "" {
			t.Errorf("UpdateConfigBundle returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("update ConfigBundle with invalid IDs", func(t *testing.T) {
		cb3 := mockConfigBundle("", "", "")
		got, err := UpdateConfigBundle(ctx, cb3)
		if err == nil {
			t.Errorf("UpdateConfigBundle succeeded with empty IDs")
		}
		if c := status.Code(err); c != codes.Internal {
			t.Errorf("Unexpected error when calling UpdateConfigBundle: %s", err)
		}

		var cbNil *payload.ConfigBundle = nil
		if diff := cmp.Diff(cbNil, got); diff != "" {
			t.Errorf("UpdateConfigBundle returned unexpected diff (-want +got):\n%s", diff)
		}
	})
}

func TestGetConfigBundle(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	t.Run("get ConfigBundle by existing ID", func(t *testing.T) {
		want := mockConfigBundle("design1", "program1", "name1")
		_, err := UpdateConfigBundle(ctx, want)
		if err != nil {
			t.Fatalf("UpdateConfigBundle failed: %s", err)
		}

		got, err := GetConfigBundle(ctx, "program1-design1")
		if err != nil {
			t.Fatalf("GetConfigBundle failed: %s", err)
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("GetConfigBundle returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("get ConfigBundle by non-existent ID", func(t *testing.T) {
		id := "program2-design2"
		_, err := GetConfigBundle(ctx, id)
		if err == nil {
			t.Errorf("GetConfigBundle succeeded with non-existent ID: %s", id)
		}
		if c := status.Code(err); c != codes.NotFound {
			t.Errorf("Unexpected error when calling GetConfigBundle: %s", err)
		}
	})

	t.Run("get ConfigBundle by invalid ID", func(t *testing.T) {
		id := "program3-design3-extraid3"
		_, err := GetConfigBundle(ctx, id)
		if err == nil {
			t.Errorf("GetConfigBundle succeeded with invalid ID: %s", id)
		}
		if c := status.Code(err); c != codes.InvalidArgument {
			t.Errorf("Unexpected error when calling GetConfigBundle: %s", err)
		}
	})
}

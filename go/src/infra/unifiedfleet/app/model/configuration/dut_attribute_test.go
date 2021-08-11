// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
)

func mockDutAttribute(id string, field_path string) *api.DutAttribute {
	return &api.DutAttribute{
		Id: &api.DutAttribute_Id{
			Value: id,
		},
		FieldPath: field_path,
	}
}

func TestUpdateDutAttribute(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	t.Run("update non-existent DutAttribute", func(t *testing.T) {
		want := mockDutAttribute("attr1", "test.path.1")
		got, err := UpdateDutAttribute(ctx, want)
		if err != nil {
			t.Fatalf("UpdateDutAttribute failed: %s", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("UpdateDutAttribute returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("update existent DutAttribute", func(t *testing.T) {
		attr2 := mockDutAttribute("attr_2", "test.path.2")
		attr2update := mockDutAttribute("attr_2", "test.path.2.update")

		// Insert attr2 into datastore
		_, _ = UpdateDutAttribute(ctx, attr2)

		// Update attr2
		got, err := UpdateDutAttribute(ctx, attr2update)
		if err != nil {
			t.Fatalf("UpdateDutAttribute failed: %s", err)
		}
		if diff := cmp.Diff(attr2update, got); diff != "" {
			t.Errorf("UpdateDutAttribute returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("update DutAttribute with empty IDs", func(t *testing.T) {
		attr3 := mockDutAttribute("", "")
		got, err := UpdateDutAttribute(ctx, attr3)
		if err == nil {
			t.Errorf("UpdateDutAttribute succeeded with empty IDs")
		}
		if c := status.Code(err); c != codes.Internal {
			t.Errorf("Unexpected error when calling UpdateDutAttribute: %s", err)
		}

		var attrNil *api.DutAttribute = nil
		if diff := cmp.Diff(attrNil, got); diff != "" {
			t.Errorf("UpdateDutAttribute returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("update DutAttribute with invalid IDs", func(t *testing.T) {
		attr4 := mockDutAttribute("attr-4", "test.path.4")
		got, err := UpdateDutAttribute(ctx, attr4)
		if err == nil {
			t.Errorf("UpdateDutAttribute succeeded with invalid ID")
		}
		if c := status.Code(err); c != codes.Internal {
			t.Errorf("Unexpected error when calling UpdateDutAttribute: %s", err)
		}

		var attrNil *api.DutAttribute = nil
		if diff := cmp.Diff(attrNil, got); diff != "" {
			t.Errorf("UpdateDutAttribute returned unexpected diff (-want +got):\n%s", diff)
		}
	})
}

func TestGetDutAttribute(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	t.Run("get DutAttribute by existing ID", func(t *testing.T) {
		want := mockDutAttribute("attr1", "test.path.1")
		_, err := UpdateDutAttribute(ctx, want)
		if err != nil {
			t.Fatalf("UpdateDutAttribute failed: %s", err)
		}

		got, err := GetDutAttribute(ctx, "attr1")
		if err != nil {
			t.Fatalf("GetDutAttribute failed: %s", err)
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("GetDutAttribute returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("get DutAttribute by non-existent ID", func(t *testing.T) {
		id := "attr2"
		_, err := GetDutAttribute(ctx, id)
		if err == nil {
			t.Errorf("GetDutAttribute succeeded with non-existent ID: %s", id)
		}
		if c := status.Code(err); c != codes.NotFound {
			t.Errorf("Unexpected error when calling GetDutAttribute: %s", err)
		}
	})
}

func TestListDutAttributes(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	wantFull := make([]*api.DutAttribute, 0, 4)
	wantKeys := make([]*api.DutAttribute, 0, 4)
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("attr%d", i)
		da := mockDutAttribute(id, fmt.Sprintf("test.path.%d", i))
		rsp, err := UpdateDutAttribute(ctx, da)
		if err != nil {
			t.Fatalf("UpdateDutAttribute failed: %s", err)
		}
		wantFull = append(wantFull, rsp)
		wantKeys = append(wantKeys, &api.DutAttribute{
			Id: &api.DutAttribute_Id{
				Value: id,
			},
		})
	}

	t.Run("list DutAttributes happy path; keysOnly false", func(t *testing.T) {
		got, err := ListDutAttributes(ctx, false)
		if err != nil {
			t.Fatalf("ListDutAttributes failed: %s", err)
		}
		if diff := cmp.Diff(wantFull, got, protocmp.Transform()); diff != "" {
			t.Errorf("ListDutAttributes returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("list DutAttributes happy path; keysOnly true", func(t *testing.T) {
		got, err := ListDutAttributes(ctx, true)
		if err != nil {
			t.Fatalf("ListDutAttributes failed: %s", err)
		}
		if diff := cmp.Diff(wantKeys, got, protocmp.Transform()); diff != "" {
			t.Errorf("ListDutAttributes returned unexpected diff (-want +got):\n%s", diff)
		}
	})
}

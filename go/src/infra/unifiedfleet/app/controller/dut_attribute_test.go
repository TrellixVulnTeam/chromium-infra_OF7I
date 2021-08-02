// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	"infra/unifiedfleet/app/model/configuration"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
)

func mockDutAttribute(id string, fieldPath string) *api.DutAttribute {
	return &api.DutAttribute{
		Id: &api.DutAttribute_Id{
			Value: id,
		},
		FieldPath: fieldPath,
	}
}

func TestGetDutAttribute(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	datastore.GetTestable(ctx).Consistent(true)

	t.Run("get DutAttribute by existing ID", func(t *testing.T) {
		want := mockDutAttribute("attr1", "test.path.1")
		_, err := configuration.UpdateDutAttribute(ctx, want)
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
	ctx := testingContext()
	datastore.GetTestable(ctx).Consistent(true)

	wantFull := make([]*api.DutAttribute, 0, 4)
	wantKeys := make([]*api.DutAttribute, 0, 4)
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("attr%d", i)
		da := mockDutAttribute(id, fmt.Sprintf("test.path.%d", i))
		rsp, err := configuration.UpdateDutAttribute(ctx, da)
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

// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package asset

import (
	"context"
	reflect "reflect"
	"testing"

	"go.chromium.org/luci/gae/impl/memory"
)

func TestAssetCreateWithValidData(t *testing.T) {
	ctx := memory.Use(context.Background())
	req := &CreateAssetRequest{
		Name:        "Test Asset",
		Description: "Test Asset description",
	}

	handler := &AssetHandler{}
	entity, err := handler.Create(ctx, req)
	if err != nil {
		t.Errorf("TestAssetCreateWithValidData error = %v", err)
	}
	if entity.GetName() != req.GetName() ||
		entity.GetDescription() != req.GetDescription() {
		t.Errorf("TestAssetCreateWithValidData() = recieved %v, send %v", entity, req)
	}
}

func TestAssetCreateWithInvalidName(t *testing.T) {
	ctx := memory.Use(context.Background())
	req := &CreateAssetRequest{
		Name:        "",
		Description: "Test Asset description",
	}

	handler := &AssetHandler{}
	entity, err := handler.Create(ctx, req)
	if err == nil {
		t.Errorf("TestAssetCreateWithInvalidName created entity = %v", entity)
	}
}

func TestAssetCreateWithInvalidDescription(t *testing.T) {
	ctx := memory.Use(context.Background())
	req := &CreateAssetRequest{
		Name:        "Test Asset",
		Description: "",
	}

	handler := &AssetHandler{}
	entity, err := handler.Create(ctx, req)
	if err == nil {
		t.Errorf("TestAssetCreateWithInvalidDescription created entity = %v", entity)
	}
}

func TestGetAssetWithValidData(t *testing.T) {
	ctx := memory.Use(context.Background())
	createRequest := &CreateAssetRequest{
		Name:        "Test Asset",
		Description: "Test Asset description",
	}

	handler := &AssetHandler{}
	entity, err := handler.Create(ctx, createRequest)
	if err != nil {
		t.Errorf("TestGetAssetWithValidData error = %v", err)
	}
	getRequest := &GetAssetRequest{
		AssetId: entity.GetAssetId(),
	}
	readEntity, err := handler.Get(ctx, getRequest)

	if err != nil {
		t.Errorf("TestGetAssetWithValidData error = %v", err)
	}
	if !reflect.DeepEqual(readEntity, entity) {
		t.Errorf("TestGetAssetWithValidData() = recieved %v, send %v", readEntity, entity)
	}
}

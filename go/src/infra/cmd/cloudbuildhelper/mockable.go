// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"io"

	"infra/cmd/cloudbuildhelper/cloudbuild"
	"infra/cmd/cloudbuildhelper/registry"
	"infra/cmd/cloudbuildhelper/storage"
)

// Collection of interfaces that mimic external APIs we use to simplify tests.
//
// Mocks are implemented in mockable_test.go.

// storageImpl is implemented by *storage.Storage.
type storageImpl interface {
	Check(ctx context.Context, name string) (*storage.Object, error)
	Upload(ctx context.Context, name, digest string, r io.Reader) (*storage.Object, error)
	UpdateMetadata(ctx context.Context, obj *storage.Object, cb func(m *storage.Metadata) error) error
}

// builderImpl is implemented by *cloudbuild.Builder.
type builderImpl interface {
	Trigger(ctx context.Context, r cloudbuild.Request) (*cloudbuild.Build, error)
	Check(ctx context.Context, bid string) (*cloudbuild.Build, error)
}

// registryImpl is implemented by *registry.Client.
type registryImpl interface {
	GetImage(ctx context.Context, image string) (*registry.Image, error)
	TagImage(ctx context.Context, img *registry.Image, tag string) error
}

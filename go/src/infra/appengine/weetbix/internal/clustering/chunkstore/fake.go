// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package chunkstore

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"

	"google.golang.org/protobuf/proto"

	cpb "infra/appengine/weetbix/internal/clustering/proto"
)

// FakeClient provides a fake implementation of a blob store, for testing.
// Data blobs are stored in-memory.
type FakeClient struct {
	Blobs map[string][]byte
}

// NewFakeClient initialises a new FakeClient.
func NewFakeClient() *FakeClient {
	return &FakeClient{
		Blobs: make(map[string][]byte),
	}
}

// Put saves the given chunk to storage. If successful, it returns
// the randomly-assigned ID of the created object.
func (fc *FakeClient) Put(ctx context.Context, project string, content *cpb.Chunk) (string, error) {
	if err := validateProject(project); err != nil {
		return "", err
	}
	b, err := proto.Marshal(content)
	if err != nil {
		return "", errors.Annotate(err, "marhsalling chunk").Err()
	}
	objID, err := generateObjectID()
	if err != nil {
		return "", err
	}
	name := fileName(project, objID)
	if _, ok := fc.Blobs[name]; ok {
		// Indicates a test with poorly seeded randomness.
		return "", errors.New("file already exists")
	}
	fc.Blobs[name] = b
	return objID, nil
}

// Get retrieves the chunk with the specified object ID and returns it.
func (fc *FakeClient) Get(ctx context.Context, project, objectID string) (*cpb.Chunk, error) {
	if err := validateProject(project); err != nil {
		return nil, err
	}
	if err := validateObjectID(objectID); err != nil {
		return nil, err
	}
	name := fileName(project, objectID)
	b, ok := fc.Blobs[name]
	if !ok {
		return nil, fmt.Errorf("blob does not exist: %q", name)
	}
	content := &cpb.Chunk{}
	if err := proto.Unmarshal(b, content); err != nil {
		return nil, errors.Annotate(err, "unmarshal chunk").Err()
	}
	return content, nil
}

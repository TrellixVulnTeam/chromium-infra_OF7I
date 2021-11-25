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

// FakeClient provides a fake implementation of a chunk store, for testing.
// Chunks are stored in-memory.
type FakeClient struct {
	// Contents are the chunk stored in the store, by their file name.
	// File names can be obtained using the FileName method.
	Contents map[string]*cpb.Chunk

	// A callback function to be called during Get(...). This allows
	// the test to change the environment during the processing of
	// a particular chunk.
	GetCallack func(objectID string)
}

// NewFakeClient initialises a new FakeClient.
func NewFakeClient() *FakeClient {
	return &FakeClient{
		Contents: make(map[string]*cpb.Chunk),
	}
}

// Put saves the given chunk to storage. If successful, it returns
// the randomly-assigned ID of the created object.
func (fc *FakeClient) Put(ctx context.Context, project string, content *cpb.Chunk) (string, error) {
	if err := validateProject(project); err != nil {
		return "", err
	}
	_, err := proto.Marshal(content)
	if err != nil {
		return "", errors.Annotate(err, "marhsalling chunk").Err()
	}
	objID, err := generateObjectID()
	if err != nil {
		return "", err
	}
	name := FileName(project, objID)
	if _, ok := fc.Contents[name]; ok {
		// Indicates a test with poorly seeded randomness.
		return "", errors.New("file already exists")
	}
	fc.Contents[name] = proto.Clone(content).(*cpb.Chunk)
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
	name := FileName(project, objectID)
	content, ok := fc.Contents[name]
	if !ok {
		return nil, fmt.Errorf("blob does not exist: %q", name)
	}
	if fc.GetCallack != nil {
		fc.GetCallack(objectID)
	}
	return proto.Clone(content).(*cpb.Chunk), nil
}

// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"infra/cmd/cloudbuildhelper/cloudbuild"
	"infra/cmd/cloudbuildhelper/registry"
	"infra/cmd/cloudbuildhelper/storage"
)

type storageImplMock struct {
	gen   int64
	blobs map[string]objBlob
}

type objBlob struct {
	storage.Object
	Blob []byte
}

func newStorageImplMock() *storageImplMock {
	return &storageImplMock{
		blobs: make(map[string]objBlob, 0),
	}
}

func (s *storageImplMock) Check(ctx context.Context, name string) (*storage.Object, error) {
	itm, ok := s.blobs[name]
	if !ok {
		return nil, nil
	}
	return &itm.Object, nil
}

func (s *storageImplMock) Upload(ctx context.Context, name, digest string, r io.Reader) (*storage.Object, error) {
	h := sha256.New()
	blob, err := ioutil.ReadAll(io.TeeReader(r, h))
	if err != nil {
		return nil, err
	}
	if d := hex.EncodeToString(h.Sum(nil)); d != digest {
		return nil, fmt.Errorf("got digest %q, expecting %q", d, digest)
	}

	s.gen++

	obj := storage.Object{
		Bucket:     testBucketName,
		Name:       name,
		Generation: s.gen,
		Metadata:   &storage.Metadata{},
	}

	s.blobs[name] = objBlob{
		Object: obj,
		Blob:   blob,
	}
	return &obj, nil
}

func (s *storageImplMock) UpdateMetadata(ctx context.Context, obj *storage.Object, cb func(m *storage.Metadata) error) error {
	itm, ok := s.blobs[obj.Name]
	if !ok {
		return fmt.Errorf("can't update metadata of %q: no such object", obj.Name)
	}

	md := itm.Metadata.Clone()
	if err := cb(md); err != nil || itm.Metadata.Equal(md) {
		return err
	}
	itm.Metadata = md

	s.blobs[obj.Name] = itm
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type builderImplMock struct {
	// Can be touched.
	checkCallback func(b *runningBuild) error
	finalStatus   cloudbuild.Status
	provenance    func(gs string) string  // gs://.... => SHA256
	outputDigests func(img string) string // full image name => "sha256:..."

	// Shouldn't be touched.
	registry *registryImplMock
	nextID   int64
	builds   map[string]runningBuild
}

type runningBuild struct {
	cloudbuild.Build
	Request cloudbuild.Request
}

func newBuilderImplMock(r *registryImplMock) *builderImplMock {
	bld := &builderImplMock{
		builds:        make(map[string]runningBuild, 0),
		finalStatus:   cloudbuild.StatusSuccess,
		provenance:    func(string) string { return "" },
		outputDigests: func(string) string { return "" },
		registry:      r,
	}

	// By default just advance the build through the stages.
	bld.checkCallback = func(b *runningBuild) error {
		switch b.Status {
		case cloudbuild.StatusQueued:
			b.Status = cloudbuild.StatusWorking
		case cloudbuild.StatusWorking:
			b.Status = bld.finalStatus
			if b.Status == cloudbuild.StatusSuccess {
				b.InputHashes = map[string]string{
					b.Request.Source.String(): bld.provenance(b.Request.Source.String()),
				}
				b.OutputImage = b.Request.Image
				if b.Request.Image != "" {
					b.OutputDigest = bld.outputDigests(b.Request.Image)
				}
			}
		}
		return nil
	}

	return bld
}

func (b *builderImplMock) Trigger(ctx context.Context, r cloudbuild.Request) (*cloudbuild.Build, error) {
	b.nextID++
	build := cloudbuild.Build{
		ID:     fmt.Sprintf("b-%d", b.nextID),
		Status: cloudbuild.StatusQueued,
		LogURL: testLogURL,
	}
	b.builds[build.ID] = runningBuild{
		Build:   build,
		Request: r,
	}
	return &build, nil
}

func (b *builderImplMock) Check(ctx context.Context, bid string) (*cloudbuild.Build, error) {
	build, ok := b.builds[bid]
	if !ok {
		return nil, fmt.Errorf("no build %q", bid)
	}
	if err := b.checkCallback(&build); err != nil {
		return nil, err
	}
	b.builds[bid] = build
	if build.Status == cloudbuild.StatusSuccess && build.Request.Image != "" {
		b.registry.put(build.Request.Image, build.OutputDigest)
	}
	return &build.Build, nil
}

////////////////////////////////////////////////////////////////////////////////

type registryImplMock struct {
	imgs map[string]registry.Image // <image>[:<tag>|@<digest>] => Image
}

func newRegistryImplMock() *registryImplMock {
	return &registryImplMock{
		imgs: make(map[string]registry.Image, 0),
	}
}

// put takes "<name>:<tag> => <digest>" image and puts it in the registry.
func (r *registryImplMock) put(image, digest string) {
	if !strings.HasPrefix(digest, "sha256:") {
		panic(digest)
	}

	var name, tag string
	switch chunks := strings.Split(image, ":"); {
	case len(chunks) == 1:
		name, tag = chunks[0], "latest"
	case len(chunks) == 2:
		name, tag = chunks[0], chunks[1]
	default:
		panic(image)
	}

	img := registry.Image{
		Registry:    "...",
		Repo:        name,
		Digest:      digest,
		RawManifest: []byte(fmt.Sprintf("raw manifest of %q", digest)),
	}

	r.imgs[fmt.Sprintf("%s@%s", name, digest)] = img
	r.imgs[fmt.Sprintf("%s:%s", name, tag)] = img
}

func (r *registryImplMock) GetImage(ctx context.Context, image string) (*registry.Image, error) {
	img, ok := r.imgs[image]
	if !ok {
		return nil, &registry.Error{
			Errors: []registry.InnerError{
				{Code: "MANIFEST_UNKNOWN"},
			},
		}
	}
	return &img, nil
}

func (r *registryImplMock) TagImage(ctx context.Context, img *registry.Image, tag string) error {
	r.imgs[fmt.Sprintf("%s:%s", img.Repo, tag)] = *img
	return nil
}

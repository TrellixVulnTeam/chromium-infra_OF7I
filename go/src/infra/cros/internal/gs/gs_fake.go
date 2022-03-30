// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package gs

import (
	"context"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"infra/cros/internal/assert"
	"infra/cros/internal/shared"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/gcloud/gs"
)

type FakeClient struct {
	T *testing.T
	// ExpectedLists is indexed by bucket and then by prefix.
	ExpectedLists     map[string]map[string][]string
	ExpectedWrites    map[string][]byte
	ExpectedDownloads map[string][]byte
	ExpectedReads     map[string][]byte
	ExpectedSetTTL    map[string]time.Duration
	ExpectedMetadata  map[string]map[string]string
}

// WriteFileToGS writes the specified data to the specified gs path.
func (f *FakeClient) WriteFileToGS(gsPath gs.Path, data []byte) error {
	expected, ok := f.ExpectedWrites[string(gsPath)]
	if !ok {
		f.T.Fatalf("unexpected write at %s", string(gsPath))
	}
	if !reflect.DeepEqual(expected, data) {
		f.T.Fatalf("mismatch for write at %s: expected:\n%v\ngot:\n%v\n", string(gsPath), string(expected), string(data))
	}
	return nil
}

func (f *FakeClient) Download(gsPath gs.Path, localPath string) error {
	data, ok := f.ExpectedDownloads[string(gsPath)]
	if !ok {
		f.T.Fatalf("unexpected download of file %s", gsPath)
	}
	if data == nil {
		return errors.Annotate(shared.ErrObjectNotExist, "download").Err()
	}
	assert.NilError(f.T, ioutil.WriteFile(localPath, data, 0644))
	return nil
}

func (f *FakeClient) DownloadWithGsutil(_ context.Context, gsPath gs.Path, localPath string) error {
	data, ok := f.ExpectedDownloads[string(gsPath)]
	if !ok {
		f.T.Fatalf("unexpected download of file %s", gsPath)
	}
	if data == nil {
		return errors.Annotate(shared.ErrObjectNotExist, "download").Err()
	}
	assert.NilError(f.T, ioutil.WriteFile(localPath, data, 0644))
	return nil
}

func (f *FakeClient) Read(gsPath gs.Path) ([]byte, error) {
	data, ok := f.ExpectedReads[string(gsPath)]
	if !ok {
		f.T.Fatalf("unexpected read of file %s", gsPath)
	}
	return data, nil
}

func (f *FakeClient) List(_ context.Context, bucket string, prefix string) ([]string, error) {
	bucketData, ok := f.ExpectedLists[bucket]
	if !ok {
		f.T.Fatalf("unexpected list of bucket %s", bucket)
	}
	data, ok := bucketData[prefix]
	if !ok {
		f.T.Fatalf("unexpected list of bucket %s, prefix %s", bucket, prefix)
	}
	return data, nil
}

func (f *FakeClient) SetTTL(_ context.Context, gsPath gs.Path, ttl time.Duration) error {
	expectedTTL, ok := f.ExpectedSetTTL[string(gsPath)]
	if !ok {
		f.T.Fatalf("unexpected call to SetTTL for object %s, ttl %v", gsPath, ttl)
	}
	if !reflect.DeepEqual(expectedTTL, ttl) {
		f.T.Fatalf("mismatch on call SetTTL for %s: expected:\n%v\ngot:\n%v\n", string(gsPath), expectedTTL, ttl)
	}
	return nil
}

func (f *FakeClient) SetMetadata(ctx context.Context, gsPath gs.Path, key, value string) error {
	expectedMetadata, ok := f.ExpectedMetadata[string(gsPath)]
	if !ok {
		f.T.Fatalf("unexpected call to SetTTL for object %s", gsPath)
	}
	expected, ok := expectedMetadata[key]
	if !ok {
		f.T.Fatalf("unexpected call to SetTTL for object %s, key %s", gsPath, key)
	}
	if expected != value {
		f.T.Fatalf("mismatch on call SetMetadata for %s, key %s: expected:\n%v\ngot:%v\n", gsPath, key, expected, value)
	}
	return nil
}

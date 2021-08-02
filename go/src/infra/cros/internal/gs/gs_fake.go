// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package gs

import (
	"context"
	"io/ioutil"
	"reflect"
	"testing"

	"infra/cros/internal/assert"

	"cloud.google.com/go/storage"
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
		return errors.Annotate(storage.ErrObjectNotExist, "download").Err()
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

// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package gs

import (
	"io/ioutil"
	"reflect"
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/repo"

	"go.chromium.org/luci/common/gcloud/gs"
)

type FakeClient struct {
	T                 *testing.T
	ExpectedWrites    map[string]*repo.Manifest
	ExpectedDownloads map[string][]byte
}

// WriteFileToGS writes the specified data to the specified gs path.
func (f *FakeClient) WriteFileToGS(gsPath gs.Path, data []byte) error {
	expected, ok := f.ExpectedWrites[string(gsPath)]
	if !ok {
		f.T.Fatalf("unexpected write at %s", string(gsPath))
	}
	got, err := repo.ParseManifest(data)
	assert.NilError(f.T, err)
	if !reflect.DeepEqual(expected, got) {
		f.T.Fatalf("mismatch for write at %s: expected:\n%v\ngot:\n%v\n", string(gsPath), expected, got)
	}
	return nil
}

func (f *FakeClient) Download(gsPath gs.Path, localPath string) error {
	data, ok := f.ExpectedDownloads[string(gsPath)]
	if !ok {
		f.T.Fatalf("unexpected download of file %s", gsPath)
	}
	assert.NilError(f.T, ioutil.WriteFile(localPath, data, 0644))
	return nil
}

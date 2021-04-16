// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build linux

package repo

import (
	"io/ioutil"
	"testing"

	assert "infra/cros/internal/assert"
)

func TestGetSetDelAttr(t *testing.T) {
	tag := `<default foo="123" bar="456" baz="789" />`

	assert.StringsEqual(t, getAttr(tag, "foo"), "123")
	assert.StringsEqual(t, getAttr(tag, "bar"), "456")
	assert.StringsEqual(t, getAttr(tag, "baz"), "789")

	assert.StringsEqual(t, setAttr(tag, "foo", "000"), `<default foo="000" bar="456" baz="789" />`)
	assert.StringsEqual(t, setAttr(tag, "bar", "000"), `<default foo="123" bar="000" baz="789" />`)
	assert.StringsEqual(t, setAttr(tag, "baz", "000"), `<default foo="123" bar="456" baz="000" />`)

	assert.StringsEqual(t, delAttr(tag, "foo"), `<default bar="456" baz="789" />`)
	assert.StringsEqual(t, delAttr(tag, "bar"), `<default foo="123" baz="789" />`)
	assert.StringsEqual(t, delAttr(tag, "baz"), `<default foo="123" bar="456" />`)
}

func TestUpdateManifestElements(t *testing.T) {
	input, err := ioutil.ReadFile("test_data/update/pre.xml")
	assert.NilError(t, err)

	referenceManifest := &Manifest{
		Default: Default{
			RemoteName: "chromeos1",
			Revision:   "456",
		},
		Remotes: []Remote{
			{
				Name:  "chromium",
				Alias: "chromeos1",
				Fetch: "https://chromium.org/remote",
			},
		},
		Projects: []Project{
			{
				Name:       "baz",
				Path:       "baz/",
				RemoteName: "chromium1",
			},
			{
				Name:       "buz1",
				Path:       "buz/",
				RemoteName: "google",
			},
		},
	}

	got, err := UpdateManifestElements(referenceManifest, input)
	assert.NilError(t, err)

	expected, err := ioutil.ReadFile("test_data/update/post.xml")
	assert.NilError(t, err)
	if string(got) != string(expected) {
		t.Fatalf("mismatch on UpdateManifestElements(...)\ngot:%v\n\nexpected:%v\n\n", string(got), string(expected))
	}
}

func TestUpdateManifestElements_extraneous(t *testing.T) {
	input, err := ioutil.ReadFile("test_data/update/pre.xml")
	assert.NilError(t, err)

	referenceManifest := &Manifest{
		Remotes: []Remote{
			{
				Name: "extraneous",
			},
		},
	}

	_, err = UpdateManifestElements(referenceManifest, input)
	assert.ErrorContains(t, err, "contained remote(s)")

	referenceManifest = &Manifest{
		Projects: []Project{
			{
				Path: "extraneous/",
			},
		},
	}

	_, err = UpdateManifestElements(referenceManifest, input)
	assert.ErrorContains(t, err, "contained project(s)")

}

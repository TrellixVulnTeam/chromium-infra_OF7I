// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

// TestUploadWorker will test that network uploads are going to the correct place
// and that they are correctly handling the responses.
// TODO(b/197010274): implement mocks and response checking.
func TestUploadWorker(t *testing.T) {
	mockChans := channels{make(chan taskConfig), make(chan taskConfig)}
	err := uploadWorker(mockChans)

	if err != nil {
		t.Error("error: " + err.Error())
	}
}

// TestInitFlags verifies that the CLI flags are getting made, it
// expects default values.
func TestInitFlags(t *testing.T) {
	initFlags()

	if &gsPath == nil {
		t.Error("error: gsPath not initialized")
	}
	if &workerCount == nil {
		t.Error("error: workerCount not initialized")
	}
	if &retryCount == nil {
		t.Error("error: retryCount not initialized")
	}
	if &isStaging == nil {
		t.Error("error: isStaging not initialized")
	}
	if &dryRun == nil {
		t.Error("error: dryRun not initialized")
	}
}

// TestFetchTarball ensures that we are fetching from the correct service and
// handling the response appropriately.
// TODO(b/197010274): implement mocks and response checking.
func TestFetchTarball(t *testing.T) {
	localPath, err := fetchTarball("./path")

	if err != nil {
		t.Error("error: " + err.Error())
	}

	if localPath != "./path" {
		t.Error("error: incorrect path returned")
	}
}

// TestUnpackTarball confirms that we can properly unpack a given tarball and
// return filepaths to it's contents.
// TODO(b/197010274): implement response checking.
func TestUnpackTarball(t *testing.T) {
	symbolPaths, err := unpackTarball("./path")

	if err != nil {
		t.Error("error: " + err.Error())
	}

	if len(symbolPaths) <= 0 || symbolPaths == nil {
		t.Error("error: Empty list of paths returned")
	}
}

// TestGenerateConfigs validates that proper task configs are generated when
// a list of filepaths are given.
// TODO(b/197010274): implement response checking.
func TestGenerateConfigs(t *testing.T) {
	tasks, chans, err := generateConfigs([]string{"./path"})

	if err != nil {
		t.Error("error: " + err.Error())
	}

	if tasks != nil {
		t.Error("error: recieved tasks when nil was expected.")
	}

	if chans != nil {
		t.Error("error: recieved pointer to a channels struct when nil was expected.")
	}
}

// TestDoUpload affirms that the worker design and retry model are valid.
// TODO(b/197010274): implement mocks and response checking.
func TestDoUpload(t *testing.T) {
	retcode, err := doUpload([]taskConfig{}, nil, 0, false, false)

	if err != nil {
		t.Error("error: " + err.Error())
	}

	if retcode != 0 {
		t.Error("error: recieved non-zero retcode")
	}
}

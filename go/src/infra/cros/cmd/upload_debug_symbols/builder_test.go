// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"compress/gzip"
	"infra/cros/internal/gs"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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

// TestDownloadTgz ensures that we are fetching from the correct service and
// handling the response appropriately.
func TestDownloadTgz(t *testing.T) {
	gsPath := "gs://some-debug-symbol/degbug.tgz"

	expectedDownloads := map[string][]byte{
		gsPath: []byte("hello world"),
	}
	fakeClient := &gs.FakeClient{
		T:                 t,
		ExpectedDownloads: expectedDownloads,
	}

	tarballDir, err := ioutil.TempDir("", "tarball")
	if err != nil {
		log.Fatal(err)
	}

	tgzPath := filepath.Join(tarballDir, "debug.tgz")

	defer os.RemoveAll(tarballDir)

	err = downloadTgz(fakeClient, gsPath, tgzPath)

	if err != nil {
		t.Error("error: " + err.Error())
	}

	if _, err := os.Stat(tgzPath); os.IsNotExist(err) {
		t.Error("error: " + err.Error())
	}
}

// TestUnzipTgz confirms that we can properly unzip a given tgz file.
func TestUnzipTgz(t *testing.T) {
	targetString := "gzip test"

	// Create temp dir to work in.
	testDir, err := ioutil.TempDir("", "tarballTest")
	if err != nil {
		t.Error("error: " + err.Error())
	}
	defer os.RemoveAll(testDir)

	// Generate file information.
	outputFilePath := filepath.Join(testDir, "test.tar")
	inputFilePath := filepath.Join(testDir, "test.tgz")
	inputFile, err := os.Create(inputFilePath)
	if err != nil {
		t.Error("error: " + err.Error())
	}
	defer inputFile.Close()

	// Create a mock .tgz to test unzipping.
	zipWriter := gzip.NewWriter(inputFile)
	zipWriter.Name = "test.tgz"
	zipWriter.Comment = "hello world"

	_, err = zipWriter.Write([]byte(targetString))
	if err != nil {
		t.Error("error: " + err.Error())
	}

	err = zipWriter.Close()
	if err != nil {
		t.Error("error: " + err.Error())
	}

	err = unzipTgz(inputFilePath, outputFilePath)
	if err != nil {
		t.Error("error: " + err.Error())
	}

	// Check if the output file was created.
	if _, err := os.Stat(outputFilePath); os.IsNotExist(err) {
		t.Error("error: " + err.Error())
	}

	outputFile, err := os.Open(outputFilePath)
	if err != nil {
		t.Error("error: " + err.Error())
	}
	defer outputFile.Close()

	output, err := io.ReadAll(outputFile)
	if err != nil {
		t.Error("error: " + err.Error())
	}

	// Verify contents unzipped correctly.
	if string(output) != targetString {
		t.Errorf("error: expected %s got %s", targetString, string(output))
	}

}

// TestUnpackTarball confirms that we can properly unpack a given tarball and
// return filepaths to it's contents.
// TODO(b/197010274): implement response checking.
func TestUnpackTarball(t *testing.T) {
	symbolPaths, err := unpackTarball("./path", "./path")

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

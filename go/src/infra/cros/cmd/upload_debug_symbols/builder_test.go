// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
//go:build !windows
// +build !windows

package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"infra/cros/internal/gs"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockClient struct {
	expectedResponses map[string]string
}

func (c *mockClient) Do(req *http.Request) (*http.Response, error) {
	// if the request matches a string in the expected responses then generate a
	// response and return it. If it isn't apart of the map then return an error
	// and a nil response.
	if val, ok := c.expectedResponses[req.URL.String()]; ok {
		response := http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(val))}
		return &response, nil
	} else {
		return nil, fmt.Errorf("error: %s is not an expected url", req.URL.String())
	}
}

func initCrashConnectionMock(mockURL, mockKey string, responseMap map[string]string) crashConnectionInfo {
	return crashConnectionInfo{
		url:    mockURL,
		key:    mockKey,
		client: &mockClient{expectedResponses: responseMap},
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
		t.Error("error: " + err.Error())
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
// return filepaths to it's contents. Basic testing pulled from
// https://pkg.go.dev/archive/tar#pkg-overview.
func TestUnpackTarball(t *testing.T) {
	// Create working directory and tarball.
	testDir, err := ioutil.TempDir("", "tarballTest")
	if err != nil {
		t.Error("error: " + err.Error())
	}
	debugSymbolsDir, err := ioutil.TempDir(testDir, "symbols")
	if err != nil {
		t.Error("error: " + err.Error())
	}
	defer os.RemoveAll(testDir)

	// Generate file information.
	tarPath := filepath.Join(testDir, "test.tar")
	inputFile, err := os.Create(tarPath)
	if err != nil {
		t.Error("error: " + err.Error())
	}

	tarWriter := tar.NewWriter(inputFile)
	// Struct for file info
	type file struct {
		name, body string
		modeType   fs.FileMode
	}

	// Create an array holding some basic info to build headers. Contains regular
	// files and directories.
	files := []file{
		{"/test1.so.sym", "debug symbols", 0600},
		{"./test2.so.sym", "debug symbols", 0600},
		{"b/c", "", fs.ModeDir},
		{"../test3.so.sym", "debug symbols", 0600},
		{"a/b/c/d/", "", fs.ModeDir},
		{"./test4.so.sym", "debug symbols", 0600},
		{"a/shouldntadd.txt", "not a symbol file", 0600},
	}

	// List of files we expect to see return after the test call.
	expectedSymbolFiles := map[string]bool{
		debugSymbolsDir + "/test1.so.sym": false,
		debugSymbolsDir + "/test2.so.sym": false,
		debugSymbolsDir + "/test3.so.sym": false,
		debugSymbolsDir + "/test4.so.sym": false,
	}

	// Write the mock files to the tarball.
	for _, file := range files {
		hdr := &tar.Header{
			Name: file.name,
			Mode: int64(file.modeType),
			Size: int64(len(file.body)),
		}
		if err := tarWriter.WriteHeader(hdr); err != nil {
			t.Error("error: " + err.Error())
		}

		if file.modeType == 0600 {
			if _, err := tarWriter.Write([]byte(file.body)); err != nil {
				t.Error("error: " + err.Error())
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Error("error: " + err.Error())
	}

	// Call the Function
	symbolPaths, err := unpackTarball(tarPath, debugSymbolsDir)
	if err != nil {
		t.Error("error: " + err.Error())
	}
	if symbolPaths == nil || len(symbolPaths) <= 0 {
		t.Error("error: Empty list of paths returned")
	}
	// Verify that we received a list pointing to all the expected files and no
	// others.
	for _, path := range symbolPaths {
		if val, ok := expectedSymbolFiles[path]; ok {
			if val {
				t.Error("error: symbol file appeared multiple times in function return")
			}
			expectedSymbolFiles[path] = true
		} else {
			t.Errorf("error: unexpected symbol file returned %s", path)
		}
	}

}

// TestGenerateConfigs validates that proper task configs are generated when a
// list of filepaths are given.
func TestGenerateConfigs(t *testing.T) {
	// Init the mock files and verifying structures.
	expectedTasks := map[taskConfig]bool{}
	mockResponses := map[string]string{
		"test1.so.sym": "FOUND",
		"test2.so.sym": "FOUND",
		"test3.so.sym": "MISSING",
		"test4.so.sym": "MISSING",
		"test5.so.sym": "STATUS_UNSPECIFIED",
		"test6.so.sym": "STATUS_UNSPECIFIED",
	}

	// Make the expected request body
	responseBody := filterResponseBody{Pairs: []filterResponseStatusPair{}}

	// Test for all 3 cases found in http://google3/net/crash/symbolcollector/symbol_collector.proto?l=19
	for filename, symbolStatus := range mockResponses {
		symbol := filterSymbolFileInfo{filename, "F4F6FA6CCBDEF455039C8DE869C8A2F40"}
		responseBody.Pairs = append(responseBody.Pairs, filterResponseStatusPair{SymbolId: symbol, Status: symbolStatus})
	}
	mockResponseBody, err := json.Marshal(responseBody)
	if err != nil {
		t.Error("error: " + err.Error())
	}

	// Mock the symbol files locally.
	testDir, err := ioutil.TempDir("", "configGenTest")
	if err != nil {
		t.Error("error: " + err.Error())
	}
	defer os.RemoveAll(testDir)

	mockPaths := []string{}
	for filename, symbolStatus := range mockResponses {
		mockPath := filepath.Join(testDir, filename)
		err = ioutil.WriteFile(mockPath, []byte("MODULE Linux arm F4F6FA6CCBDEF455039C8DE869C8A2F40 blkid"), 0644)
		if err != nil {
			t.Error("error: " + err.Error())
		}
		task := taskConfig{mockPath, filename, "F4F6FA6CCBDEF455039C8DE869C8A2F40", false, false}

		if symbolStatus != "FOUND" {
			expectedTasks[task] = false
		}
		mockPaths = append(mockPaths, mockPath)
	}

	// Init global variables and
	mockCrash := initCrashConnectionMock("google.com", "1234", map[string]string{"google.com/symbols:checkStatuses?key=1234": string(mockResponseBody)})

	tasks, err := generateConfigs(mockPaths, 0, false, mockCrash)
	if err != nil {
		t.Error("error: " + err.Error())
	}
	// Check that returns aren't nil.
	if tasks == nil {
		t.Error("error: recieved tasks when nil was expected")
	}

	// Verify that we received a list pointing to all the expected files and no
	// others.
	for _, task := range tasks {
		if val, ok := expectedTasks[task]; ok {
			if val {
				t.Error("error: task appeared multiple times in function return")
			}
			expectedTasks[task] = true
		} else {
			t.Errorf("error: unexpected task returned %+v", task)
		}
	}

	for task, value := range expectedTasks {
		if value == false {
			t.Errorf("error: task for file %s never seen", task.debugFile)
		}
	}
}

// TestUploadSymbols affirms that the worker design and retry model are valid.
func TestUploadSymbols(t *testing.T) {
	// Create tasks and expected returns.
	tasks := []taskConfig{
		{"", "test1.so.sym", "", false, false},
		{"", "test2.so.sym", "", false, false},
		{"", "test3.so.sym", "", false, false},
		{"", "test4.so.sym", "", false, false},
	}

	// Mock the symbol files locally.
	testDir, err := ioutil.TempDir("", "uploadSymbolsTest")
	if err != nil {
		t.Error("error: " + err.Error())
	}
	defer os.RemoveAll(testDir)

	// Write mock files locally.
	for index, task := range tasks {
		mockPath := filepath.Join(testDir, task.debugFile)
		err = ioutil.WriteFile(mockPath, []byte("MODULE Linux arm F4F6FA6CCBDEF455039C8DE869C8A2F40 blkid"), 0644)
		if err != nil {
			t.Error("error: " + err.Error())
		}
		tasks[index].symbolPath = mockPath
	}

	// unique URL and key
	mockURLKeyPair := crashUploadInformation{UploadUrl: "crashupload.com", UploadKey: "abc"}
	mockCompleteResponse := crashSubmitResposne{Result: "OK"}
	mockURLKeyPairJSON, err := json.Marshal(mockURLKeyPair)
	if err != nil {
		t.Error("error: could not marshal response")
	}
	mockCompleteResponseJSON, err := json.Marshal(mockCompleteResponse)
	if err != nil {
		t.Error("error: could not marshal response")
	}

	expectedResponses := map[string]string{
		"google.com/uploads:create?key=1234":       string(mockURLKeyPairJSON),
		"crashupload.com":                          "",
		"google.com/uploads/abc:complete?key=1234": string(mockCompleteResponseJSON),
	}

	crashMock := initCrashConnectionMock("google.com", "1234", expectedResponses)
	retcode, err := uploadSymbols(tasks, 64, 2, false, crashMock)

	if err != nil {
		t.Error("error: " + err.Error())
	}

	if retcode != 0 {
		t.Errorf("error: recieved non-zero retcode %d", retcode)
	}
}

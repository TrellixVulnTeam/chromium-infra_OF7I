// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package atutil

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// HostInfo is a struct providing a mapping
// to an autotest host_info file.
type HostInfo struct {
	Attributes        map[string]string `json:"attributes"`
	Labels            []string          `json:"labels"`
	SerializerVersion int               `json:"serializer_version,omitempty"`
	StableVersions    map[string]string `json:"stable_versions"`
}

func HostInfoFilePath(rootDir, host string) string {
	f := fmt.Sprintf("%s.store", host)
	return filepath.Join(rootDir, hostInfoSubDir, f)
}

func AddLabelToHostInfoFile(filePath, label string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Wrap(err, "add label to host info file")
	}
	hostInfo := HostInfo{}
	if err := json.Unmarshal([]byte(data), &hostInfo); err != nil {
		return errors.Wrap(err, "add label to host info file")
	}

	if hostInfo.StableVersions == nil {
		hostInfo.StableVersions = make(map[string]string)
	}
	hostInfo.Labels = append(hostInfo.Labels, label)
	data, err = json.MarshalIndent(hostInfo, "", "    ")
	if err != nil {
		return errors.Wrap(err, "add label to host info file")
	}

	if err := ioutil.WriteFile(filePath, data, 0); err != nil {
		return errors.Wrap(err, "add label to host info file")
	}
	return nil
}

// LinkHostInfoFile prepares the host info store by linking the host
// file in the dstResultDir to the srcResultsDir.
// It is intended as an alternative to prepareHostInfo, which contains
// autoserv-specific logic and does not work for TLS.
func LinkHostInfoFile(srcResultsDir, dstResultDir, host string) error {
	dstdir := filepath.Join(dstResultDir, hostInfoSubDir)
	if err := os.MkdirAll(dstdir, 0777); err != nil {
		return err
	}
	f := fmt.Sprintf("%s.store", host)
	src := HostInfoFilePath(srcResultsDir, f)
	dst := HostInfoFilePath(dstResultDir, f)
	if err := linkFile(src, dst); err != nil {
		return err
	}
	return nil
}

// prepareHostInfo prepares the host info store for the autoserv job
// using the master host info store in the results directory.
func prepareHostInfo(resultsDir string, j AutoservJob) error {
	ja := j.AutoservArgs()
	dstdir := filepath.Join(ja.ResultsDir, hostInfoSubDir)
	if err := os.MkdirAll(dstdir, 0777); err != nil {
		return err
	}
	for _, h := range ja.Hosts {
		f := fmt.Sprintf("%s.store", h)
		src := filepath.Join(resultsDir, hostInfoSubDir, f)
		dst := filepath.Join(dstdir, f)
		if err := linkFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// retrieveHostInfo retrieves the host info store for the autoserv job
// back to the master host info store in the results directory.
func retrieveHostInfo(resultsDir string, j AutoservJob) error {
	ja := j.AutoservArgs()
	for _, h := range ja.Hosts {
		f := fmt.Sprintf("%s.store", h)
		src := filepath.Join(ja.ResultsDir, hostInfoSubDir, f)
		dst := filepath.Join(resultsDir, hostInfoSubDir, f)
		if err := linkFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

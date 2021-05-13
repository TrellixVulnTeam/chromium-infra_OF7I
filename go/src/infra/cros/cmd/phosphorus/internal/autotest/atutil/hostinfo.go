// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package atutil

import (
	"context"
	"encoding/json"
	"fmt"
	"infra/cros/cmd/phosphorus/internal/gs"
	"infra/cros/cmd/phosphorus/internal/tls"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const (
	crosVersionLabel = "cros-version"
	imageCacheLabel  = "job_repo_url"
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

func AddProvisionDetailsToHostInfoFile(ctx context.Context, bt *tls.BackgroundTLS, infoFileDir, dutName, crosVersion string) error {
	errWrap := fmt.Sprintf("add %s and %s labels to host info file", imageCacheLabel, crosVersionLabel)
	infoFilePath := HostInfoFilePath(infoFileDir, dutName)
	hostInfo, err := readHostInfoFile(infoFilePath)
	if err != nil {
		return errors.Wrap(err, errWrap)
	}

	if hostInfo.StableVersions == nil {
		hostInfo.StableVersions = make(map[string]string)
	}
	if err := hostInfo.setImageCacheURL(ctx, bt, dutName, crosVersion); err != nil {
		return errors.Wrap(err, errWrap)
	}
	hostInfo.setCrosVersion(crosVersion)

	if err := writeHostInfoFile(infoFilePath, hostInfo); err != nil {
		return errors.Wrap(err, errWrap)
	}
	return nil
}

func (hi *HostInfo) setImageCacheURL(ctx context.Context, bt *tls.BackgroundTLS, dutName, crosVersion string) error {
	gsImagePath := fmt.Sprintf("%s/%s", gs.ImageArchivePrefix, crosVersion)
	repoURL, err := bt.CacheForDut(ctx, gsImagePath, dutName)
	if err != nil {
		return err
	}
	hi.Attributes[imageCacheLabel] = repoURL
	return nil
}

func (hi *HostInfo) setCrosVersion(crosVersion string) {
	// Clear existing cros-version label.
	for i, label := range hi.Labels {
		if strings.HasPrefix(label, crosVersionLabel+":") {
			hi.Labels = append(hi.Labels[:i], hi.Labels[i+1:]...)
			break
		}
	}
	newLabel := fmt.Sprintf("%s:%s", crosVersionLabel, crosVersion)
	hi.Labels = append(hi.Labels, newLabel)
}

func readHostInfoFile(infoFilePath string) (*HostInfo, error) {
	data, err := ioutil.ReadFile(infoFilePath)
	if err != nil {
		return nil, err
	}
	hostInfo := &HostInfo{}
	if err := json.Unmarshal([]byte(data), hostInfo); err != nil {
		return nil, err
	}
	return hostInfo, nil
}

func writeHostInfoFile(infoFilePath string, hostInfo *HostInfo) error {
	updatedData, err := json.MarshalIndent(hostInfo, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(infoFilePath, updatedData, 0)

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
	src := HostInfoFilePath(srcResultsDir, host)
	dst := HostInfoFilePath(dstResultDir, host)
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

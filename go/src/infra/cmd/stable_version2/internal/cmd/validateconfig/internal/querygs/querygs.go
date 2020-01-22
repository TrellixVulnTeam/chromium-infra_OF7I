// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package querygs

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/common/gcloud/gs"

	labPlatform "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	gslib "infra/cmd/stable_version2/internal/gs"
)

// BoardModel is a combined build target and model. It is used for models that aren't present
// in any metadata.json file that we read.
type BoardModel struct {
	BuildTarget string `json:"build_target"`
	Model       string `json:"model"`
}

// VersionMismatch is a discrepancy between the version in config file and the version in the
// associated stable_versions.cfg file.
type VersionMismatch struct {
	BuildTarget string `json:"build_target"`
	Model       string `json:"model"`
	Wanted      string `json:"wanted"`
	Got         string `json:"got"`
}

// ValidationResult is a summary of the result of validating a stable version config file.
type ValidationResult struct {
	// MissingBoards are the boards that don't have a metadata file in Google Storage.
	MissingBoards []string `json:"missing_boards"`
	// FailedToLookup are board/model pairs that aren't present in the descriptions fetched from Google Storage.
	FailedToLookup []*BoardModel `json:"failed_to_lookup"`
	// InvalidVersions is the list of entries where the version in the config file does not match Google Storage.
	InvalidVersions []*VersionMismatch `json:"invalid_versions"`
}

// RemoveWhitelistedDUTs removes DUTs that are whitelisted from the validation error summary.
// examples include labstations
func (r *ValidationResult) RemoveWhitelistedDUTs() {
	var newMissingBoards []string
	var newFailedToLookup []*BoardModel
	if len(r.MissingBoards) != 0 {
		for _, item := range r.MissingBoards {
			if !missingBoardWhitelist[item] {
				newMissingBoards = append(newMissingBoards, item)
			}
		}
	}
	if len(r.FailedToLookup) != 0 {
		for _, item := range r.FailedToLookup {
			if !failedToLookupWhiteList[fmt.Sprintf("%s;%s", item.BuildTarget, item.Model)] {
				newFailedToLookup = append(newFailedToLookup, item)
			}
		}
	}
	r.MissingBoards = newMissingBoards
	r.FailedToLookup = newFailedToLookup
}

// AnomalyCount counts the total number of issues found in the results summary.
func (r *ValidationResult) AnomalyCount() int {
	return len(r.MissingBoards) + len(r.FailedToLookup) + len(r.InvalidVersions)
}

type downloader func(gsPath gs.Path) ([]byte, error)

// Reader reads metadata.json files from google storage and caches the result.
type Reader struct {
	dld   downloader
	cache map[string]map[string]string
}

// Init creates a new Google Storage Client.
// TODO(gregorynisbet): make it possible to initialize a test gsClient as well
func (r *Reader) Init(ctx context.Context, t http.RoundTripper, unmarshaler jsonpb.Unmarshaler, tempPrefix string) error {
	var gsc gslib.Client
	if err := gsc.Init(ctx, t, unmarshaler); err != nil {
		return fmt.Errorf("Reader::Init: %s", err)
	}
	r.dld = func(remotePath gs.Path) ([]byte, error) {
		dir, err := ioutil.TempDir("", tempPrefix)
		if err != nil {
			return nil, fmt.Errorf("download adapter: making temporary directory: %s", err)
		}
		defer os.RemoveAll(dir)
		localPath := filepath.Join(dir, "metadata.json")
		if err := gsc.Download(remotePath, localPath); err != nil {
			return nil, fmt.Errorf("download adapter: fetching file: %s", err)
		}
		contents, err := ioutil.ReadFile(localPath)
		if err != nil {
			return nil, fmt.Errorf("download adapter: reading local file: %s", err)
		}
		return contents, nil
	}
	return nil
}

// ValidateConfig takes a stable version protobuf and attempts to validate every entry.
func (r *Reader) ValidateConfig(sv *labPlatform.StableVersions) (*ValidationResult, error) {
	var cfgCrosVersions = make(map[string]string, len(sv.GetCros()))
	var out ValidationResult
	if sv == nil {
		return nil, fmt.Errorf("Reader::ValidateConfig: config file cannot be nil")
	}
	// use the CrOS keys in the sv file to seed the reader.
	for _, item := range sv.GetCros() {
		bt := item.GetKey().GetBuildTarget().GetName()
		version := item.GetVersion()
		if _, err := r.getAllModelsForBuildTarget(bt, version); err != nil {
			out.MissingBoards = append(out.MissingBoards, bt)
			continue
		}
		cfgCrosVersions[bt] = version
	}
	for _, item := range sv.GetFirmware() {
		bt := item.GetKey().GetBuildTarget().GetName()
		model := item.GetKey().GetModelId().GetValue()
		cfgVersion := item.GetVersion()
		realVersion, err := r.getFirmwareVersion(bt, model, cfgCrosVersions[bt])
		if err != nil {
			out.FailedToLookup = append(out.FailedToLookup, &BoardModel{bt, model})
			continue
		}
		if cfgVersion != realVersion {
			out.InvalidVersions = append(out.InvalidVersions, &VersionMismatch{bt, model, realVersion, cfgVersion})
			continue
		}
	}
	return &out, nil
}

// allModels returns a mapping from model names to fimrware versions given a buildTaret and CrOS version.
func (r *Reader) getAllModelsForBuildTarget(buildTarget string, version string) (map[string]string, error) {
	if err := r.maybeDownloadFile(buildTarget, version); err != nil {
		return nil, fmt.Errorf("AllModels: %s", err)
	}
	if _, ok := r.cache[buildTarget]; !ok {
		return nil, fmt.Errorf("AllModels: buildTarget MUST be present (%s)", buildTarget)
	}
	return r.cache[buildTarget], nil
}

// getFirmwareVersion returns the firmware version for a specific model given the buildTarget and CrOS version.
func (r *Reader) getFirmwareVersion(buildTarget string, model string, version string) (string, error) {
	if err := r.maybeDownloadFile(buildTarget, version); err != nil {
		return "", fmt.Errorf("FirmwareVersion: %s", err)
	}
	if _, ok := r.cache[buildTarget]; !ok {
		// If control makes it here, then maybeDownloadFile should have returned
		// a non-nil error.
		return "", fmt.Errorf("getFirmwareVersion: buildTarget MUST be present (%s)", buildTarget)
	}
	version, ok := r.cache[buildTarget][model]
	if !ok {
		return "", fmt.Errorf("no info for model (%s)", model)
	}
	return version, nil
}

// maybeDownloadFile fetches a metadata.json corresponding to a buildTarget and version if it doesn't already exist in the cache.
func (r *Reader) maybeDownloadFile(buildTarget string, crosVersion string) error {
	if r.cache == nil {
		r.cache = make(map[string]map[string]string)
	}
	if _, ok := r.cache[buildTarget]; ok {
		return nil
	}
	// TODO(gregorynisbet): extend gslib with function to get path
	remotePath := gs.Path(fmt.Sprintf("gs://chromeos-image-archive/%s-release/%s/metadata.json", buildTarget, crosVersion))
	contents, err := (r.dld)(remotePath)
	if err != nil {
		return fmt.Errorf("Reader::maybeDownloadFile: fetching file: %s", err)
	}
	fws, err := gslib.ParseMetadata(contents)
	if err != nil {
		return fmt.Errorf("Reader::maybeDownloadFile: parsing metadata.json: %s", err)
	}
	// TODO(gregorynisbet): Consider throwing an error or panicking if we encounter
	// a duplicate when populating the cache.
	for _, fw := range fws {
		buildTarget := fw.GetKey().GetBuildTarget().GetName()
		model := fw.GetKey().GetModelId().GetValue()
		version := fw.GetVersion()
		if _, ok := r.cache[buildTarget]; !ok {
			r.cache[buildTarget] = make(map[string]string)
		}
		r.cache[buildTarget][model] = version
	}
	return nil
}

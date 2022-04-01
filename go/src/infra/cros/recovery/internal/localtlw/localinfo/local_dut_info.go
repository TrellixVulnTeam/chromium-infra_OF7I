// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package localinfo

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// Store holds a DUT's localDUTState and dut host name.
type store struct {
	LocalDUTState *localDUTState
	dutHostname   string
}

// ReadProvisionInfo takes in the dut name and find the file that existed on the drone about the dut's
// provision information (cros version and job repo url) and then returns a tlw.DUTProvisionedInfo that
// contains these two fields.
func ReadProvisionInfo(ctx context.Context, dutHostname string) (*tlw.DUTProvisionedInfo, error) {
	pi := &tlw.DUTProvisionedInfo{}
	s, err := readStore(dutHostname)
	if err != nil {
		return pi, errors.Annotate(err, "read provision info").Err()
	}
	crosVersion, ok := s.LocalDUTState.ProvisionableLabels[CrosVersionKey]
	if !ok {
		log.Debugf(ctx, "local dut info file does not have provisional information of cros-version.")
	}
	pi.CrosVersion = crosVersion
	jobRepoURL, ok := s.LocalDUTState.ProvisionableAttributes[JobRepoURLKey]
	if !ok {
		log.Debugf(ctx, "local dut info file does not have provisional information of job_repo_url.")
	}
	pi.JobRepoURL = jobRepoURL
	return pi, nil
}

// UpdateProvisionInfo updates the provion file by updating the store.LocalDUTState's two attributes
// ProvisionableLabels as well as ProvisionableAttributes to the dut's corresponding provision info values.
func UpdateProvisionInfo(ctx context.Context, dut *tlw.Dut) error {
	s, err := readStore(dut.Name)
	if err != nil {
		return errors.Annotate(err, "update provision info").Err()
	}
	log.Debugf(ctx, "Update provision info to %s", dut.ProvisionedInfo)
	s.LocalDUTState.ProvisionableAttributes = make(provisionableAttributes)
	s.LocalDUTState.ProvisionableLabels = make(provisionableLabels)
	if dut.ProvisionedInfo.CrosVersion != "" {
		s.LocalDUTState.ProvisionableLabels[CrosVersionKey] = dut.ProvisionedInfo.CrosVersion
	}
	if dut.ProvisionedInfo.JobRepoURL != "" {
		s.LocalDUTState.ProvisionableAttributes[JobRepoURLKey] = dut.ProvisionedInfo.JobRepoURL
	}
	return errors.Annotate(s.writeStore(), "update provision info").Err()
}

// Update writes the localDUTState back to disk.
func (s *store) writeStore() error {
	if s == nil {
		return nil
	}
	data, err := s.LocalDUTState.marshal()
	if err != nil {
		return errors.Annotate(err, "write store").Err()
	}
	filePathForDut, err := localDUTInfoFilePath(s.dutHostname)
	if err != nil {
		return errors.Annotate(err, "write store").Err()
	}
	// Create directories if it is not exist.
	// The directory need to be created with high permission to allowed to create files inside.
	newpath := filepath.Dir(filePathForDut)
	if err := os.MkdirAll(newpath, 0755); err != nil {
		return errors.Annotate(err, "write store").Err()
	}
	// Write DUT state into a local file named by DUT's hostname.
	if err := ioutil.WriteFile(filePathForDut, data, 0666); err != nil {
		return errors.Annotate(err, "write store").Err()
	}
	return nil
}

// readStore loads the store for the DUT.
func readStore(dutHostname string) (*store, error) {
	s := store{LocalDUTState: &localDUTState{}, dutHostname: dutHostname}
	if err := s.retrieveLocalState(); err != nil {
		return nil, errors.Annotate(err, "read store").Err()
	}
	return &s, nil
}

// localDUTInfoFilePath returns the path of the local dut info file for the given DUT name.
func localDUTInfoFilePath(dutName string) (string, error) {
	return filepath.Join(localDUTInfoDirPath(), fmt.Sprintf("%s.json", dutName)), nil
}

// localDUTInfoDirPath returns the path to the cache directory for the given DUT for PARIS.
func localDUTInfoDirPath() string {
	return filepath.Join(os.Getenv("AUTOTEST_DIR"), "swarming_state")
}

// retrieveLocalState read DUT state data from local file and unmarshal the data.
// If the read fails due to target file not exists, the method will initialize
// an empty localDUTState.
func (s *store) retrieveLocalState() error {
	filePathForDut, err := localDUTInfoFilePath(s.dutHostname)
	if err != nil {
		return errors.Annotate(err, "retrive local state").Err()
	}
	data, err := ioutil.ReadFile(filePathForDut)
	if os.IsNotExist(err) {
		// If the file not exists we Marshal and Unmarshal a nil value here
		// to take advantage of Unmarshal initializing nested maps.
		data, err = s.LocalDUTState.marshal()
		if err != nil {
			return errors.Annotate(err, "retrive local state").Err()
		}
	} else if err != nil {
		return errors.Annotate(err, "retrive local state").Err()
	}
	if err := s.LocalDUTState.unmarshal(data); err != nil {
		return errors.Annotate(err, "retrive local state").Err()
	}
	return nil
}

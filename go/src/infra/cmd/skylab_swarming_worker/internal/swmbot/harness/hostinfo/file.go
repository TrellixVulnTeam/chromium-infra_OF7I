// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hostinfo

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"go.chromium.org/luci/common/errors"

	"infra/cmd/skylab_swarming_worker/internal/autotest/hostinfo"
)

// File represents a hostinfo file exposed for Autotest to use.
type File struct {
	hostInfo *hostinfo.HostInfo
	path     string
}

// hostInfoSubDir is the filename of the directory for storing host info.
const hostInfoSubDir = "host_info_store"

// Expose exposes the HostInfo as a file for Autotest to use.
func Expose(hi *hostinfo.HostInfo, resultsDir string, dutName string) (*File, error) {
	log.Printf("hostinfo::Expose: hi (%#v) resultsDir (%s) dutName (%s)", hi, resultsDir, dutName)
	blob, err := hostinfo.Marshal(hi)
	if err != nil {
		return nil, errors.Annotate(err, "expose hostinfo").Err()
	}
	storeDir := filepath.Join(resultsDir, hostInfoSubDir)
	if err := os.Mkdir(storeDir, 0755); err != nil {
		return nil, errors.Annotate(err, "expose hostinfo").Err()
	}
	storeFile := filepath.Join(storeDir, fmt.Sprintf("%s.store", dutName))
	if err := ioutil.WriteFile(storeFile, blob, 0644); err != nil {
		return nil, errors.Annotate(err, "expose hostinfo").Err()
	}
	// Write the secondary host info store file.
	// TODO(gregorynisbet): Remove secondary host info file or remove original host info file.
	// Failure to create a secondary host info store file
	// is not a serious enough error to make the whole Expose action
	// unsuccessful.
	content, err := hostinfo.MarshalIndent(hi)
	if err != nil {
		log.Printf("Expose: failed to marshalIndent hostinfo file (%#v)", err)
		content = []byte{}
	}
	hostInfoStore2 := filepath.Join(storeDir, fmt.Sprintf("%s.host_info_store2", dutName))
	// NOTE(gregorynisbet): we always want to create this file, even if it has length zero due to a previous error
	if err := ioutil.WriteFile(hostInfoStore2, content, 0o644); err != nil {
		log.Printf("Expose: failed to write file contents to path (%s)", hostInfoStore2)
	}
	return &File{
		hostInfo: hi,
		path:     storeFile,
	}, nil
}

// Close marks that Autotest is finished using the exposed HostInfo
// file and loads any changes back into the original HostInfo.
// Subsequent calls do nothing.  This is safe to call on a nil pointer.
func (f *File) Close() error {
	if f == nil {
		return nil
	}
	if f.path == "" {
		return nil
	}
	blob, err := ioutil.ReadFile(f.path)
	if err != nil {
		return errors.Annotate(err, "close exposed hostinfo").Err()
	}
	hi, err := hostinfo.Unmarshal(blob)
	if err != nil {
		return errors.Annotate(err, "close exposed hostinfo").Err()
	}
	f.path = ""
	log.Printf("File::Close: hostinfo before fixup (%#v)", f.hostInfo)
	*f.hostInfo = *hi
	log.Printf("File::Close: hostinfo after fixup (%#v)", f.hostInfo)
	return nil
}

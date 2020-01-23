// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/phosphorus"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/phosphorus/internal/autotest"
	"infra/cros/cmd/phosphorus/internal/autotest/atutil"
)

type commonRun struct {
	subcommands.CommandRunBase

	authFlags  authcli.Flags
	inputPath  string
	outputPath string
}

func (c *commonRun) validateArgs() error {
	if c.inputPath == "" {
		return fmt.Errorf("-input_json not specified")
	}

	return nil
}

// readJSONPb reads a JSON string from inFile and unpacks it as a proto.
// Unexpected fields are ignored.
func readJSONPb(inFile string, payload proto.Message) error {
	r, err := os.Open(inFile)
	if err != nil {
		return errors.Annotate(err, "read JSON pb").Err()
	}
	defer r.Close()

	unmarshaler := jsonpb.Unmarshaler{AllowUnknownFields: true}
	if err := unmarshaler.Unmarshal(r, payload); err != nil {
		return errors.Annotate(err, "read JSON pb").Err()
	}
	return nil
}

// writeJSONPb writes a JSON encoded proto to outFile.
func writeJSONPb(outFile string, payload proto.Message) error {
	dir := filepath.Dir(outFile)
	// Create the directory if it doesn't exist.
	if err := os.MkdirAll(dir, 0777); err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}

	w, err := os.Create(outFile)
	if err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}
	defer w.Close()

	marshaler := jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, payload); err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}
	return nil
}

// getCommonMissingArgs returns the list of missing required config
// arguments.
func getCommonMissingArgs(c *phosphorus.Config) []string {
	// TODO(1039484): Split this into subcommand-specific functions
	var missingArgs []string

	if c.GetBot().GetAutotestDir() == "" {
		missingArgs = append(missingArgs, "autotest dir")
	}

	if c.GetTask().GetResultsDir() == "" {
		missingArgs = append(missingArgs, "results dir")
	}

	return missingArgs
}

// getMainJob constructs a atutil.MainJob from a Config proto.
func getMainJob(c *phosphorus.Config) *atutil.MainJob {
	return &atutil.MainJob{
		AutotestConfig: autotest.Config{
			AutotestDir: c.GetBot().GetAutotestDir(),
		},
		ResultsDir:       c.GetTask().GetResultsDir(),
		UseLocalHostInfo: true,
	}

}

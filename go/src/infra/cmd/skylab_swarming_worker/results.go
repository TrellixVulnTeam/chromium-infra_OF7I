// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/common/errors"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"infra/cmd/skylab_swarming_worker/internal/annotations"
)

// TODO(zamorzaev): move this into the isolate client.
const resultsFileName = "results.json"

// writeResultsFile writes the results to "results.json" inside the given dir.
func writeResultsFile(outdir string, r *skylab_test_runner.Result, logdogOutput io.Writer) error {
	annotations.SeedStep(logdogOutput, "Write results.json")
	annotations.StepCursor(logdogOutput, "Write results.json")
	annotations.StepStarted(logdogOutput)
	defer annotations.StepClosed(logdogOutput)

	f := filepath.Join(outdir, resultsFileName)
	fmt.Fprintf(logdogOutput, "Writing results to %s", f)

	w, err := os.Create(f)
	if err != nil {
		return errors.Annotate(err, "write results.json").Err()
	}
	defer w.Close()

	marshaler := jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, r); err != nil {
		annotations.StepException(logdogOutput)
		fmt.Fprint(logdogOutput, err.Error())
		return errors.Annotate(err, "write results.json").Err()
	}
	return nil
}

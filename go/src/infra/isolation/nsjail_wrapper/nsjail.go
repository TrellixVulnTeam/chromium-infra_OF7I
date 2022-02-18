// Copyright 2022 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	_ "embed"

	"go.chromium.org/luci/logdog/client/butlerlib/bootstrap"
	"go.chromium.org/luci/logdog/client/butlerlib/streamclient"
)

// This compiles `task.cfg` into the final program as a []byte
//go:embed task.cfg
var Config []byte

const configName = "task.cfg"

var execCommand = exec.Command

// This function is replaced in the test.
var setupNsJailLog = func(ctx context.Context) (*os.File, error) {
	logdogBootstrap, err := bootstrap.Get()
	if err != nil {
		return nil, fmt.Errorf("bootstrapping failed: %s", err.Error())
	}
	nsjailLog, err := logdogBootstrap.Client.NewStream(ctx, "nsjail", streamclient.ForProcess())
	if err != nil {
		return nil, fmt.Errorf("could not open logstream for nsjail: %s", err.Error())
	}
	return nsjailLog.(*os.File), nil
}

// RunInNsjail takes in the command to be run as a []string
// where command[0] is the executable to be run, and
// command[1...] are the arguments to pass to the executable
func RunInNsjail(ctx context.Context, command []string) error {

	if err := os.WriteFile(configName, Config, 0644); err != nil {
		return fmt.Errorf("problem writing config: %s", err.Error())
	}

	nsjailFile, err := setupNsJailLog(ctx)
	if err != nil {
		return fmt.Errorf("problem retrieving nsjail log file: %s", err.Error())
	}

	dir, err := os.Getwd()
	if err != nil {
		return errors.New("could not obtain working directory")
	}

	logFd := strconv.FormatUint(uint64(nsjailFile.Fd())+3, 10)
	nsjailPath := filepath.Join(dir, "nsjail")
	cmdConfig := append([]string{"--config", configName, "--log_fd", logFd, "--seccomp_log"}, command...)
	nsjailCmd := execCommand(nsjailPath, cmdConfig...)
	nsjailCmd.ExtraFiles = []*os.File{nsjailFile}

	// Configure stdin/stdout/stderr
	nsjailCmd.Stdin = os.Stdin
	nsjailCmd.Stdout = os.Stdout
	nsjailCmd.Stderr = os.Stderr

	if err := nsjailCmd.Start(); err != nil {
		return fmt.Errorf("problem running nsjail: %s", err.Error())
	}

	// Not closing stdout/stderr because we expect the wrapper's
	// output to mingle with subprocess output, and when the
	// wrapper process exits the system will close them anyways.
	if err := os.Stdin.Close(); err != nil {
		return fmt.Errorf("closing stdin: %s", err.Error())
	}

	if err := nsjailCmd.Wait(); err != nil {
		return fmt.Errorf("nsjail completed with error: %s", err.Error())
	}

	if err := nsjailFile.Close(); err != nil {
		return fmt.Errorf("could not close log: %s", err.Error())
	}

	return nil
}

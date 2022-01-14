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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var execCommand = exec.Command

// RunInNsjail takes in the command to be run as a []string
// where command[0] is the executable to be run, and
// command[1...] are the arguments to pass to the executable
// TODO(yulanlin) use a config file
// TODO(yulanlin) add seccomp-bpf settings
func RunInNsjail(command []string) error {

	dir, err := os.Getwd()
	if err != nil {
		return errors.New("could not obtain working directory")
	}
	nsjailPath := filepath.Join(dir, "nsjail")

	// TODO(yulanlin): embed the config and pass that to nsjail
	config := "--user 99999 --group 99999"
	cmdConfig := append([]string{config, "--"}, command...)

	nsjailCmd := execCommand(nsjailPath, cmdConfig...)

	// TODO(yulanlin): handle log output properly
	_, err = nsjailCmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("problem running nsjail: %s", err.Error())
	}

	return nil
}

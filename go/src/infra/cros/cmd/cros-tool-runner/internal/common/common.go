// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/golang/protobuf/jsonpb"

	"go.chromium.org/luci/common/errors"
)

// RunWithTimeout runs command with timeout limit.
func RunWithTimeout(ctx context.Context, cmd *exec.Cmd, timeout time.Duration, block bool) (stdout string, stderr string, err error) {
	// TODO(b/219094608): Implement timeout usage.
	var se, so bytes.Buffer
	cmd.Stderr = &se
	cmd.Stdout = &so
	defer func() {
		stdout = so.String()
		stderr = se.String()
	}()

	log.Printf("Run cmd: %q", cmd)
	if block {
		err = cmd.Run()
	} else {
		err = cmd.Start()
	}

	if err != nil {
		log.Printf("error found with cmd: %q: %s", cmd, err)
	}
	return
}

// PrintToLog prints cmd, stdout, stderr to log
func PrintToLog(cmd string, stdout string, stderr string) {
	if cmd != "" {
		log.Printf("%q command execution.", cmd)
	}
	if stdout != "" {
		log.Printf("stdout: %s", stdout)
	}
	if stderr != "" {
		log.Printf("stderr: %s", stderr)
	}
}

// AddContentsToLog adds contents of the file of fileName to log
func AddContentsToLog(fileName string, rootDir string, msgToAdd string) error {
	filePath, err := FindFile(fileName, rootDir)
	if err != nil {
		log.Printf("%s finding file '%s' at '%s' failed:%s", msgToAdd, fileName, filePath, err)
		return err
	}
	fileContents, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("%s reading file '%s' at '%s' failed:%s", msgToAdd, fileName, filePath, err)
		return err
	}
	log.Printf("%s file '%s' info at '%s':\n\n%s\n", msgToAdd, fileName, filePath, string(fileContents))
	return nil
}

// FindFile finds file path in rootDir of fileName
func FindFile(fileName string, rootDir string) (string, error) {
	filePath := ""
	filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == fileName {
			filePath = path
		}
		return nil
	})

	if filePath != "" {
		return filePath, nil
	}

	return "", errors.Reason(fmt.Sprintf("file '%s' not found!", fileName)).Err()
}

// JsonPbUnMarshaler returns the unmarshaler which should be used across CTR.
func JsonPbUnmarshaler() jsonpb.Unmarshaler {
	return jsonpb.Unmarshaler{AllowUnknownFields: true}
}

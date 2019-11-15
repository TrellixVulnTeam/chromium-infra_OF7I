// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package sideeffects implements the validation of side effects
// configuration.
package sideeffects

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/common/errors"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/side_effects"
	"infra/libs/skylab/cloudkms"
)

// ValidateConfig checks the presence of all required fields in
// side_effects.Config and the existence of all required files.
func ValidateConfig(c *side_effects.Config) error {
	ma := getMissingArgs(c)

	if len(ma) > 0 {
		return fmt.Errorf("Error validating side_effects.Config: no %s provided",
			strings.Join(ma, ", "))
	}

	mf := getMissingFiles(c)

	if len(mf) > 0 {
		return fmt.Errorf("Error getting the following file(s): %s",
			strings.Join(mf, ", "))
	}

	return nil
}

func getMissingArgs(c *side_effects.Config) []string {
	var r []string

	if c.Tko.GetProxySocket() == "" {
		r = append(r, "proxy socket")
	}

	if c.Tko.GetMysqlUser() == "" {
		r = append(r, "MySQL user")
	}

	if c.Tko.GetEncryptedMysqlPassword() == "" {
		r = append(r, "Encrypted MySQL password")
	}

	if c.GoogleStorage.GetBucket() == "" {
		r = append(r, "Google Storage bucket")
	}

	return r
}

func getMissingFiles(c *side_effects.Config) []string {
	var r []string

	if _, err := os.Stat(c.Tko.ProxySocket); err != nil {
		r = append(r, err.Error()+" (proxy socket)")
	}

	return r
}

const configFileName = "side_effects_config.json"

// WriteConfigToDisk writes a JSON encoded side_effects.Config proto to
// <dir>/side_effects_config.json.
func WriteConfigToDisk(dir string, c *side_effects.Config) error {
	f := filepath.Join(dir, configFileName)
	w, err := os.Create(f)
	if err != nil {
		return errors.Annotate(err, "write side_effects_config.json to disk").Err()
	}
	defer w.Close()

	marshaler := jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, c); err != nil {
		return errors.Annotate(err, "write side_effects_config.json to disk").Err()
	}
	return nil
}

// PopulateTKOPasswordFile decrypts the encrypted MySQL password, writes it
// to a temp file and updates the corresponding config field.
func PopulateTKOPasswordFile(ctx context.Context, ckc cloudkms.Client, c *side_effects.Config) error {
	pwd, err := ckc.Decrypt(ctx, c.GetTko().GetEncryptedMysqlPassword())
	if err != nil {
		return errors.Annotate(err, "populate TKO password file").Err()
	}

	f, err := ioutil.TempFile("", "tko_password")
	if err != nil {
		return errors.Annotate(err, "populate TKO password file").Err()
	}
	defer f.Close()
	_, err = f.Write(pwd)
	if err != nil {
		return errors.Annotate(err, "populate TKO password file").Err()
	}

	c.GetTko().MysqlPasswordFile = f.Name()

	return nil
}

// CleanupTempFiles deletes all temp files used in side_effects.Config.
func CleanupTempFiles(c *side_effects.Config) error {
	f := c.GetTko().GetMysqlPasswordFile()
	if _, err := os.Stat(f); os.IsNotExist(err) {
		return nil
	}
	if err := os.Remove(f); err != nil {
		return errors.Annotate(err, "cleanup temp side effects files").Err()
	}
	return nil
}

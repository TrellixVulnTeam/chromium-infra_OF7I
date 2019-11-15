// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package sideeffects

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/google/uuid"
	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/side_effects"
)

func basicConfig() *side_effects.Config {
	return &side_effects.Config{
		Tko: &side_effects.TKOConfig{
			ProxySocket:            tempFile(),
			MysqlUser:              "foo-user",
			EncryptedMysqlPassword: "encrypted-password",
		},
		GoogleStorage: &side_effects.GoogleStorageConfig{
			Bucket:          "foo-bucket",
			CredentialsFile: tempFile(),
		},
	}
}

func tempFile() string {
	f, _ := ioutil.TempFile("", "")
	return f.Name()
}

func TestSuccess(t *testing.T) {
	Convey("Given a complete config pointing to existing files", t, func() {
		cfg := basicConfig()
		err := ValidateConfig(cfg)
		Convey("no error is returned.", func() {
			So(err, ShouldBeNil)
		})
	})
}

func TestMissingArgs(t *testing.T) {
	Convey("Given a side_effects.Config with a missing", t, func() {
		cases := []struct {
			name         string
			fieldDropper func(*side_effects.Config)
		}{
			{
				name: "proxy socket",
				fieldDropper: func(c *side_effects.Config) {
					c.Tko.ProxySocket = ""
				},
			},
			{
				name: "MySQL user",
				fieldDropper: func(c *side_effects.Config) {
					c.Tko.MysqlUser = ""
				},
			},
			{
				name: "Encrypted MySQL password",
				fieldDropper: func(c *side_effects.Config) {
					c.Tko.EncryptedMysqlPassword = ""
				},
			},
			{
				name: "Google Storage bucket",
				fieldDropper: func(c *side_effects.Config) {
					c.GoogleStorage.Bucket = ""
				},
			},
		}
		for _, c := range cases {
			Convey(c.name, func() {
				cfg := basicConfig()
				c.fieldDropper(cfg)
				err := ValidateConfig(cfg)
				Convey("then the correct error is returned.", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, c.name)
				})
			})
		}
	})
}

func TestMissingFiles(t *testing.T) {
	Convey("Given a missing", t, func() {
		cases := []struct {
			name        string
			fileDropper func(c *side_effects.Config)
		}{
			{
				name: "proxy socket",
				fileDropper: func(c *side_effects.Config) {
					c.Tko.ProxySocket = uuid.New().String()
				},
			},
		}
		for _, c := range cases {
			Convey(c.name, func() {
				cfg := basicConfig()
				c.fileDropper(cfg)
				err := ValidateConfig(cfg)
				Convey("then the correct error is returned.", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, c.name)
				})
			})
		}
	})
}

func TestWriteConfigToDisk(t *testing.T) {
	Convey("Given side_effects.Config object", t, func() {
		want := side_effects.Config{
			Tko: &side_effects.TKOConfig{
				ProxySocket:       "foo-socket",
				MysqlUser:         "foo-user",
				MysqlPasswordFile: "foo-password-file",
			},
			GoogleStorage: &side_effects.GoogleStorageConfig{
				Bucket:          "foo-bucket",
				CredentialsFile: "foo-creds",
			},
		}
		Convey("when WriteConfigToDisk is called", func() {
			dir, _ := ioutil.TempDir("", "")
			err := WriteConfigToDisk(dir, &want)
			So(err, ShouldBeNil)

			Convey("then the side_effects_config.json file contains the original object", func() {
				f, fileErr := os.Open(filepath.Join(dir, "side_effects_config.json"))
				So(fileErr, ShouldBeNil)

				var got side_effects.Config
				um := jsonpb.Unmarshaler{}
				unmarshalErr := um.Unmarshal(f, &got)
				So(unmarshalErr, ShouldBeNil)
				So(got, ShouldResemble, want)
			})
		})
	})
}

type fakeCloudKMSClient struct{}

func newFakeCloudKMSClient() *fakeCloudKMSClient {
	return &fakeCloudKMSClient{}
}

func (c *fakeCloudKMSClient) Decrypt(_ context.Context, _ string) ([]byte, error) {
	return []byte("decrypted-password"), nil
}

func TestPopulateTKOPasswordFile(t *testing.T) {
	Convey("Given side_effects.Config with an encrypted password", t, func() {
		ctx := context.Background()
		cfg := basicConfig()
		fc := newFakeCloudKMSClient()
		Convey("when PopulateTKOPasswordFile is called", func() {
			err := PopulateTKOPasswordFile(ctx, fc, cfg)
			So(err, ShouldBeNil)

			Convey("then side_effects.Config is populated with the password file path", func() {
				So(cfg.GetTko().GetMysqlPasswordFile(), ShouldNotBeBlank)

				Convey("which points to a file populated with right contents", func() {
					got, err := ioutil.ReadFile(cfg.GetTko().GetMysqlPasswordFile())
					So(err, ShouldBeNil)
					So(string(got), ShouldEqual, "decrypted-password")

					os.Remove(cfg.GetTko().GetMysqlPasswordFile())
				})
			})
		})
	})
}

func TestCleanupExistingFiles(t *testing.T) {
	Convey("Given side_effects.Config pointing to an existing MySQL password file", t, func() {
		f := tempFile()
		cfg := &side_effects.Config{
			Tko: &side_effects.TKOConfig{
				MysqlPasswordFile: f,
			},
		}
		Convey("when CleanupTempFiles is called", func() {
			err := CleanupTempFiles(cfg)
			So(err, ShouldBeNil)

			Convey("then the password file is removed from both disk and config", func() {
				_, err := os.Stat(f)
				So(err, ShouldNotBeNil)
				So(os.IsNotExist(err), ShouldBeTrue)
			})
		})
	})
}

func TestCleanupNonExistingFiles(t *testing.T) {
	Convey("Given side_effects.Config pointing to a non existing MySQL password file", t, func() {
		cfg := &side_effects.Config{
			Tko: &side_effects.TKOConfig{
				MysqlPasswordFile: uuid.New().String(),
			},
		}
		Convey("when CleanupTempFiles is called it does not return an error", func() {
			err := CleanupTempFiles(cfg)
			So(err, ShouldBeNil)
		})
	})
}

// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package deviceconfig

import (
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/luci/common/logging"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"

	inv "infra/cros/lab_inventory/protos"
)

func compareBoxsterWithV0(ctx context.Context, boxsterCfgs, v0Cfgs []*device.Config) error {
	writer, err := getCloudStorageWriter(ctx, "cros-lab-inventory-dev.appspot.com", fmt.Sprintf("device_config_diff/%s.log", getDiffFilename()))
	if err != nil {
		return err
	}
	defer func() {
		if writer != nil {
			if err := writer.Close(); err != nil {
				logging.Warningf(ctx, "failed to close cloud storage writer: %s", err)
			}
		}
	}()

	logging.Debugf(ctx, "start comparing device configs from boxster & V0")
	v0CfgMap := make(map[string]*device.Config)
	boxsterCfgMap := make(map[string]*device.Config)
	for _, c := range v0Cfgs {
		v0CfgMap[GetDeviceConfigIDStr(c.GetId())] = c
	}
	for _, c := range boxsterCfgs {
		boxsterCfgMap[GetDeviceConfigIDStr(c.GetId())] = c
	}

	count := 0
	var diff []string
	var logs []string
	for _, c := range boxsterCfgs {
		idStr := GetDeviceConfigIDStr(c.GetId())
		if found, ok := v0CfgMap[idStr]; ok {
			// device.Config is not in type protoreflect.ProtoMessage as it's not generated via cproto.
			// Convert device.Config to inv.Config
			newFound := copyDeviceConfig(found)
			newC := copyDeviceConfig(c)
			if !proto.Equal(found, c) {
				diff = append(diff, fmt.Sprintf("ID: %s", idStr))
				names := []protoreflect.Name{
					protoreflect.Name("ee"),
					protoreflect.Name("tam"),
					protoreflect.Name("soc_email_group"),
					protoreflect.Name("odm_email_group"),
					protoreflect.Name("oem_email_group"),
					protoreflect.Name("oem"),
					protoreflect.Name("odm"),
					protoreflect.Name("id"),
				}
				opt1 := protocmp.IgnoreFields(newFound, names...)
				diff = append(diff, cmp.Diff(newFound, newC, protocmp.Transform(), opt1))
				count++
			}
		}
	}

	logs = append(logs, fmt.Sprintf("######## %d different device configs ############", count))
	logs = append(logs, diff...)

	logs = append(logs, "\n\n######## device config ID only exists in boxster ############")
	for _, c := range boxsterCfgs {
		idStr := GetDeviceConfigIDStr(c.GetId())
		if _, ok := v0CfgMap[idStr]; !ok {
			logs = append(logs, idStr)
		}
	}

	logs = append(logs, "\n\n######## device config ID only exists in device config V0 ############")
	for _, c := range v0Cfgs {
		idStr := GetDeviceConfigIDStr(c.GetId())
		if _, ok := boxsterCfgMap[idStr]; !ok {
			logs = append(logs, idStr)
		}
	}

	if _, err := fmt.Fprintf(writer, strings.Join(logs, "\n")); err != nil {
		return err
	}
	return nil
}

func getDiffFilename() string {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err == nil {
		return time.Now().UTC().In(loc).Format("2006-01-02T03:04:05")
	}
	return time.Now().UTC().Format("2006-01-02T03:04:05")
}

func getCloudStorageWriter(ctx context.Context, bucketName, filename string) (*storage.Writer, error) {
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		logging.Warningf(ctx, "failed to create cloud storage client")
		return nil, err
	}
	bucket := storageClient.Bucket(bucketName)
	logging.Infof(ctx, "all diff will be saved to https://storage.cloud.google.com/%s/%s", bucketName, filename)
	return bucket.Object(filename).NewWriter(ctx), nil
}

func copyDeviceConfig(old *device.Config) *inv.Config {
	if old == nil {
		return nil
	}
	s := proto.MarshalTextString(old)
	var newDC inv.Config
	proto.UnmarshalText(s, &newDC)
	return &newDC
}

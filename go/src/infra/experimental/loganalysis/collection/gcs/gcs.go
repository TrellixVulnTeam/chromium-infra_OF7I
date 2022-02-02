// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gcs

import (
	"context"
	"io/ioutil"
	"strings"

	"cloud.google.com/go/storage"

	"go.chromium.org/luci/common/errors"
)

const (
	BucketID          = "chromeos-test-logs"
	LogsName          = "log.txt"
	StoragePathPrefix = "storage"
)

// ReadFileContents reads logs contents with a given bucket ID and an object ID.
// Example of a bucket ID: "chromeos-test-logs"
// Example of an object ID: "test-runner/prod/2021-12-12/0340bd1e-e9c8-464c-afa2-d968934c747a/autoserv_test/tast/results/tests/crostini.DisplayDensity.x11_bullseye_unstable/log.txt"
func ReadFileContents(ctx context.Context, bucketID string, objID string) ([]byte, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "create client").Err()
	}
	r, err := client.Bucket(bucketID).Object(objID).NewReader(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "create reader").Err()
	}
	defer r.Close()
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Annotate(err, "read logs file contents").Err()
	}
	return bytes, nil
}

// CreateObjectID creates object ID for a log given its basic information.
func CreateObjectID(logsURL string, test string, bucketID string, logsName string) string {
	return strings.TrimPrefix(logsURL, "/browse/"+bucketID+"/") + "/autoserv_test/tast/results/tests/" + test[5:] + "/" + logsName
}

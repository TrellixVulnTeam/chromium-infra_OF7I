// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	bqlib "infra/libs/cros/lab_inventory/bq"
	ufspb "infra/unifiedfleet/api/v1/proto"
	apibq "infra/unifiedfleet/api/v1/proto/bigquery"
	"infra/unifiedfleet/app/model/configuration"
)

func dumpConfigurations(ctx context.Context, bqClient *bigquery.Client, curTimeStr string) (err error) {
	defer func() {
		dumpChromePlatformTick.Add(ctx, 1, err == nil)
	}()

	uploader := bqlib.InitBQUploaderWithClient(ctx, bqClient, "ufs", fmt.Sprintf("chrome_platforms$%s", curTimeStr))
	platforms, err := configuration.GetAllChromePlatforms(ctx)
	if err != nil {
		return errors.Annotate(err, "dump chrome platforms").Err()
	}
	msgs := make([]proto.Message, 0)
	for _, p := range *platforms {
		if p.Err != nil {
			continue
		}
		msg := &apibq.ChromePlatformRow{
			Platform: p.Data.(*ufspb.ChromePlatform),
		}
		msgs = append(msgs, msg)
	}
	logging.Debugf(ctx, "Dumping %d records of chrome_platform to bigquery", len(msgs))
	if err := uploader.Put(ctx, msgs...); err != nil {
		return err
	}
	return nil
}

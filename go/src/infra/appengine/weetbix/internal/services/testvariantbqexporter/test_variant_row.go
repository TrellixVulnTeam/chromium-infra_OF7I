// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantbqexporter

import (
	"context"
	"fmt"

	"infra/appengine/weetbix/internal/bqutil"
)

func (b *BQExporter) exportTestVariantRows(ctx context.Context, ins *bqutil.Inserter) error {
	return fmt.Errorf("not implemented")
}

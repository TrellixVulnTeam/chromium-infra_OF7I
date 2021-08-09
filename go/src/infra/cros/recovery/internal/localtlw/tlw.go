// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package localtlw

import (
	"context"
	"infra/libs/lro"

	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/longrunning"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
)

// CacheForDut queries the underlying TLW server to find a healthy devserver
// with a cached version of the given chromeOS image, and returns the URL
// of the cached image on the devserver.
func CacheForDut(ctx context.Context, conn *grpc.ClientConn, imageURL, dutName string) (string, error) {
	c := tls.NewWiringClient(conn)
	resOperation, err := c.CacheForDut(ctx, &tls.CacheForDutRequest{
		Url:     imageURL,
		DutName: dutName,
	})
	if err != nil {
		return "", err
	}
	operation, err := lro.Wait(ctx, longrunning.NewOperationsClient(conn), resOperation.Name)
	if err != nil {
		return "", errors.Annotate(err, "cache for dut: failed to wait for CacheForDut").Err()
	}
	if s := operation.GetError(); s != nil {
		return "", errors.Reason("cache for dut: failed to get CacheForDut, %s", s).Err()
	}
	opRes := operation.GetResponse()
	if opRes == nil {
		return "", errors.Reason("cacheForDut: failed to get CacheForDut response for URL=%s and Name=%s", imageURL, dutName).Err()
	}
	resp := &tls.CacheForDutResponse{}
	if err := ptypes.UnmarshalAny(opRes, resp); err != nil {
		return "", errors.Annotate(err, "cache for dut: unexpected response from CacheForDut, %v", opRes).Err()
	}
	return resp.GetUrl(), nil
}

// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"context"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/appstatus"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/grpc/codes"

	"github.com/golang/protobuf/proto"
)

const allowGroup = "weetbix-access"

func commonPrelude(ctx context.Context, methodName string, req proto.Message) (context.Context, error) {
	if err := checkAllowed(ctx); err != nil {
		return ctx, err
	}
	return ctx, nil
}

func commonPostlude(ctx context.Context, methodName string, rsp proto.Message, err error) error {
	return appstatus.GRPCifyAndLog(ctx, err)
}

func checkAllowed(ctx context.Context) error {
	switch yes, err := auth.IsMember(ctx, allowGroup); {
	case err != nil:
		return errors.Annotate(err, "failed to check ACL").Err()
	case !yes:
		return appstatus.Errorf(codes.PermissionDenied, "not a member of %s", allowGroup)
	default:
		return nil
	}
}

package rpc

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/grpc/appstatus"
)

// Checksif this call is allowed, returns an error if it is.
func checkAllowedPrelude(ctx context.Context, methodName string, req proto.Message) (context.Context, error) {
	if err := checkAllowed(ctx); err != nil {
		return ctx, err
	}
	return ctx, nil
}

// Logs and converts the errors to GRPC type errors.
func gRPCifyAndLogPostlude(ctx context.Context, methodName string, rsp proto.Message, err error) error {
	return appstatus.GRPCifyAndLog(ctx, err)
}

// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package localtlw

import (
	"context"
	"infra/libs/lro"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"

	"infra/cros/recovery/tlw"
)

// TLSProvision calls TLS service and request provisioning with force.
func TLSProvision(ctx context.Context, conn *grpc.ClientConn, req *tlw.ProvisionRequest) error {
	c := tls.NewCommonClient(conn)
	op, err := c.ProvisionDut(
		ctx,
		&tls.ProvisionDutRequest{
			Name: req.GetResource(),
			TargetBuild: &tls.ChromeOsImage{
				PathOneof: &tls.ChromeOsImage_GsPathPrefix{
					GsPathPrefix: req.GetSystemImagePath(),
				},
			},
			ForceProvisionOs: true,
			PreventReboot:    req.GetPreventReboot(),
		},
	)
	if err != nil {
		// Errors here indicate a failure in starting the operation, not failure
		// in the provision attempt.
		return errors.Annotate(err, "tls provision").Err()
	}

	operation, err := lro.Wait(ctx, longrunning.NewOperationsClient(conn), op.GetName())
	if err != nil {
		return errors.Annotate(err, "tls provision: failed to wait").Err()
	}
	if s := operation.GetError(); s != nil {
		return errors.Reason("tls provision: failed to provision, %s", s).Err()
	}
	return nil
}

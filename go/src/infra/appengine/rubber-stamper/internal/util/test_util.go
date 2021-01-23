// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/signing"
	"go.chromium.org/luci/server/auth/signing/signingtest"
	"go.chromium.org/luci/server/tq"
	"go.chromium.org/luci/server/tq/tqtesting"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/internal/gerrit"
)

// SetupTestingContext contains all the context setup things needed for
// testing. It returns the context, a MockGerritClient and a
func SetupTestingContext(ctx context.Context, cfg *config.Config, serviceAccount, host string, t *testing.T) (context.Context, *gerritpb.MockGerritClient, *tqtesting.Scheduler) {
	ctx = gerrit.Setup(ctx)
	config.SetTestConfig(ctx, cfg)
	ctx = auth.ModifyConfig(ctx, func(cfg auth.Config) auth.Config {
		cfg.Signer = signingtest.NewSigner(&signing.ServiceInfo{
			ServiceAccountName: serviceAccount,
		})
		return cfg
	})

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	gerritMock := gerritpb.NewMockGerritClient(ctl)
	clientMap := map[string]gerrit.Client{
		host + "-review.googlesource.com": gerritMock,
	}
	ctx = gerrit.SetTestClientFactory(ctx, clientMap)

	ctx, sched := tq.TestingContext(ctx, nil)

	return ctx, gerritMock, sched
}

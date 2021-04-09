// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package tlslib

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/cros/tlslib/internal/nebraska"
)

// CreateFakeOmaha implements TLS CreateFakeOmaha API.
func (s *Server) CreateFakeOmaha(ctx context.Context, req *tls.CreateFakeOmahaRequest) (_ *tls.FakeOmaha, err error) {
	fo := req.GetFakeOmaha()
	gsPathPrefix := fo.GetTargetBuild().GetGsPathPrefix()
	if gsPathPrefix == "" {
		return nil, status.Errorf(codes.InvalidArgument, "empty GS path in the target build")
	}
	payloads := fo.GetPayloads()
	if len(payloads) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no payloads specified")
	}
	for _, p := range payloads {
		if p.GetType() == tls.FakeOmaha_Payload_TYPE_UNSPECIFIED {
			return nil, status.Errorf(codes.InvalidArgument, "payload %q has unspecified type", p.GetId())
		}
	}

	dutName := fo.GetDut()
	if dutName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "empty DUT name")
	}
	updatePayloadsAddress, err := s.cacheForDut(ctx, gsPathPrefix, dutName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get cache payload URL: %s", err)
	}
	n, err := nebraska.NewServer(ctx, nebraska.NewEnvironment(), gsPathPrefix, payloads, updatePayloadsAddress)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to start fake Omaha: %s", err)
	}
	defer func() {
		if err != nil {
			if err := n.Close(); err != nil {
				log.Printf("CreateFakeOmaha: close Nebraska when can't expose but failed: %s", err)
			}
		}
	}()

	c := nebraska.Config{
		CriticalUpdate:         fo.GetCriticalUpdate(),
		ReturnNoupdateStarting: int(fo.GetReturnNoupdateStarting()),
	}
	if err := n.UpdateConfig(c); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to config fake Omaha: %s", err)
	}
	u, err := s.exposePort(ctx, dutName, n.Port(), fo.GetExposedViaProxy())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to expose fake Omaha: %s", err)
	}
	// We don't have to close the tunnel created above in case we have errors
	// hereafter. All resources allocated to TLW are closed/released after the
	// "session" of the test.

	fo.Name = fmt.Sprintf("fakeOmaha/%s", uuid.New().String())
	exposedURL := url.URL{Scheme: "http", Host: u, Path: "/update"}
	fo.OmahaUrl = exposedURL.String()

	if err := s.resMgr.Add(fo.Name, n); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create resource: %s", err)
	}
	log.Printf("CreateFakeOmaha: %q update URL: %s", fo.Name, fo.OmahaUrl)
	return fo, nil
}

// DeleteFakeOmaha implements TLS DeleteFakeOmaha API.
func (s *Server) DeleteFakeOmaha(ctx context.Context, req *tls.DeleteFakeOmahaRequest) (*empty.Empty, error) {
	r, err := s.resMgr.Remove(req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete resource: %s", err)
	}
	if err := r.Close(); err != nil {
		return nil, status.Errorf(codes.Internal, "close fake Omaha: %s", err)
	}
	return nil, nil
}

func (s *Server) exposePort(ctx context.Context, dutName string, localPort int, requireProxy bool) (string, error) {
	c := s.wiringClient()
	rsp, err := c.ExposePortToDut(ctx, &tls.ExposePortToDutRequest{
		DutName:            dutName,
		LocalPort:          int32(localPort),
		RequireRemoteProxy: requireProxy,
	})
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(rsp.GetExposedAddress(), strconv.Itoa(int(rsp.GetExposedPort()))), nil
}

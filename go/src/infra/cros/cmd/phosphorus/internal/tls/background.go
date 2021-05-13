// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tls

import (
	"context"
	"fmt"
	"infra/libs/lro"

	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
)

const (
	droneTLWPort = 7151
)

// BackgroundTLS represents a TLS server and a client for using it.
type BackgroundTLS struct {
	server *Server
	Client *grpc.ClientConn
}

// Close cleans up resources associated with the BackgroundTLS.
func (b *BackgroundTLS) Close() error {
	// Make it safe to Close() more than once.
	if b.server == nil {
		return nil
	}
	err := b.Client.Close()
	b.server.Stop()
	b.server = nil
	return err
}

// NewBackgroundTLS runs a TLS server in the background and create a gRPC client to it.
//
// On success, the caller must call BackgroundTLS.Close() to clean up resources.
func NewBackgroundTLS() (*BackgroundTLS, error) {
	s, err := StartBackground(fmt.Sprintf("0.0.0.0:%d", droneTLWPort))
	if err != nil {
		return nil, errors.Annotate(err, "start background TLS").Err()
	}
	c, err := grpc.Dial(s.Address(), grpc.WithInsecure())
	if err != nil {
		s.Stop()
		return nil, errors.Annotate(err, "connect to background TLS").Err()
	}
	return &BackgroundTLS{
		server: s,
		Client: c,
	}, nil
}

// CacheForDut queries the underlying TLW server to find a healthy devserver
// with a cached version of the given chromeOS image, and returns the URL
// of the cached image on the devserver.
func (b *BackgroundTLS) CacheForDut(ctx context.Context, imageURL, dutName string) (string, error) {
	s := b.server
	c := tls.NewWiringClient(s.tlwConn)
	op, err := c.CacheForDut(ctx, &tls.CacheForDutRequest{
		Url:     imageURL,
		DutName: dutName,
	})
	if err != nil {
		return "", err
	}

	op, err = lro.Wait(ctx, longrunning.NewOperationsClient(s.tlwConn), op.Name)
	if err != nil {
	}
	if err != nil {
		return "", fmt.Errorf("cacheForDut: failed to wait for CacheForDut, %s", err)
	}

	if s := op.GetError(); s != nil {
		return "", fmt.Errorf("cacheForDut: failed to get CacheForDut, %s", s)
	}

	a := op.GetResponse()
	if a == nil {
		return "", fmt.Errorf("cacheForDut: failed to get CacheForDut response for URL=%s and Name=%s", imageURL, dutName)
	}

	resp := &tls.CacheForDutResponse{}
	if err := ptypes.UnmarshalAny(a, resp); err != nil {
		return "", fmt.Errorf("cacheForDut: unexpected response from CacheForDut, %v", a)
	}

	return resp.GetUrl(), nil
}

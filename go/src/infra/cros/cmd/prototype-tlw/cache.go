// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"log"
	"net/url"

	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
)

// cache implements the logic for the CacheForDut method and runs as a goroutine.
func (s *server) cache(ctx context.Context, parsedURL *url.URL, dutName string, opName string) {
	log.Printf("CacheForDut: Started Operation = %v", opName)
	// Devserver URL to be used. In the "real" CacheForDut implementation,
	// devservers should be resolved here.
	const baseURL = "http://chromeos6-devserver2.cros:8888/download/"
	path := parsedURL.Host + parsedURL.Path
	a, _ := ptypes.MarshalAny(
		&tls.CacheForDutResponse{
			Url: baseURL + path,
		},
	)
	opResult := &longrunning.Operation_Response{
		Response: a,
	}
	if err := s.lroMgr.setResult(opName, opResult); err != nil {
		log.Printf("CacheForDut: failed while updating result due to: %s", err)
	}
	log.Printf("CacheForDut: Operation Completed = %v", opName)
}

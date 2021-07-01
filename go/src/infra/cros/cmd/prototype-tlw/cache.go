// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"log"
	"net/url"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
)

// cache implements the logic for the CacheForDut method and runs as a goroutine.
func (s *server) cache(ctx context.Context, parsedURL *url.URL, dutName, opName string) {
	log.Printf("CacheForDut: Started Operation = %v", opName)
	// GS cache URL to be used.
	// In the "real" CacheForDut implementation, GS cache should be resolved here.
	// TODO(guocb): Use chromeos6-cache2 instead of IP.
	const baseURL = "http://100.115.220.100:8888/download/"
	if err := s.lroMgr.SetResult(opName, &tls.CacheForDutResponse{
		Url: baseURL + parsedURL.Host + parsedURL.Path,
	}); err != nil {
		log.Printf("CacheForDut: failed while updating result due to: %s", err)
	}
	log.Printf("CacheForDut: Operation Completed = %v", opName)
}

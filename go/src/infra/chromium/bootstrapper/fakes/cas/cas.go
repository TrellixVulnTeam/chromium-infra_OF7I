// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cas

import (
	"context"
	bscas "infra/chromium/bootstrapper/cas"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/filemetadata"
	"go.chromium.org/luci/common/errors"
)

// Instance provides the fake data for a RBE-CAS instance.
type Instance struct {
	// blobs maps the hashes that identify a blob to whether or not the blob
	// exists.
	//
	// Missing hashes will be treated as existing.
	blobs map[string]bool
}

// Client is the client that will serve fake data for a given instance
type Client struct {
	name     string
	instance *Instance
}

// Factory creates a factory that returns CAS clients that use fake data to
// respond to requests.
//
// The fake data is taken from the fakes argument, which is a map from instance
// names to the Instance containing the fake data for the instance. Missing keys
// will have a default Instance. A nil value indicates that the given instance
// does not exist.
func Factory(fakes map[string]*Instance) bscas.CasClientFactory {
	return func(ctx context.Context, instance string) (bscas.CasClient, error) {
		fake, ok := fakes[instance]
		if !ok {
			fake = &Instance{}
		} else if fake == nil {
			return nil, errors.Reason("%s is not a CAS instance", instance).Err()
		}
		return &Client{instance, fake}, nil
	}
}

func (c *Client) DownloadDirectory(ctx context.Context, d digest.Digest, execRoot string, cache filemetadata.Cache) (map[string]*client.TreeOutput, *client.MovedBytesMetadata, error) {
	blobExists, ok := c.instance.blobs[d.Hash]
	if ok && !blobExists {
		return nil, nil, errors.Reason("hash %s does not identify any blobs in instance %s", d.Hash, c.name).Err()
	}
	return nil, nil, nil
}

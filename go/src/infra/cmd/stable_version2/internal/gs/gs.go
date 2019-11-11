// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gs

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/common/gcloud/gs"

	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
)

// Client refers to a google storage client.
type Client struct {
	C            gs.Client
	unmarshaller jsonpb.Unmarshaler
}

// Init a Google Storage client.
func (gsc *Client) Init(ctx context.Context, t http.RoundTripper, unmarshaler jsonpb.Unmarshaler) error {
	c, err := gs.NewProdClient(ctx, t)
	if err != nil {
		return err
	}
	gsc.C = c
	gsc.unmarshaller = unmarshaler
	return nil
}

// MetaFilePath returns gs path for meta file
func MetaFilePath(crosSV *sv.StableCrosVersion) gs.Path {
	p := fmt.Sprintf("gs://chromeos-image-archive/%s-release/%s/metadata.json", crosSV.GetKey().GetBuildTarget().GetName(), crosSV.GetVersion())
	return gs.Path(p)
}

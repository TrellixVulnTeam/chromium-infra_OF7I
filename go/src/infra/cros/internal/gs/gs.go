// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package gs

import (
	"context"
	gerrs "errors"
	"net/http"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/gcloud/gs"
	"google.golang.org/api/googleapi"
)

type Client interface {
	WriteFileToGS(gsPath gs.Path, data []byte) error
}

type ProdClient struct {
	client gs.Client
}

func NewProdClient(ctx context.Context, authedClient *http.Client) (*ProdClient, error) {
	gsClient, err := gs.NewProdClient(ctx, authedClient.Transport)
	if err != nil {
		return nil, errors.Annotate(err, "new Google Storage client").Err()
	}
	return &ProdClient{
		client: gsClient,
	}, nil
}

// WriteFileToGS writes the specified data to the specified gs path.
func (g *ProdClient) WriteFileToGS(gsPath gs.Path, data []byte) error {
	gsWriter, err := g.client.NewWriter(gsPath)
	if err != nil {
		return err
	}
	_, err = gsWriter.Write(data)
	if err != nil {
		return errors.Annotate(err, "error writing manifest to gs path %v", gsPath).Err()
	}

	if err := gsWriter.Close(); err != nil {
		var ge *googleapi.Error
		if gerrs.As(err, &ge) && ge.Code == 404 {
			return errors.Annotate(err, "GS returned 404, are you sure that bucket %s exists?", gsPath.Bucket()).Err()
		} else {
			return errors.Annotate(err, "error writing manifest to gs path %v", gsPath).Err()
		}
	}

	return nil
}

// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package gs

import (
	"bytes"
	"context"
	gerrs "errors"
	"io"
	"net/http"
	"os"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/gcloud/gs"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Client interface {
	WriteFileToGS(gsPath gs.Path, data []byte) error
	Download(gsPath gs.Path, localPath string) error
	Read(gsPath gs.Path) ([]byte, error)
	List(ctx context.Context, bucket string, prefix string) ([]string, error)
}

type ProdClient struct {
	client      gs.Client
	plainClient *storage.Client
}

func NewProdClient(ctx context.Context, authedClient *http.Client) (*ProdClient, error) {
	gsClient, err := gs.NewProdClient(ctx, authedClient.Transport)
	if err != nil {
		return nil, errors.Annotate(err, "new Google Storage client").Err()
	}
	plainClient, err := storage.NewClient(ctx, option.WithHTTPClient(&http.Client{
		Transport: authedClient.Transport,
	}))

	return &ProdClient{
		client:      gsClient,
		plainClient: plainClient,
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

// Download reads the specified path from gs to the specified local path.
func (g *ProdClient) Download(gsPath gs.Path, localPath string) error {
	r, err := g.client.NewReader(gsPath, 0, -1)
	if err != nil {
		return errors.Annotate(err, "download").Err()
	}
	w, err := os.Create(localPath)
	if err != nil {
		return errors.Annotate(err, "download").Err()
	}
	if _, err := io.Copy(w, r); err != nil {
		return errors.Annotate(err, "download %s to %s", gsPath, localPath).Err()
	}
	return nil
}

// Read reads the specified path from gs and returns its contents.
func (g *ProdClient) Read(gsPath gs.Path) ([]byte, error) {
	r, err := g.client.NewReader(gsPath, 0, -1)
	if err != nil {
		return nil, errors.Annotate(err, "download").Err()
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(r)

	return buf.Bytes(), nil
}

// List lists all the files in a specific bucket matching the given prefix.
func (g *ProdClient) List(ctx context.Context, bucket string, prefix string) ([]string, error) {
	bkt := g.plainClient.Bucket(bucket)
	query := &storage.Query{Prefix: prefix}

	var names []string
	it := bkt.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		names = append(names, attrs.Name)
	}
	return names, nil
}

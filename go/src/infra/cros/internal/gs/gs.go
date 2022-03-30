// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package gs

import (
	"bytes"
	"context"
	gerrs "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"infra/cros/internal/cmd"
	"infra/cros/internal/shared"

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
	DownloadWithGsutil(ctx context.Context, gsPath gs.Path, localPath string) error
	Read(gsPath gs.Path) ([]byte, error)
	SetTTL(ctx context.Context, gsPath gs.Path, ttl time.Duration) error
	SetMetadata(ctx context.Context, gsPath gs.Path, key, value string) error
	List(ctx context.Context, bucket string, prefix string) ([]string, error)
}

type ProdClient struct {
	noAuth      bool
	client      gs.Client
	plainClient *storage.Client
}

func NewProdClient(ctx context.Context, authedClient *http.Client) (*ProdClient, error) {
	if authedClient != nil {
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

	return &ProdClient{
		noAuth: true,
	}, nil
}

// WriteFileToGS writes the specified data to the specified gs path.
func (g *ProdClient) WriteFileToGS(gsPath gs.Path, data []byte) error {
	if g.noAuth {
		return fmt.Errorf("client was initialized without auth, this method is unsupported")
	}
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
	if g.noAuth {
		return fmt.Errorf("client was initialized without auth, this method is unsupported")
	}
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

// DownloadWithGsutil reads the specified path from gs to the specified local
// path by shelling out to `gsutil` (which must exist on the client's path).
// This function is used in contexts where OAuth-based authentication is not
// available.
func (g *ProdClient) DownloadWithGsutil(ctx context.Context, gsPath gs.Path, localPath string) error {
	cmdRunner := cmd.RealCommandRunner{}
	cmd := []string{"cat", string(gsPath)}
	var stdoutBuf, stderrBuf bytes.Buffer
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := cmdRunner.RunCommand(ctx, &stdoutBuf, &stderrBuf, cwd, "gsutil", cmd...); err != nil {
		if strings.Contains(stderrBuf.String(), "404") && strings.Contains(stderrBuf.String(), "does not exist") ||
			strings.Contains(stderrBuf.String(), "No URLs matched") {
			return errors.Annotate(shared.ErrObjectNotExist, "download (%s)", stderrBuf.String()).Err()
		}
		return errors.Annotate(err, "download (%s)", stderrBuf.String()).Err()
	}
	f, err := os.Create(localPath)
	if err != nil {
		return errors.Annotate(err, "download").Err()
	}
	if _, err := f.Write(stdoutBuf.Bytes()); err != nil {
		return errors.Annotate(err, "download %s to %s", gsPath, localPath).Err()
	}
	return nil
}

// Read reads the specified path from gs and returns its contents.
func (g *ProdClient) Read(gsPath gs.Path) ([]byte, error) {
	if g.noAuth {
		return nil, fmt.Errorf("client was initialized without auth, this method is unsupported")
	}
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
	if g.noAuth {
		return nil, fmt.Errorf("client was initialized without auth, this method is unsupported")
	}
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

// Set TTL sets the object's time to live.
func (g *ProdClient) SetTTL(ctx context.Context, gsPath gs.Path, ttl time.Duration) error {
	return g.SetMetadata(ctx, gsPath, "timetolive", strconv.Itoa(int(ttl.Seconds())))
}

// SetMetadata sets object metadata for an arbitrary key-value pair.
func (g *ProdClient) SetMetadata(ctx context.Context, gsPath gs.Path, key, value string) error {
	if g.noAuth {
		return fmt.Errorf("client was initialized without auth, this method is unsupported")
	}
	bucket := gsPath.Bucket()
	path := gsPath.Filename()
	_, err := g.plainClient.Bucket(bucket).Object(path).Update(ctx, storage.ObjectAttrsToUpdate{
		Metadata: map[string]string{
			key: value,
		},
	})
	return err
}

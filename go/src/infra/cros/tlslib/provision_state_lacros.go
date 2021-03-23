// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tlslib

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"golang.org/x/crypto/ssh"
)

const (
	lacrosRootComponentPath = "/var/lib/imageloader/lacros"
	pageSize                = 4096
)

type lacrosMetadata struct {
	Content struct {
		Version string `json:"version"`
	} `json:"content"`
}

type provisionLacrosState struct {
	s                   *Server
	c                   *ssh.Client
	dutName             string
	gsPathPrefix        string
	metadata            lacrosMetadata
	lacrosComponentPath string
	lacrosImagePath     string
	lacrosTablePath     string
}

func newProvisionLacrosState(s *Server, req *tls.ProvisionLacrosRequest) (*provisionLacrosState, error) {
	p := &provisionLacrosState{
		s:       s,
		dutName: req.Name,
	}

	// Verify the incoming request path oneof is valid.
	switch t := req.GetImage().GetPathOneof().(type) {
	case *tls.ProvisionLacrosRequest_LacrosImage_GsPathPrefix:
		p.gsPathPrefix = req.GetImage().GetGsPathPrefix()
	default:
		return nil, fmt.Errorf("newProvisionStateLacros: unsupported ImagePathOneof in ProvisionLacrosRequest, %T", t)
	}

	return p, nil
}

func (p *provisionLacrosState) connect(ctx context.Context, addr string) (func(), error) {
	c, err := p.s.clientPool.GetContext(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("connect: DUT unreachable, %s", err)
	}

	p.c = c
	disconnect := func() {
		p.s.clientPool.Put(addr, p.c)
	}
	return disconnect, nil
}

func (p *provisionLacrosState) provisionLacros(ctx context.Context) error {
	if err := p.extractLacrosMetadata(ctx); err != nil {
		return fmt.Errorf("provisionLacros: failed to extract Lacros metadata.json, %s", err)
	}

	if err := p.installLacrosAsComponent(ctx); err != nil {
		return fmt.Errorf("provisionLacros: failed to install Lacros as component, %s", err)
	}

	// Get the Lacros image hash. (Must be after installing Lacros image and running verity).
	lacrosImageHash, err := getSHA256Sum(ctx, p.c, p.lacrosImagePath)
	if err != nil {
		return fmt.Errorf("provisionLacros: failed to get Lacros image hash, %s", err)
	}

	// Get the Lacros table hash.
	lacrosTableHash, err := getSHA256Sum(ctx, p.c, p.lacrosTablePath)
	if err != nil {
		return fmt.Errorf("provisionLacros: failed to get Lacros table hash, %s", err)
	}

	// Create the Lacros manifest file.
	if err := p.writeManifest(ctx, lacrosImageHash, lacrosTableHash); err != nil {
		return fmt.Errorf("provisionLacros: failed to write Lacros manifest, %s", err)
	}

	// Write the Lacros version to the latest-version file.
	lacrosLastestVersionPath := path.Join(lacrosRootComponentPath, "latest-version")
	if err := runCmd(p.c, fmt.Sprintf("echo -n %s > %s", p.metadata.Content.Version, lacrosLastestVersionPath)); err != nil {
		return fmt.Errorf("provisionLacros: failed to write Lacros version to latest-version file, %s", err)
	}

	return nil
}

// extractLacrosMetadata will unmarshal the metadata.json in the GS path into the state.
func (p *provisionLacrosState) extractLacrosMetadata(ctx context.Context) error {
	lacrosGSMetadataPath := path.Join(p.gsPathPrefix, "metadata.json")
	// TODO(kimjae): Use CacheForDrone TLS API once implemented and then download
	// from TLS side instead of from the DUT.
	url, err := p.s.cacheForDut(ctx, lacrosGSMetadataPath, p.dutName)
	if err != nil {
		return fmt.Errorf("extractMetadata: failed to CacheForDut Lacros metadata.json, %s", err)
	}
	metadataJSONStr, err := runCmdOutput(p.c, fmt.Sprintf("curl -s %s", url))
	if err != nil {
		return fmt.Errorf("extractMetadata: failed to read Lacros metadata.json, %s", err)
	}
	metadataJSON := lacrosMetadata{}
	if err := json.Unmarshal([]byte(metadataJSONStr), &metadataJSON); err != nil {
		return fmt.Errorf("extractMetadata: failed to unmarshal Lacros metadata.json, %s", err)
	}
	p.metadata = metadataJSON
	return nil
}

// installLacrosAsComponent will download and install the Lacros image into the imageloader
// component directory with the version from the metadata.json file.
// The Lacros image will also be modified with the correct verity output.
func (p *provisionLacrosState) installLacrosAsComponent(ctx context.Context) error {
	lacrosGSImagePath := path.Join(p.gsPathPrefix, "lacros_compressed.squash")
	url, err := p.s.cacheForDut(ctx, lacrosGSImagePath, p.dutName)
	if err != nil {
		return fmt.Errorf("installLacrosAsComponent: failed to CacheForDut, %s", err)
	}
	p.lacrosComponentPath = path.Join(lacrosRootComponentPath, p.metadata.Content.Version)
	p.lacrosImagePath = path.Join(p.lacrosComponentPath, "image.squash")
	if err := runCmd(p.c, fmt.Sprintf("mkdir -p %s && curl %s --output %s", p.lacrosComponentPath, url, p.lacrosImagePath)); err != nil {
		return fmt.Errorf("installLacrosAsComponent: failed to install Lacros image from %s, %s", lacrosGSImagePath, err)
	}

	lacrosBlocks, err := alignImageToPage(ctx, p.c, p.lacrosImagePath)
	if err != nil {
		return fmt.Errorf("installLacrosAsComponent: failed to align Lacros image, %s", err)
	}

	// Generate the verity (hashtree and table) from Lacros image.
	lacrosHashtreePath := path.Join(p.lacrosComponentPath, "hashtree")
	p.lacrosTablePath = path.Join(p.lacrosComponentPath, "table")
	if err := runCmd(p.c,
		fmt.Sprintf("verity mode=create alg=sha256 payload=%s payload_blocks=%d hashtree=%s salt=random > %s",
			p.lacrosImagePath, lacrosBlocks, lacrosHashtreePath, p.lacrosTablePath)); err != nil {
		return fmt.Errorf("installLacrosAsComponent: failed to generate verity for Lacros image, %s", err)
	}

	// Append the hashtree (merkle tree) onto the end of the Lacros image.
	if err := runCmd(p.c, fmt.Sprintf("cat %s >> %s", lacrosHashtreePath, p.lacrosImagePath)); err != nil {
		return fmt.Errorf("installLacrosAsComponent: failed to append hashtree to Lacros image, %s", err)
	}

	return nil
}

// getSHA256Sum will get the SHA256 sum of a file on the device.
func getSHA256Sum(ctx context.Context, c *ssh.Client, path string) (string, error) {
	hash, err := runCmdOutput(c, fmt.Sprintf("sha256sum %s | cut -d' ' -f1", path))
	if err != nil {
		return "", fmt.Errorf("getSHA256Sum: failed to get hash of %s, %s", path, err)
	}
	return strings.TrimSpace(hash), nil
}

// alignImageToPage will align the file to 4KB page alignment and return the number of page blocks.
func alignImageToPage(ctx context.Context, c *ssh.Client, path string) (int, error) {
	sizeStr, err := runCmdOutput(c, "stat -c%s "+path)
	if err != nil {
		return 0, fmt.Errorf("alignImageToPage: failed to get image size, %s", err)
	}
	sizeStr = strings.TrimSpace(sizeStr)
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return 0, fmt.Errorf("alignImageToPage: failed to get image size as an integer, %s", err)
	}

	// Round up to the nearest 4KB block size.
	blocks := (size + pageSize - 1) / pageSize

	// Check if the Lacros image is 4KB aligned, if not extend it to 4KB alignment.
	if size != blocks*pageSize {
		log.Printf("alignImageToPage: image %s isn't 4KB aligned, so extending it", path)
		if err := runCmd(c, fmt.Sprintf("dd if=/dev/zero bs=1 count=%d seek=%d of=%s",
			blocks*pageSize-size, size, path)); err != nil {
			return 0, fmt.Errorf("alignImageToPage: failed to align image to 4KB, %s", err)
		}
	}
	return blocks, nil
}

// writeManifest will create and write the Lacros component manifest out.
func (p *provisionLacrosState) writeManifest(ctx context.Context, imageHash, tableHash string) error {
	lacrosManifestJSON, err := json.MarshalIndent(struct {
		ManifestVersion int    `json:"manifest-version"`
		FsType          string `json:"fs-type"`
		Version         string `json:"version"`
		ImageSha256Hash string `json:"image-sha256-hash"`
		TableSha256Hash string `json:"table-sha256-hash"`
	}{
		ManifestVersion: 1,
		FsType:          "squashfs",
		Version:         p.metadata.Content.Version,
		ImageSha256Hash: imageHash,
		TableSha256Hash: tableHash,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("writeManifest: failed to Marshal Lacros manifest json, %s", err)
	}
	lacrosManifestPath := path.Join(p.lacrosComponentPath, "imageloader.json")
	if err := runCmd(p.c, fmt.Sprintf("echo '%s' > %s", lacrosManifestJSON, lacrosManifestPath)); err != nil {
		return fmt.Errorf("writeManifest: failed to write Lacros manifest json to DUT, %s", err)
	}
	return nil
}

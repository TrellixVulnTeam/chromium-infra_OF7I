// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tlslib

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"golang.org/x/crypto/ssh"
)

const (
	imageloaderLacrosRootComponentPath = "/var/lib/imageloader/lacros"
	pageSize                           = 4096
)

var versionRegex = regexp.MustCompile(`^(\d+\.)(\d+\.)(\d+\.)(\d+)$`)

type lacrosMetadata struct {
	Content struct {
		Version string `json:"version"`
	} `json:"content"`
}

// lacrosSourceType describes the source types of Lacros for provisioning (either GS or device file).
type lacrosSourceType int

const (
	GsPath lacrosSourceType = iota
	DeviceFilePath
)

type provisionLacrosState struct {
	s                       *Server
	c                       *ssh.Client
	dutName                 string
	sourceURL               string
	sourceType              lacrosSourceType
	metadata                lacrosMetadata
	lacrosComponentRootPath string // component path, eg. "/home/chronos/cros-components/lacros-dogfood-dev/"
	lacrosComponentPath     string // versioned path, eg. "/home/chronos/cros-components/lacros-dogfood-dev/9999.0.0.0/"
	lacrosImagePath         string
	lacrosTablePath         string
	overrideVersion         string
}

func newProvisionLacrosState(s *Server, req *tls.ProvisionLacrosRequest) (*provisionLacrosState, error) {
	p := &provisionLacrosState{
		s:       s,
		dutName: req.Name,
	}

	// Verify the incoming request path oneof is valid.
	switch t := req.GetImage().GetPathOneof().(type) {
	case *tls.ProvisionLacrosRequest_LacrosImage_GsPathPrefix:
		p.sourceURL = req.GetImage().GetGsPathPrefix()
		p.sourceType = GsPath
	case *tls.ProvisionLacrosRequest_LacrosImage_DeviceFilePrefix:
		p.sourceURL = req.GetImage().GetDeviceFilePrefix()
		p.sourceType = DeviceFilePath
	default:
		return nil, fmt.Errorf("newProvisionStateLacros: unsupported ImagePathOneof in ProvisionLacrosRequest, %T", t)
	}

	// Verify the override version is valid if |OverrideVersion| is specified.
	if req.OverrideVersion != "" {
		if !versionRegex.MatchString(req.OverrideVersion) {
			return nil, fmt.Errorf("newProvisionLacrosState: failed to parse version: %v", req.OverrideVersion)
		}
		p.overrideVersion = req.OverrideVersion
	}

	// Override the component root path if |OverrideInstallPath| is specified.
	if req.OverrideInstallPath != "" {
		p.lacrosComponentRootPath = req.OverrideInstallPath
	} else {
		// Use the ImageLoader component path by default.
		p.lacrosComponentRootPath = imageloaderLacrosRootComponentPath
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

	// Create the component updater Lacros manifest file.
	if err := p.writeComponentManifest(ctx); err != nil {
		return fmt.Errorf("provisionLacros: failed to write component Lacros manifest, %s", err)
	}

	// Write the Lacros version to the latest-version file.
	lacrosLastestVersionPath := path.Join(p.lacrosComponentRootPath, "latest-version")
	if err := runCmd(p.c, fmt.Sprintf("echo -n %s > %s", p.metadata.Content.Version, lacrosLastestVersionPath)); err != nil {
		return fmt.Errorf("provisionLacros: failed to write Lacros version to latest-version file, %s", err)
	}

	// Change file mode and owner of provisioned files if the path is prefixed with the CrOS component path.
	const crosComponentRootPath = "/home/chronos/cros-components"
	if strings.HasPrefix(p.lacrosComponentRootPath, crosComponentRootPath) {
		if err := runCmd(p.c, fmt.Sprintf("chown -R chronos:chronos %s && chmod -R 0755 %s", crosComponentRootPath, crosComponentRootPath)); err != nil {
			return fmt.Errorf("provisionLacros: failed to chown/chmod provisioned files under %s, %s", crosComponentRootPath, err)
		}
	}
	return nil
}

// extractLacrosMetadata will unmarshal the metadata.json in the GS path into the state.
func (p *provisionLacrosState) extractLacrosMetadata(ctx context.Context) error {
	metadataURL, err := p.getSourceFullURL(ctx, "metadata.json")
	if err != nil {
		return fmt.Errorf("extractMetadata: failed to get a metadata URL, %s", err)
	}
	metadataJSONStr, err := runCmdOutput(p.c, fmt.Sprintf("curl -s %s", metadataURL))
	if err != nil {
		return fmt.Errorf("extractMetadata: failed to read Lacros metadata.json from %v, %s", metadataURL, err)
	}

	metadataJSON := lacrosMetadata{}
	if err := json.Unmarshal([]byte(metadataJSONStr), &metadataJSON); err != nil {
		return fmt.Errorf("extractMetadata: failed to unmarshal Lacros metadata.json, %s", err)
	}
	p.metadata = metadataJSON
	if p.overrideVersion != "" {
		log.Printf("extractLacrosMetadata: override version %v to %v\n", p.metadata.Content.Version, p.overrideVersion)
		p.metadata.Content.Version = p.overrideVersion
	}
	return nil
}

// installLacrosAsComponent will download and install the Lacros image into the imageloader
// component directory with the version from the metadata.json file.
// The Lacros image will also be modified with the correct verity output.
func (p *provisionLacrosState) installLacrosAsComponent(ctx context.Context) error {
	var imageFileName string
	switch p.sourceType {
	case GsPath:
		imageFileName = "lacros_compressed.squash"
	case DeviceFilePath:
		imageFileName = "lacros.squash"
	default:
		return fmt.Errorf("installLacrosAsComponent: unknown source type: %v", p.sourceType)
	}
	imageURL, err := p.getSourceFullURL(ctx, imageFileName)
	if err != nil {
		return fmt.Errorf("installLacrosAsComponent: failed to get a image URL, %s", err)
	}

	p.lacrosComponentPath = path.Join(p.lacrosComponentRootPath, p.metadata.Content.Version)
	p.lacrosImagePath = path.Join(p.lacrosComponentPath, "image.squash")

	if err := runCmd(p.c, fmt.Sprintf("mkdir -p %s && curl %s --output %s", p.lacrosComponentPath, imageURL, p.lacrosImagePath)); err != nil {
		return fmt.Errorf("installLacrosAsComponent: failed to install Lacros image from %s, %s", imageURL, err)
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

// writeComponentManifest will create and write the Lacros component manifest out usable by component updater.
func (p *provisionLacrosState) writeComponentManifest(ctx context.Context) error {
	lacrosComponentManifestJSON, err := json.MarshalIndent(struct {
		ManifestVersion int    `json:"manifest-version"`
		Name            string `json:"name"`
		Version         string `json:"version"`
		ImageName       string `json:"imageName"`
		Squash          bool   `json:"squash"`
		FsType          string `json:"fsType"`
		IsRemovable     bool   `json:"isRemovable"`
	}{
		ManifestVersion: 2,
		Name:            "lacros",
		Version:         p.metadata.Content.Version,
		ImageName:       "image.squash",
		Squash:          true,
		FsType:          "squashfs",
		IsRemovable:     false,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("writeComponentManifest: failed to Marshal Lacros manifest json, %s", err)
	}
	lacrosComponentManifestPath := path.Join(p.lacrosComponentPath, "manifest.json")
	if err := runCmd(p.c, fmt.Sprintf("echo '%s' > %s", lacrosComponentManifestJSON, lacrosComponentManifestPath)); err != nil {
		return fmt.Errorf("writeComponentManifest: failed to write Lacros manifest json to DUT, %s", err)
	}
	return nil
}

// getSourceFullURL returns a full URL of the Lacros binary files in a given base URL and path.
func (p *provisionLacrosState) getSourceFullURL(ctx context.Context, part string) (string, error) {
	url, err := func(base string, part string) (*url.URL, error) {
		parsed, err := url.Parse(base)
		if err != nil {
			return nil, err
		}
		parsed.Path = path.Join(parsed.Path, part)
		return parsed, nil
	}(p.sourceURL, part)
	if err != nil {
		return "", fmt.Errorf("getSourceFullURL: failed to join a URL, %s", err)
	}

	fullURL := ""
	switch p.sourceType {
	case GsPath:
		// TODO(kimjae): Use CacheForDrone TLS API once implemented and then download
		// from TLS side instead of from the DUT.
		fullURL, err = p.s.cacheForDut(ctx, url.String(), p.dutName)
		if err != nil {
			return "", fmt.Errorf("getSourceFullURL: failed to CacheForDut, %s", err)
		}
	case DeviceFilePath:
		if url.Scheme != "" && url.Scheme != "file" {
			return "", fmt.Errorf("getSourceFullURL: wrong URL scheme. expected: file, actual: %s, url:%s", url.Scheme, url.String())
		}
		url.Scheme = "file"
		fullURL = url.String()
	}
	return fullURL, nil
}

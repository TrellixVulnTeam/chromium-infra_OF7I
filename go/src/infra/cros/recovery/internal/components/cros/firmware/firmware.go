// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package firmware

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/logger"
)

const (
	extractFileTimeout = 300 * time.Second
	ecMonitorFileName  = "npcx_monitor.bin"
)

// Helper function to extract EC image from downloaded tarball.
func extractECImage(ctx context.Context, tarballPath string, run components.Runner, log logger.Logger, board, model string) (string, error) {
	if board == "" || model == "" {
		return "", errors.Reason("extract ec files: board or model is not provided").Err()
	}
	destDir := filepath.Join(filepath.Dir(tarballPath), "EC")
	candidatesFiles := []string{
		"ec.bin",
		fmt.Sprintf("%s/ec.bin", model),
		fmt.Sprintf("%s/ec.bin", board),
	}
	imagePath, err := extractFromTarball(ctx, tarballPath, destDir, candidatesFiles, run, log)
	if err != nil {
		return "", errors.Annotate(err, "extract ec files").Err()
	}
	// Extract subsidiary binaries for EC
	// Find a monitor binary for NPCX_UUT chip type, if any.
	var monitorFiles []string
	for _, f := range candidatesFiles {
		monitorFiles = append(monitorFiles, strings.Replace(f, "ec.bin", ecMonitorFileName, 1))
	}
	if _, err := extractFromTarball(ctx, tarballPath, destDir, monitorFiles, run, log); err != nil {
		log.Debug("Extract EC files: fail to extract %q file. Error: %s", ecMonitorFileName, err)
	}
	return filepath.Join(destDir, imagePath), nil
}

// Helper function to extract BIOS image from downloaded tarball.
func extractAPImage(ctx context.Context, tarballPath string, run components.Runner, log logger.Logger, board, model string) (string, error) {
	if board == "" || model == "" {
		return "", errors.Reason("extract ap files: board or model is not provided").Err()
	}
	destDir := filepath.Join(filepath.Dir(tarballPath), "AP")
	candidatesFiles := []string{
		"image.bin",
		fmt.Sprintf("image-%s.bin", model),
		fmt.Sprintf("image-%s.bin", board),
	}
	imagePath, err := extractFromTarball(ctx, tarballPath, destDir, candidatesFiles, run, log)
	if err != nil {
		return "", errors.Annotate(err, "extract ec files").Err()
	}
	return filepath.Join(destDir, imagePath), nil
}

const (
	createDirSafeGlob         = "mkdir -p %s"
	tarballListTheFileGlob    = "tar tf %s"
	tarballExtractTheFileGlob = "tar xf %s -C %s %s"
)

// Try extracting the image_candidates from the tarball.
func extractFromTarball(ctx context.Context, tarballPath, destDirPath string, candidates []string, run components.Runner, log logger.Logger) (string, error) {
	// Create the firmware_name subdirectory if it doesn't exist
	if _, err := run(ctx, fmt.Sprintf(createDirSafeGlob, destDirPath), extractFileTimeout); err != nil {
		return "", errors.Annotate(err, "extract from tarball: fail to create a destination directory %s", destDirPath).Err()
	}
	// Generate a list of all tarball files
	tarballFiles := make(map[string]bool, 50)
	if out, err := run(ctx, fmt.Sprintf(tarballListTheFileGlob, tarballPath), extractFileTimeout); err != nil {
		return "", errors.Annotate(err, "extract from tarball").Err()
	} else {
		for _, fn := range strings.Split(out, "\n") {
			tarballFiles[fn] = true
		}
	}
	// Check if image candidates are in the list of tarball files.
	for _, cf := range candidates {
		if !tarballFiles[cf] {
			log.Debug("Extract from tarball: candidate file %q is not in tarball.", cf)
			continue
		}
		cmd := fmt.Sprintf(tarballExtractTheFileGlob, tarballPath, destDirPath, cf)
		if _, err := run(ctx, cmd, extractFileTimeout); err != nil {
			log.Debug("Extract from tarball: candidate %q fail to be extracted from tarball.", cf)
		} else {
			return cf, nil
		}
	}
	return "", errors.Reason("extract from tarball: no candidate file found").Err()
}

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

// ReadAPInfoRequest holds request date to read AP info.
type ReadAPInfoRequest struct {
	FilePath string
	// Force extract AP from the DUT.
	ForceExtractAPFile bool
	GBBFlags           bool
	Keys               bool
}

// ReadAPInfoResponse holds response of AP info.
type ReadAPInfoResponse struct {
	GBBFlagsRaw string
	GBBFlags    int
	Keys        []string
}

// ReadAPInfoByServo read AP info from DUT.
//
// AP will be extracted from the DUT to flash back with changes.
func ReadAPInfoByServo(ctx context.Context, req *ReadAPInfoRequest, run components.Runner, servod components.Servod, log logger.Logger) (*ReadAPInfoResponse, error) {
	if run == nil || servod == nil || log == nil {
		return nil, errors.Reason("read ap info: run, servod or logger is not provided").Err()
	}
	p, err := NewProgrammer(ctx, run, servod, log)
	if err != nil {
		return nil, errors.Annotate(err, "read ap info").Err()
	}
	if err := p.ExtractAP(ctx, req.FilePath, req.ForceExtractAPFile); err != nil {
		return nil, errors.Annotate(err, "read ap info").Err()
	}
	res := &ReadAPInfoResponse{}
	if req.GBBFlags {
		cmd := fmt.Sprintf("gbb_utility --get --flags %s", req.FilePath)
		gbbOut, err := run(ctx, 30*time.Second, cmd)
		if err != nil {
			return nil, errors.Annotate(err, "read ap info: read flags").Err()
		}
		// Parsing output to extract real GBB value.
		parts := strings.Split(gbbOut, ":")
		if len(parts) < 2 {
			return nil, errors.Annotate(err, "read ap info: gbb not found").Err()
		} else if raw := strings.TrimSpace(parts[1]); raw == "" {
			return nil, errors.Annotate(err, "read ap info: gbb not found").Err()
		} else {
			log.Info("Read GBB raw: %v", raw)
			res.GBBFlagsRaw = raw
		}
		gbb, err := gbbToInt(res.GBBFlagsRaw)
		if err != nil {
			return nil, errors.Annotate(err, "read ap info").Err()
		}
		log.Debug("Read GBB flags: %v", gbb)
		res.GBBFlags = gbb
	}
	if req.Keys {
		cmd := fmt.Sprintf("futility show %s |grep \"Key sha1sum:\" |awk '{print $3}'", req.FilePath)
		out, err := run(ctx, 30*time.Second, cmd)
		if err != nil {
			return nil, errors.Annotate(err, "read ap info: read flags").Err()
		}
		log.Debug("Read firmware keys: %v", out)
		res.Keys = strings.Split(out, "\n")
	}
	return res, nil
}

// SetApInfoByServoRequest hols and provides info to update AP.
type SetApInfoByServoRequest struct {
	// Path to where AP used or will be extracted
	FilePath string
	// Force extract AP from the DUT.
	ForceExtractAPFile bool
	UpdateGBBFlags     bool
	// GBB flags value need to be set to system.
	// Example: 0x18
	GBBFlags string
}

// SetApInfoByServo sets info to AP on the DUT by servo.
//
// AP will be extracted from the DUT to flash back with changes.
func SetApInfoByServo(ctx context.Context, req *SetApInfoByServoRequest, run components.Runner, servod components.Servod, log logger.Logger) error {
	if run == nil || servod == nil || log == nil {
		return errors.Reason("set ap info: run, servod or logger is not provided").Err()
	}
	p, err := NewProgrammer(ctx, run, servod, log)
	if err != nil {
		return errors.Annotate(err, "set ap info").Err()
	}
	if err := p.ExtractAP(ctx, req.FilePath, req.ForceExtractAPFile); err != nil {
		return errors.Annotate(err, "set ap info").Err()
	}
	log.Debug("Set AP info: starting flashing AP to the DUT")
	if err := p.ProgramAP(ctx, req.FilePath, req.GBBFlags); err != nil {
		return errors.Annotate(err, "set ap info: read flags").Err()
	}
	return nil
}

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
	if _, err := run(ctx, extractFileTimeout, fmt.Sprintf(createDirSafeGlob, destDirPath)); err != nil {
		return "", errors.Annotate(err, "extract from tarball: fail to create a destination directory %s", destDirPath).Err()
	}
	// Generate a list of all tarball files
	tarballFiles := make(map[string]bool, 50)
	if out, err := run(ctx, extractFileTimeout, fmt.Sprintf(tarballListTheFileGlob, tarballPath)); err != nil {
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
		if _, err := run(ctx, extractFileTimeout, cmd); err != nil {
			log.Debug("Extract from tarball: candidate %q fail to be extracted from tarball.", cf)
		} else {
			return cf, nil
		}
	}
	return "", errors.Reason("extract from tarball: no candidate file found").Err()
}

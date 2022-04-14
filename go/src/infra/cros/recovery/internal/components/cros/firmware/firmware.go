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
	"infra/cros/recovery/internal/components/servo"
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
	defer func() {
		if cerr := p.Close(ctx); cerr != nil {
			log.Debugf("Close programmer fail: %s", cerr)
		}
	}()
	p.Prepare(ctx)
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
			log.Infof("Read GBB raw: %v", raw)
			res.GBBFlagsRaw = raw
		}
		gbb, err := gbbToInt(res.GBBFlagsRaw)
		if err != nil {
			return nil, errors.Annotate(err, "read ap info").Err()
		}
		log.Debugf("Read GBB flags: %v", gbb)
		res.GBBFlags = gbb
	}
	if req.Keys {
		if keys, err := readAPKeysFromFile(ctx, req.FilePath, run, log); err != nil {
			return nil, errors.Annotate(err, "read ap info").Err()
		} else {
			res.Keys = keys
		}
	}
	return res, nil
}

const (
	DevSignedFirmwareKeyPrefix = "b11d"
)

// IsDevKeys checks if any of provided keys are dev signed.
func IsDevKeys(keys []string, log logger.Logger) bool {
	for _, key := range keys {
		if strings.HasPrefix(key, DevSignedFirmwareKeyPrefix) {
			log.Debugf("Found dev signed key: %q !", key)
			return true
		}
	}
	return false
}

// readAPKeysFromFile read firmware keys from the AP image.
func readAPKeysFromFile(ctx context.Context, filePath string, run components.Runner, log logger.Logger) ([]string, error) {
	cmd := fmt.Sprintf("futility show %s |grep \"Key sha1sum:\" |awk '{print $3}'", filePath)
	out, err := run(ctx, time.Minute, cmd)
	if err != nil {
		return nil, errors.Annotate(err, "read ap keys").Err()
	}
	log.Debugf("Read firmware keys: %v", out)
	return strings.Split(out, "\n"), nil
}

// SetApInfoByServoRequest holds and provides info to update AP.
type SetApInfoByServoRequest struct {
	// Path to where AP used or will be extracted
	FilePath string
	// Force extract AP from the DUT.
	ForceExtractAPFile bool
	UpdateGBBFlags     bool
	// GBB flags value need to be set to AP.
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
	defer func() {
		if cerr := p.Close(ctx); cerr != nil {
			log.Debugf("Close programmer fail: %s", cerr)
		}
	}()
	p.Prepare(ctx)
	if err := p.ExtractAP(ctx, req.FilePath, req.ForceExtractAPFile); err != nil {
		return errors.Annotate(err, "set ap info").Err()
	}
	log.Debugf("Set AP info: starting flashing AP to the DUT")
	err = p.ProgramAP(ctx, req.FilePath, req.GBBFlags)
	return errors.Annotate(err, "set ap info: read flags").Err()
}

const (
	extractFileTimeout = 300 * time.Second
	ecMonitorFileName  = "npcx_monitor.bin"
)

// InstallFwFromFwImageRequest holds info for InstallFwFromFwImage method to flash EC/AP on the DUT.
type InstallFwFromFwImageRequest struct {
	// Board and model of the DUT.
	Board string
	Model string

	// Dir where we download the fw image file and then extracted.
	DownloadDir string
	// Path to the fw-Image file and timeout to download it.
	DownloadImagePath    string
	DownloadImageTimeout time.Duration

	// Specify that Update EC is is requested.
	UpdateEC bool
	// Specify that Update AP is is requested.
	UpdateAP bool
	// GBB flags value need to be set to AP.
	// Example: 0x18
	GBBFlags string
}

// InstallFwFromFwImage updates EC/AP on the DUT by servo from fw-image.
func InstallFwFromFwImage(ctx context.Context, req *InstallFwFromFwImageRequest, run components.Runner, servod components.Servod, log logger.Logger) error {
	if req == nil || req.Board == "" || req.Model == "" {
		return errors.Reason("install fw from fw-image: request missed board/model data").Err()
	} else if !req.UpdateEC && !req.UpdateAP {
		return errors.Reason("install fw from fw-image: at least ec or ap update need to be requested").Err()
	}
	const (
		// Specify the name used for download file.
		downloadFilename = "fw_image.tar.bz2"
	)
	clearDirectory := func() {
		_, err := run(ctx, time.Minute, "rm", "-rf", req.DownloadDir)
		log.Debugf("Failed to remove download directory %q, Error: %s", req.DownloadDir, err)
	}
	// Remove directory in case something left from last times.
	clearDirectory()
	if _, err := run(ctx, time.Minute, "mkdir", "-p", req.DownloadDir); err != nil {
		return errors.Annotate(err, "install fw from fw-image").Err()
	}
	// Always clean up after creating folder as host has limit storage space.
	defer func() { clearDirectory() }()
	// Spicily filename for file to download.
	tarballPath := filepath.Join(req.DownloadDir, downloadFilename)
	if out, err := run(ctx, req.DownloadImageTimeout, "curl", req.DownloadImagePath, "--output", tarballPath); err != nil {
		log.Debugf("Output to download fw-image: %s", out)
		return errors.Annotate(err, "install fw from fw-image").Err()
	}
	p, err := NewProgrammer(ctx, run, servod, log)
	if err != nil {
		return errors.Annotate(err, "install fw from fw-image").Err()
	}
	log.Infof("Successful download tarbar %q from %q", tarballPath, req.DownloadImagePath)
	if req.UpdateEC {
		log.Debugf("Start extraction EC image from %q", tarballPath)
		ecImage, err := extractECImage(ctx, tarballPath, run, servod, log, req.Board, req.Model)
		if err != nil {
			return errors.Annotate(err, "install fw from fw-image").Err()
		}
		log.Debugf("Start program EC image %q", ecImage)
		if err := p.ProgramEC(ctx, ecImage); err != nil {
			return errors.Annotate(err, "install fw from fw-image").Err()
		}
		log.Infof("Finished program EC image %q", ecImage)
	}
	if req.UpdateAP {
		log.Debugf("Start extraction AP image from %q", tarballPath)
		apImage, err := extractAPImage(ctx, tarballPath, run, servod, log, req.Board, req.Model)
		if err != nil {
			return errors.Annotate(err, "install fw from fw-image").Err()
		}
		log.Debugf("Start program AP image %q", apImage)
		if err := p.ProgramAP(ctx, apImage, req.GBBFlags); err != nil {
			return errors.Annotate(err, "install fw from fw-image").Err()
		}
		log.Infof("Finished program AP image %q", apImage)
	}
	return nil
}

// Helper function to extract EC image from downloaded tarball.
func extractECImage(ctx context.Context, tarballPath string, run components.Runner, servod components.Servod, log logger.Logger, board, model string) (string, error) {
	if board == "" || model == "" {
		return "", errors.Reason("extract ec files: board or model is not provided").Err()
	}
	destDir := filepath.Join(filepath.Dir(tarballPath), "EC")
	candidatesFiles := []string{
		fmt.Sprintf("%s/ec.bin", model),
	}
	// TODO(b/226402941): Read existing ec image name using futility.
	if model == "dragonair" {
		candidatesFiles = append(candidatesFiles, "dratini/ec.bin")
	}
	if servod != nil {
		fwBoard, err := servo.GetString(ctx, servod, "ec_board")
		if err != nil {
			log.Debugf("Fail to read `ec_board` value from servo. Skipping.")
		}
		// Based on b:220157423 some board report name is upper case.
		fwBoard = strings.ToLower(fwBoard)
		if fwBoard != "" {
			candidatesFiles = append(candidatesFiles, fmt.Sprintf("%s/ec.bin", fwBoard))
		}
	}
	candidatesFiles = append(candidatesFiles,
		fmt.Sprintf("%s/ec.bin", board),
		"ec.bin",
	)
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
		log.Debugf("Extract EC files: fail to extract %q file. Error: %s", ecMonitorFileName, err)
	}
	return filepath.Join(destDir, imagePath), nil
}

// Helper function to extract BIOS image from downloaded tarball.
func extractAPImage(ctx context.Context, tarballPath string, run components.Runner, servod components.Servod, log logger.Logger, board, model string) (string, error) {
	if board == "" || model == "" {
		return "", errors.Reason("extract ap files: board or model is not provided").Err()
	}
	destDir := filepath.Join(filepath.Dir(tarballPath), "AP")
	candidatesFiles := []string{
		fmt.Sprintf("image-%s.bin", model),
	}
	// TODO(b/226402941): Read existing ec image name using futility.
	if model == "dragonair" {
		candidatesFiles = append(candidatesFiles, "image-dratini.bin")
	}
	if servod != nil {
		fwBoard, err := servo.GetString(ctx, servod, "ec_board")
		if err != nil {
			log.Debugf("Fail to read `ec_board` value from servo. Skipping.")
		}
		// Based on b:220157423 some board report name is upper case.
		fwBoard = strings.ToLower(fwBoard)
		if fwBoard != "" {
			candidatesFiles = append(candidatesFiles, fmt.Sprintf("image-%s.bin", fwBoard))
		}
	}
	candidatesFiles = append(candidatesFiles,
		fmt.Sprintf("image-%s.bin", board),
		"image.bin",
	)
	imagePath, err := extractFromTarball(ctx, tarballPath, destDir, candidatesFiles, run, log)
	if err != nil {
		return "", errors.Annotate(err, "extract ec files").Err()
	}
	return filepath.Join(destDir, imagePath), nil
}

// Try extracting the image_candidates from the tarball.
func extractFromTarball(ctx context.Context, tarballPath, destDirPath string, candidates []string, run components.Runner, log logger.Logger) (string, error) {
	const (
		// Extract list of files present in archive.
		// To avoid extraction of all files we can limit it t the list of files we interesting in by provide them as arguments at the end.
		tarballListTheFileGlob = "tar tf %s %s"
		// Extract file from the archive.
		tarballExtractTheFileGlob = "tar xf %s -C %s %s"
	)
	// Create the firmware_name subdirectory if it doesn't exist
	if _, err := run(ctx, extractFileTimeout, "mkdir", "-p", destDirPath); err != nil {
		return "", errors.Annotate(err, "extract from tarball: fail to create a destination directory %s", destDirPath).Err()
	}
	// Generate a list of all tarball files
	tarballFiles := make(map[string]bool, 50)
	cmd := fmt.Sprintf(tarballListTheFileGlob, tarballPath, strings.Join(candidates, " "))
	out, err := run(ctx, extractFileTimeout, cmd)
	if err != nil {
		log.Debugf("Fail with error: %s", err)
	}
	log.Debugf("Found candidates: %q", out)
	for _, fn := range strings.Split(out, "\n") {
		tarballFiles[fn] = true
	}
	// Check if image candidates are in the list of tarball files.
	for _, cf := range candidates {
		if !tarballFiles[cf] {
			log.Debugf("Extract from tarball: candidate file %q is not in tarball.", cf)
			continue
		}
		cmd := fmt.Sprintf(tarballExtractTheFileGlob, tarballPath, destDirPath, cf)
		if _, err := run(ctx, extractFileTimeout, cmd); err != nil {
			log.Debugf("Extract from tarball: candidate %q fail to be extracted from tarball.", cf)
		} else {
			log.Infof("Extract from tarball: candidate file %q extracted.", cf)
			return cf, nil
		}
	}
	return "", errors.Reason("extract from tarball: no candidate file found").Err()
}

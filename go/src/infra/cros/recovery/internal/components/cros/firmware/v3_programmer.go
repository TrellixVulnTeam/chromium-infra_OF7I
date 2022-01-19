// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package firmware

import (
	"context"
	base_errors "errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/logger"
)

type v3Programmer struct {
	run    components.Runner
	servod components.Servod
	log    logger.Logger
}

const (
	// Number of seconds for program EC/BIOS to time out.
	firmwareProgramTimeout = 1800 * time.Second

	// Tools and commands used for flashing EC.
	ecProgrammerToolName     = "flash_ec"
	ecProgrammerCmdGlob      = "flash_ec --chip=%s --image=%s --port=%d --verify --verbose"
	ecProgrammerStm32CmdGlob = "flash_ec --chip=%s --image=%s --port=%d --bitbang_rate=57600 --verify --verbose"

	// Tools and commands used for flashing AP.
	apProgrammerToolName  = "futility"
	apProgrammerCmdGlob   = "futility update -i %s --servo_port=%d"
	apProgrammerWithParam = " --gbb_flags=%d"
)

// ProgramEC programs EC firmware to devices by servo.
func (p *v3Programmer) ProgramEC(ctx context.Context, imagePath string) error {
	if err := isImageExist(imagePath); err != nil {
		return errors.Annotate(err, "program ec").Err()
	}
	return p.programEC(ctx, imagePath)
}

// programEC programs EC firmware to devices by servo.
// Extracted for test purpose to avoid file present check.
func (p *v3Programmer) programEC(ctx context.Context, imagePath string) error {
	if err := isToolPresent(ctx, ecProgrammerToolName, p.run); err != nil {
		return errors.Annotate(err, "program ec").Err()
	}
	ecChip, err := p.ecChip(ctx)
	if err != nil {
		return errors.Annotate(err, "program ec").Err()
	}
	var cmd string
	if ecChip == "stm32" {
		cmd = fmt.Sprintf(ecProgrammerStm32CmdGlob, ecChip, imagePath, p.servod.Port())
	} else {
		cmd = fmt.Sprintf(ecProgrammerCmdGlob, ecChip, imagePath, p.servod.Port())
	}
	out, err := p.run(ctx, cmd, firmwareProgramTimeout)
	p.log.Debug("Program EC output: \n%s", out)
	return errors.Annotate(err, "program ec").Err()
}

// ProgramAP programs AP firmware to devices by servo.
//
// To set/update GBB flags please provide value in hex representation.
// E.g. 0x18 to set force boot in DEV-mode and allow to boot from USB-drive in DEV-mode.
func (p *v3Programmer) ProgramAP(ctx context.Context, imagePath, gbbHex string) error {
	if err := isImageExist(imagePath); err != nil {
		return errors.Annotate(err, "program ec").Err()
	}
	return p.programAP(ctx, imagePath, gbbHex)
}

// programAP programs AP firmware to devices by servo.
// Extracted for test purpose to avoid file present check.
func (p *v3Programmer) programAP(ctx context.Context, imagePath, gbbHex string) error {
	if err := isToolPresent(ctx, apProgrammerToolName, p.run); err != nil {
		return errors.Annotate(err, "program ap").Err()
	}
	cmd := fmt.Sprintf(apProgrammerCmdGlob, imagePath, p.servod.Port())
	if gbbHex != "" {
		if v, err := gbbToInt(gbbHex); err != nil {
			return errors.Annotate(err, "program ap").Err()
		} else {
			cmd += fmt.Sprintf(apProgrammerWithParam, v)
		}
	}
	out, err := p.run(ctx, cmd, firmwareProgramTimeout)
	p.log.Debug("Program AP output: \n%s", out)
	return errors.Annotate(err, "program ap").Err()
}

// ecChip reads ec_chip from servod.
func (p *v3Programmer) ecChip(ctx context.Context) (string, error) {
	if ecChipI, err := p.servod.Get(ctx, "ec_chip"); err != nil {
		return "", errors.Annotate(err, "get ec_chip").Err()
	} else {
		return ecChipI.GetString_(), nil
	}
}

// gbbToInt converts hex value to int.
//
// E.g. 0x18 to set force boot in DEV-mode and allow to boot from USB-drive in DEV-mode.
func gbbToInt(hex string) (int, error) {
	hex = strings.ToLower(hex)
	hexCut := strings.Replace(hex, "0x", "", -1)
	if v, err := strconv.ParseInt(hexCut, 16, 64); err != nil {
		return 0, errors.Annotate(err, "gbb to int").Err()
	} else {
		return int(v), nil
	}
}

// isImageExist checks is provide image file exists.
func isImageExist(imagePath string) error {
	if _, err := os.Stat(imagePath); err == nil {
		// File exists.
		return nil
	} else if base_errors.Is(err, os.ErrNotExist) {
		return errors.Annotate(err, "image %q exist: file does not exist", imagePath).Err()
	} else {
		return errors.Annotate(err, "image %q exist: fail to check", imagePath).Err()
	}
}

// isToolPresent checks if tool is installed on the host.
func isToolPresent(ctx context.Context, toolName string, run components.Runner) error {
	cmd := fmt.Sprintf("which %s", toolName)
	_, err := run(ctx, cmd, 30*time.Second)
	return errors.Annotate(err, "tool %s is not found", toolName).Err()
}

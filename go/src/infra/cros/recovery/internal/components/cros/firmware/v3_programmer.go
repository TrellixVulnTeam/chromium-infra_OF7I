// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package firmware

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/components/servo"
	"infra/cros/recovery/logger"
)

// servodStateRecord holds state of servod before apply preparation of programmer.
type servodStateRecord struct {
	cmd string
	val interface{}
}

type v3Programmer struct {
	st     *servo.ServoType
	run    components.Runner
	servod components.Servod
	log    logger.Logger

	// Servod state before execution.
	servodState []*servodStateRecord
}

const (
	// Number of seconds for program EC/BIOS to time out.
	firmwareProgramTimeout = 30 * time.Minute

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
	if err := isFileExist(ctx, imagePath, p.run); err != nil {
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
	out, err := p.run(ctx, firmwareProgramTimeout, cmd)
	p.log.Debugf("Program EC output: \n%s", out)
	return errors.Annotate(err, "program ec").Err()
}

// ProgramAP programs AP firmware to devices by servo.
//
// To set/update GBB flags please provide value in hex representation.
// E.g. 0x18 to set force boot in DEV-mode and allow to boot from USB-drive in DEV-mode.
func (p *v3Programmer) ProgramAP(ctx context.Context, imagePath, gbbHex string) error {
	if err := isFileExist(ctx, imagePath, p.run); err != nil {
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
	out, err := p.run(ctx, firmwareProgramTimeout, cmd)
	p.log.Debugf("Program AP output:\n%s", out)
	return errors.Annotate(err, "program ap").Err()
}

// ExtractAP extracts AP firmware from device.
func (p *v3Programmer) ExtractAP(ctx context.Context, imagePath string, force bool) error {
	if imagePath == "" {
		return errors.Reason("extract ap from dut: path for extracting file is not provided").Err()
	}
	if force || isFileExist(ctx, imagePath, p.run) != nil {
		p.log.Infof("Proceed to extract AP from the DUT to %q path", imagePath)
		pn, err := p.name(ctx)
		if err != nil {
			return errors.Annotate(err, "extract ap from dut").Err()
		}
		p.log.Debugf("Using programmer %q", pn)
		// Reading AP from the DUT.
		args := []string{"-p", pn, "-f", "-r", imagePath}
		if out, err := p.run(ctx, firmwareProgramTimeout, "flashrom", args...); err != nil {
			return errors.Annotate(err, "extract ap from dut: read ap").Err()
		} else {
			p.log.Debugf("Extract AP: %v", out)
		}
	} else {
		p.log.Infof("AP file is present by %q and not need to extract it again", imagePath)
	}
	return nil
}

// name provides the name of programmer need to be used.
func (p *v3Programmer) name(ctx context.Context) (string, error) {
	var serialname string
	if res, err := p.servod.Get(ctx, p.st.SerialnameOption()); err != nil {
		return "", errors.Annotate(err, "name").Err()
	} else {
		serialname = res.GetString_()
	}
	if p.st.IsMicro() || p.st.IsC2D2() {
		return fmt.Sprintf("raiden_debug_spi:serial=%s", serialname), nil
	} else if p.st.IsCCD() {
		return fmt.Sprintf("raiden_debug_spi:target=AP,serial=%s", serialname), nil
	}
	return "", errors.Reason("name: Not supported servo type").Err()
}

// Prepare programmer for actions.
func (p *v3Programmer) Prepare(ctx context.Context) error {
	err := p.setServodState(ctx)
	return errors.Annotate(err, "prepare").Err()
}

func (p *v3Programmer) setServodState(ctx context.Context) error {
	p.log.Debugf("Set servod state to prepare programmer.")
	for _, s := range p.servodStateList() {
		sp := strings.Split(s, ":")
		if len(sp) != 2 {
			return errors.Reason("prepare servod state: state %q is incorrect", s).Err()
		}
		command := sp[0]
		val := sp[1]
		if cs, err := p.servod.Get(ctx, command); err != nil {
			return errors.Annotate(err, "prepare servod state: read servod state").Err()
		} else if val != cs.GetString_() {
			// If value is different then we need to save it so later we can restore it.
			r := &servodStateRecord{
				cmd: command,
				val: cs.GetString_(),
			}
			p.servodState = append(p.servodState, r)
		}
		if err := p.servod.Set(ctx, command, val); err != nil {
			return errors.Annotate(err, "prepare servod state: set servod state").Err()
		}
	}
	return nil
}

func (p *v3Programmer) restoreServodState(ctx context.Context) error {
	for _, s := range p.servodState {
		if err := p.servod.Set(ctx, s.cmd, s.val); err != nil {
			return errors.Annotate(err, "prepare servod state: set servod state").Err()
		}
	}
	return nil
}

func (p *v3Programmer) servodStateList() []string {
	if p.st.IsCCD() {
		return nil
	}
	return []string{
		"spi2_vref:pp3300", //Need verify as in some cases it can be pp1800
		"spi2_buf_en:on",
		"spi2_buf_on_flex_en:on",
		"spi_hold:off",
		"cold_reset:on",
		"usbpd_reset:on",
	}
}

// Close closes programming resources.
func (p *v3Programmer) Close(ctx context.Context) error {
	if err := p.restoreServodState(ctx); err != nil {
		return errors.Annotate(err, "close").Err()
	}
	return nil
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

// isFileExist checks is provided file exists.
func isFileExist(ctx context.Context, filepath string, run components.Runner) error {
	_, err := run(ctx, 30*time.Second, "test", "-f", filepath)
	return errors.Annotate(err, "if file exist: file %q does not exist", filepath).Err()
}

// isToolPresent checks if tool is installed on the host.
func isToolPresent(ctx context.Context, toolName string, run components.Runner) error {
	cmd := fmt.Sprintf("which %s", toolName)
	_, err := run(ctx, 30*time.Second, cmd)
	return errors.Annotate(err, "tool %s is not found", toolName).Err()
}

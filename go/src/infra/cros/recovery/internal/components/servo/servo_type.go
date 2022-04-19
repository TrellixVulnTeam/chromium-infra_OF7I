// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import "strings"

const (
	// Servo components/types used by system.
	SERVO_V2    = "servo_v2"
	SERVO_V3    = "servo_v3"
	SERVO_V4    = "servo_v4"
	SERVO_V4P1  = "servo_v4p1"
	CCD_CR50    = "ccd_cr50"
	CCD_GSC     = "ccd_gsc"
	C2D2        = "c2d2"
	SERVO_MICRO = "servo_micro"
	SWEETBERRY  = "sweetberry"

	// Prefix for CCD components.
	CCD_PREFIX = "ccd_"
)

var (
	// List of servos that connect to a debug header on the board.
	FLEX_SERVOS = []string{C2D2, SERVO_MICRO, SERVO_V3}
	// List of servos that rely on gsc commands for some part of dut control.
	GSC_DRV_SERVOS = []string{C2D2, CCD_GSC, CCD_CR50}
)

// ServoType represent structure to allow distinguishe servo components described in servo-type string.
type ServoType struct {
	str string
}

// NewServoType creates new ServoType with provided string representation.
func NewServoType(servoType string) *ServoType {
	return &ServoType{servoType}
}

// IsV2 checks whether the servo has a servo_v2 component.
func (s *ServoType) IsV2() bool {
	return strings.Contains(s.str, SERVO_V2)
}

// IsV3 checks whether the servo has a servo_v3 component.
func (s *ServoType) IsV3() bool {
	return strings.Contains(s.str, SERVO_V3)
}

// IsV4 checks whether the servo has servo_v4 or servo_v4p1 component.
func (s *ServoType) IsV4() bool {
	return strings.Contains(s.str, SERVO_V4)
}

// IsC2D2 checks whether the servo has a c2d2 component.
func (s *ServoType) IsC2D2() bool {
	return strings.Contains(s.str, C2D2)
}

// IsCCD checks whether the servo has a CCD component.
func (s *ServoType) IsCCD() bool {
	return strings.Contains(s.str, CCD_PREFIX)
}

// IsMicro checks whether the servo has a servo_micro component.
func (s *ServoType) IsMicro() bool {
	return strings.Contains(s.str, SERVO_MICRO)
}

// IsDualSetup checks whether the servo has a dual setup.
func (s *ServoType) IsDualSetup() bool {
	return s.IsV4() && (s.IsMicro() || s.IsC2D2()) && s.IsCCD()
}

// SerialnameOption returns servod control string that query serial number
// of servo component connected to the DUT directly.
func (s *ServoType) SerialnameOption() string {
	if s.IsV4() && s.IsMicro() {
		return "servo_micro_serialname"
	}
	if s.IsV4() && s.IsCCD() {
		return "ccd_serialname"
	}
	return "serialname"
}

// IsMultipleServos checks whether the servo has more than one component.
func (s *ServoType) IsMultipleServos() bool {
	return strings.Contains(s.str, "_and_")
}

// String provide ability to use ToString functionality.
func (s *ServoType) String() string {
	return s.str
}

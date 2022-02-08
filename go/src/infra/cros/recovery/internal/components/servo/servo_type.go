// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import "strings"

// ServoType represent structure to allow distinguishe servo components described in servo-type string.
type ServoType struct {
	str string
}

// NewServoType creates new ServoType with provided string representation.
func NewServoType(servoType string) *ServoType {
	return &ServoType{servoType}
}

// Servo used servo_v2 component.
func (s *ServoType) IsV2() bool {
	return strings.Contains(s.str, "servo_v2")
}

// Servo used servo_v3 component.
func (s *ServoType) IsV3() bool {
	return strings.Contains(s.str, "servo_v3")
}

// Servo used servo_v4 or servo_v4p1 component.
func (s *ServoType) IsV4() bool {
	return strings.Contains(s.str, "servo_v4")
}

// Servo used c2d2 component.
func (s *ServoType) IsC2D2() bool {
	return strings.Contains(s.str, "c2d2")
}

// Servo used cr50 component.
func (s *ServoType) IsCCD() bool {
	return strings.Contains(s.str, "ccd")
}

// Servo used servo_micro component.
func (s *ServoType) IsMicro() bool {
	return strings.Contains(s.str, "servo_micro")
}

// Servo has dual setup.
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

// Servo has more than one component.
func (s *ServoType) IsMultipleServos() bool {
	return strings.Contains(s.str, "_and_")
}

// String provide ability to use ToString functionality.
func (s *ServoType) String() string {
	return s.str
}

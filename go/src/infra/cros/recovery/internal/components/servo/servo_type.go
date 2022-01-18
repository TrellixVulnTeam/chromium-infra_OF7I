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

// String provide ability to use ToString functionality.
func (s *ServoType) String() string {
	return s.str
}

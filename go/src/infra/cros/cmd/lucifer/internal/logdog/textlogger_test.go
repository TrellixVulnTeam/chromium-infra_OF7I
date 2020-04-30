// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logdog

import "os"

func Example_text() {
	lg := NewTextLogger(os.Stdout)
	ConfigForTest(lg)
	lg.Print("Example logs")
	func() {
		s := lg.Step("step1")
		defer s.Close()
		s.Print("Some logs")
	}()
	func() {
		s := lg.Step("failure")
		defer s.Close()
		s.Failure()
		s.Print("Oops, we failed")
	}()
	func() {
		s := lg.Step("error")
		defer s.Close()
		s.Exception()
		s.Print("Some error happened")
	}()
	// Output:
	// example: Example logs
	// example: STEP step1
	// example: Some logs
	// example: STEP step1 OK
	// example: STEP failure
	// example: Oops, we failed
	// example: STEP failure FAIL
	// example: STEP error
	// example: Some error happened
	// example: STEP error ERROR
}

func Example_text_add_link() {
	lg := NewTextLogger(os.Stdout)
	ConfigForTest(lg)
	func() {
		s := lg.Step("step1")
		defer s.Close()
		s.AddLink("example", "https://example.com")
	}()
	// Output:
	// example: STEP step1
	// example: LINK example https://example.com
	// example: STEP step1 OK
}

func Example_text_substeps() {
	lg := NewTextLogger(os.Stdout)
	ConfigForTest(lg)
	func() {
		s := lg.Step("step1")
		defer s.Close()
		func() {
			s2 := s.Step("substep1")
			defer s2.Close()
		}()
	}()
	func() {
		s := lg.Step("step2")
		defer s.Close()
		func() {
			s2 := s.Step("substep2")
			defer s2.Close()
			func() {
				s3 := s2.Step("substep3")
				defer s3.Close()
			}()
		}()
	}()
	// Output:
	// example: STEP step1
	// example: STEP step1::substep1
	// example: STEP step1::substep1 OK
	// example: STEP step1 OK
	// example: STEP step2
	// example: STEP step2::substep2
	// example: STEP step2::substep2::substep3
	// example: STEP step2::substep2::substep3 OK
	// example: STEP step2::substep2 OK
	// example: STEP step2 OK
}

func Example_text_labeled_log() {
	lg := NewTextLogger(os.Stdout)
	ConfigForTest(lg)
	lg.LabeledLog("ijn", "ayanami")
	lg.LabeledLog("ijn", "jintsuu")
	// Output:
	// example: LOG ijn ayanami
	// example: LOG ijn jintsuu
}

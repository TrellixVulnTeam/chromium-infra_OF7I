// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logdog

import "os"

func Example() {
	lg := NewLogger(os.Stdout)
	// Configure the Logger for deterministic output.
	lg2 := lg.(*realLogger).logger
	lg2.SetFlags(0)
	lg2.SetPrefix("example: ")
	lg.Print("Example logs")
	func() {
		s := lg.Step("step1")
		defer s.Close()
		s.Print("Some logs")
	}()
	func() {
		s := lg.Step("failure")
		defer s.Close()
		s.Print("Oops, we failed")
		s.Failure()
	}()
	func() {
		s := lg.Step("error")
		defer s.Close()
		s.Print("Some error happened")
		s.Exception()
	}()
	// Output:
	// example: Example logs
	// @@@SEED_STEP step1@@@
	// @@@STEP_CURSOR step1@@@
	// @@@STEP_STARTED@@@
	// example: Some logs
	// @@@STEP_CLOSED@@@
	// @@@SEED_STEP failure@@@
	// @@@STEP_CURSOR failure@@@
	// @@@STEP_STARTED@@@
	// example: Oops, we failed
	// @@@STEP_FAILURE@@@
	// @@@STEP_CLOSED@@@
	// @@@SEED_STEP error@@@
	// @@@STEP_CURSOR error@@@
	// @@@STEP_STARTED@@@
	// example: Some error happened
	// @@@STEP_EXCEPTION@@@
	// @@@STEP_CLOSED@@@
}

func Example_add_link() {
	lg := NewLogger(os.Stdout)
	func() {
		s := lg.Step("step1")
		defer s.Close()
		s.AddLink("example", "https://example.com")
	}()
	// Output:
	// @@@SEED_STEP step1@@@
	// @@@STEP_CURSOR step1@@@
	// @@@STEP_STARTED@@@
	// @@@STEP_LINK@example@https://example.com@@@
	// @@@STEP_CLOSED@@@
}

func Example_substeps() {
	lg := NewLogger(os.Stdout)
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
	// @@@SEED_STEP step1@@@
	// @@@STEP_CURSOR step1@@@
	// @@@STEP_STARTED@@@
	// @@@SEED_STEP substep1@@@
	// @@@STEP_CURSOR substep1@@@
	// @@@STEP_NEST_LEVEL@1@@@
	// @@@STEP_STARTED@@@
	// @@@STEP_CLOSED@@@
	// @@@STEP_CURSOR step1@@@
	// @@@STEP_CLOSED@@@
	// @@@SEED_STEP step2@@@
	// @@@STEP_CURSOR step2@@@
	// @@@STEP_STARTED@@@
	// @@@SEED_STEP substep2@@@
	// @@@STEP_CURSOR substep2@@@
	// @@@STEP_NEST_LEVEL@1@@@
	// @@@STEP_STARTED@@@
	// @@@SEED_STEP substep3@@@
	// @@@STEP_CURSOR substep3@@@
	// @@@STEP_NEST_LEVEL@2@@@
	// @@@STEP_STARTED@@@
	// @@@STEP_CLOSED@@@
	// @@@STEP_CURSOR substep2@@@
	// @@@STEP_CLOSED@@@
	// @@@STEP_CURSOR step2@@@
	// @@@STEP_CLOSED@@@
}

func Example_labeled_log() {
	lg := NewLogger(os.Stdout)
	lg.LabeledLog("ijn", "ayanami")
	lg.LabeledLog("ijn", "jintsuu")
	// Output:
	// @@@STEP_LOG_LINE@ijn@ayanami@@@
	// @@@STEP_LOG_LINE@ijn@jintsuu@@@
}

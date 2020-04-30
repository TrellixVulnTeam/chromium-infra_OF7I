// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package event

func ExampleSend() {
	Send(Starting)
	Send(Completed)
	// Output:
	// starting
	// completed
}

func ExampleSendWithMsg() {
	SendWithMsg(Starting, "some message")
	// Output:
	// starting some message
}

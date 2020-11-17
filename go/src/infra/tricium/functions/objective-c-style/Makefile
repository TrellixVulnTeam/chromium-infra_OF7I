# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

.PHONY: test clean

objective-c-style:
	go build

test: objective-c-style
	./objective-c-style -input=test -output=out

clean:
	rm -rf out objective-c-style

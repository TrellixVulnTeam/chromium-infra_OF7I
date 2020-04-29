# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

.PHONY: test clean

https-check:
	go build

test: https-check
	./https-check -input=test -output=out

clean:
	rm -rf out https-check

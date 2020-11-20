# Copyright 2020 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.


.PHONY: build
build:
	go build -o eventdumper ./cmd

.PHONY: test
test:
	go test ./...

.PHONY: clean
clean:
	$(RM) ./eventdumper

.PHONY: format
format:
	gofmt -s -w .

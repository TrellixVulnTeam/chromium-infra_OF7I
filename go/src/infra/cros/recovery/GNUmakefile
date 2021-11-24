# Copyright 2021 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

GOTESTFLAGS ?= -count 20

.PHONY: default
default: build

.PHONY:
build:
	@gofmt -s -w .
	@go vet ./...
	@go test $(GOTESTFLAGS) ./...
	go build ./...

.PHONY:
test-slow:
	go test ./... $(GOTESTFLAGS) -p 1

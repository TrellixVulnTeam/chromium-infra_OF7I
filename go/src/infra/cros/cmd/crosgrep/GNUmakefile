# Copyright 2021 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# IENV is the infra environment wrapper script
# include an explicit path to the vpython interpreter because sudo cleans the PATH.
IENV    := $(realpath ../../../../../../depot_tools/vpython) $(realpath ../../../../env.py)

.PHONY: default
default: build

.PHONY:
build:
	$(IENV) go build -o ./crosgrep main.go

.PHONY: check
check:
	$(IENV) go vet ./...
	$(IENV) go test ./...

.PHONY: test
test:
	$(IENV) go test ./...


# clean just the binary artifacts
.PHONY: clean
clean:
	$(IENV) $(RM) ./crosgrep

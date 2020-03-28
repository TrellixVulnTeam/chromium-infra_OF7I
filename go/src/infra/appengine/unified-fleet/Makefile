# Copyright 2020 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
APP_PATH := $(PROJECT_PATH)
GCP_PROJECT_DEV = unified-fleet-system-dev

# Generate bindings
.PHONY: gen
gen:
	go generate ./...

# Run unittests
.PHONY: test
test:
	go test ./...

# Build and run service locally
.PHONY: dev
dev:
	go build -o ufs-service && ./ufs-service


# Uploading & Switching
.PHONY: up-dev
up-dev:
	gae.py upload -A $(GCP_PROJECT_DEV) --app-dir $(APP_PATH) -f
	gae.py switch -A $(GCP_PROJECT_DEV) --app-dir $(APP_PATH) -f

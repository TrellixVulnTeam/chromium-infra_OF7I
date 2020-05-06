# Copyright 2020 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
APP_PATH := $(PROJECT_PATH)
GCP_PROJECT_DEV = unified-fleet-system-dev

define help_message
Helpers for managing ufs project in $(PROJECT_PATH):
  gen: regenerate all go bindings
  test: run all unittests
  cover: generate code coverage report
  build: build the main ufs service
  build-dumper: build the dumper service
  dev: run the main ufs service locally
  dev-dumper: run the dumper service locally
  up-dev: upload the ufs service to dev appengine service
endef

.PHONY: help
help:
	$(info $(help_message))

# Generate bindings
.PHONY: gen
gen:
	go generate ./...

# Run unittests
.PHONY: test
test:
	go test ./... -coverprofile=coverage.out

# Check Code coverage
.PHONY: cover
cover:
	go tool cover -func=coverage.out

.PHONY: build
build:
	go build -o ufs-service $(PROJECT_PATH)/cmd/ufs-service

.PHONY: build-dumper
build-dumper:
	go build -o dumper $(PROJECT_PATH)/cmd/dumper

# Run service locally
.PHONY: dev
dev: build
	./ufs-service --cloud-project="unified-fleet-system-dev"

.PHONY: dev-dumper
dev-dumper: build-dumper
	./dumper --cloud-project="unified-fleet-system-dev"

# Uploading & Switching
.PHONY: up-dev
up-dev: build
	gae.py upload -A $(GCP_PROJECT_DEV) --app-dir $(APP_PATH) -f
	gae.py switch -A $(GCP_PROJECT_DEV) --app-dir $(APP_PATH) -f

# Copyright 2019 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
APP_PATH := $(PROJECT_PATH)/app/appengine
DEV_CFG_PATH := $(PROJECT_PATH)/devcfg/services/dev/

GCP_PROJECT = cros-lab-inventory
GCP_PROJECT_DEV = cros-lab-inventory-dev

.PHONY: default
default: vet test

.PHONY: vet
vet:
	go vet -all ./...

.PHONY: test
test:
	go test ./...

.PHONY: gen
gen:
	go generate ./...

# Development server

.PHONY: config
config:
	@printf ">>> Creating dev config at %s/%s.cfg. You may want to edit this for your needs.\n" $(DEV_CFG_PATH) $@
	cp $(DEV_CFG_PATH)/config-template.cfg $(DEV_CFG_PATH)/$@.cfg

.PHONY: dev
dev:
	gae.py devserver -A dev --app-dir $(APP_PATH) -- --host 0.0.0.0 --port 8082 --admin_port 7999 --log_level debug

# Uploading
.PHONY: up-dev
up-dev:
	gae.py upload -A $(GCP_PROJECT_DEV) --app-dir $(APP_PATH) -f
	gae.py switch -A $(GCP_PROJECT_DEV) --app-dir $(APP_PATH) -f

.PHONY: cleanup-dev
cleanup-dev:
	gae.py cleanup -A $(GCP_PROJECT_DEV) --app-dir $(APP_PATH)

# Copyright 2019 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
APP_PATH := $(PROJECT_PATH)/app/appengine
MR_APP_PATH := $(PROJECT_PATH)/app/ui/manual-repair
DEV_CFG_PATH := $(PROJECT_PATH)/devcfg/services/dev/
LIBS_PATH := $(PROJECT_PATH)/../../../cros/lab_inventory
GCP_PROJECT = cros-lab-inventory
GCP_PROJECT_DEV = cros-lab-inventory-dev

ifeq (${DEBUG}, 1)
	TESTOPTIONS =-v --parallel=1
endif

.PHONY: default
default: vet test

define help_message
Helpers for managing Chrome OS Lab Inventory project in $(PROJECT_PATH):
  vet: run 'go vet' on current directory
  test: run all Go unittests
  gen: regenerate all Go bindings

  config: create a dev config for local Go devserver
  dev: run the main Inventory v2 service locally

  up-dev: upload Inventory v2 service to dev appengine service and route traffic
  cleanup-dev: run gae.py cleanup for Inventory v2 on dev appengine
  up-prod: upload Inventory v2 service to prod appengine service without routing traffic
  switch-prod: switch Inventory v2 prod version and route traffic

  mr-setup: install all node_modules for Manual Repair App
  mr-dev: run Manual Repair App locally
  mr-deps: build dependencies needed for deployment
  mr-up-dev: upload Manual Repair App to dev appengine service and route traffic
  mr-up-prod: upload Manual Repair App to prod appengine service without routing traffic

  up-dev-all: upload all services to dev appengine and route traffic
  up-prod-all: upload all services to prod appengine without routing traffic
  switch-prod-all: switch Inventory v2 prod version and route traffic
endef

.PHONY: help
help:
	$(info $(help_message))

.PHONY: vet
vet:
	go vet -all ./...

.PHONY: test
test:
	go test ${TESTOPTIONS} -coverprofile=coverage.out ./... ${LIBS_PATH}/... || true
	go tool cover -html=coverage.out -o coverage.html

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

.PHONY: check-sync
check-sync:
	@printf "Please update the following luci path to ToT:\n"
	@printf "	infra/go/src/go.chromium.org/chromiumos/infra/proto\n"
	@printf "	infra/go/src/go.chromium.org/chromiumos/config\n"
	@printf "\n"
	@printf "Have you finished syncing? [y/N]" && read ans && [ $${ans:-N} = y ]

.PHONY: up-dev
up-dev: check-sync
	gae.py upload -A $(GCP_PROJECT_DEV) --app-dir $(APP_PATH) -f
	gae.py switch -A $(GCP_PROJECT_DEV) --app-dir $(APP_PATH) -f

.PHONY: cleanup-dev
cleanup-dev:
	gae.py cleanup -A $(GCP_PROJECT_DEV) --app-dir $(APP_PATH)

.PHONY: up-prod
up-prod: check-sync
	gae.py upload -A $(GCP_PROJECT) --app-dir $(APP_PATH)

.PHONY: switch-prod
switch-prod:
	gae.py switch -A $(GCP_PROJECT) --app-dir $(APP_PATH)

# Manual Repair App

.PHONY: mr-setup
mr-setup:
	cd $(MR_APP_PATH); \
	npm install -g typescript; \
	npm i

.PHONY: mr-dev
mr-dev:
	cd $(MR_APP_PATH); \
	npm run dev

.PHONY: mr-deps
mr-deps:
	cd $(MR_APP_PATH); \
	npm install --no-save

.PHONY: mr-up-dev
mr-up-dev: mr-deps
	cd $(MR_APP_PATH); \
	npm run build-stage; \
	gcloud app deploy --promote $(MR_APP_PATH)/app.stage.yaml \
	$(PROJECT_PATH)/app/dispatch.yaml --project $(GCP_PROJECT_DEV)

.PHONY: mr-up-prod
mr-up-prod: mr-deps
	cd $(MR_APP_PATH); \
	npm run build-prod; \
	gcloud app deploy --no-promote $(MR_APP_PATH)/app.prod.yaml \
	$(PROJECT_PATH)/app/dispatch.yaml --project $(GCP_PROJECT)

# Full Deployment

.PHONY: up-dev-all
up-dev-all:
	up-dev
	mr-up-dev

.PHONY: up-prod-all
up-prod-all:
	up-prod
	mr-up-prod

.PHONY: switch-prod-all
switch-prod-all:
	switch-prod

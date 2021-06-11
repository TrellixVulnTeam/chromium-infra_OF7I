# Copyright 2021 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# IENV is the infra environment wrapper script
# include an explicit path to the vpython interpreter because sudo cleans the PATH.
IENV    := $(realpath ../../../../../../depot_tools/vpython) $(realpath ../../../../env.py)

GAE_PY := ../../../../../luci/appengine/components/tools/gae.py

.PHONY: default
default: build

.PHONY:
build:
	$(IENV) go build -o ./cmd/karte/karte             ./cmd/karte
	$(IENV) go build -o ./cmd/karteserver/karteserver ./cmd/karteserver

.PHONY: check
check:
	$(IENV) go vet ./...
	$(IENV) go test ./...

.PHONY: test
test:
	$(IENV) go test ./...

.PHONY: gen
gen:
	$(IENV) go generate ./...

# clean just the binary artifacts
.PHONY: clean
clean:
	$(IENV) $(RM) ./cmd/karte/karte ./cmd/karteserver/karteserver ./karte ./karteserver

# Add a target for removing the generated files.
# It is easier to keep the generated files in sync if there's
# a straightforward way to remove them.
.PHONY: realclean
realclean: clean
	$(IENV) find . -name '*.pb.go' -delete
	$(IENV) $(RM) ./api/pb.discovery.go

.PHONY: run
run:
	$(IENV) ./karteserver -auth-service-host chrome-infra-auth-dev.appspot.com

.PHONY: deploy
deploy:
	$(IENV) $(GAE_PY) upload --app-id=chrome-fleet-karte-dev --app-dir=./cmd/karteserver

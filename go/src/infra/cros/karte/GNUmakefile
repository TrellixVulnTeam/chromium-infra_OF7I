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
	@go test -race $(GOTESTFLAGS) ./...
	go build -o ./cmd/karte/karte             ./cmd/karte
	go build -o ./cmd/karteserver/karteserver ./cmd/karteserver

.PHONY: check
check:
	go vet ./...
	go test ./...

.PHONY: test
test:
	go test ./...

.PHONY: gen
gen:
	go generate ./...

# clean just the binary artifacts
.PHONY: clean
clean:
	$(RM) ./cmd/karte/karte ./cmd/karteserver/karteserver ./karte ./karteserver

# Add a target for removing the generated files.
# It is easier to keep the generated files in sync if there's
# a straightforward way to remove them.
.PHONY: realclean
realclean: clean
	find . -name '*.pb.go' -delete
	$(RM) ./api/pb.discovery.go

.PHONY: run
run:
	./cmd/karteserver/karteserver -auth-service-host chrome-infra-auth-dev.appspot.com

# Intentionally use the gae.py version that is present on the path. If no such
# command is available on the path, you are not in the correct environment.
.PHONY: upload-dev
upload-dev:
	@echo BEGIN UPLOAD DEV
	gae.py upload --app-id=chrome-fleet-karte-dev --app-dir=./cmd/karteserver
	@echo END UPLOAD DEV

.PHONY: switch-dev
switch-dev:
	@echo BEGIN SWITCH DEV
	gae.py switch --app-id=chrome-fleet-karte-dev --app-dir=./cmd/karteserver
	@echo END SWITCH DEV

.PHONY: upload-switch-dev
upload-switch-dev:
	gae.py upload --app-id=chrome-fleet-karte-dev --app-dir=./cmd/karteserver --force --verbose
	gae.py switch --app-id=chrome-fleet-karte-dev --app-dir=./cmd/karteserver --force --verbose

# Intentionally use the gae.py version that is present on the path. If no such
# command is available on the path, you are not in the correct environment.
.PHONY: upload-prod
upload-prod:
	@echo BEGIN UPLOAD PROD
	gae.py upload --app-id=chrome-fleet-karte --app-dir=./cmd/karteserver
	@echo END UPLOAD PROD
	@echo "If you did NOT intend to push prod, do NOT call \"switch prod\""

.PHONY: switch-prod
switch-prod:
	@echo BEGIN SWITCH PROD
	gae.py switch --app-id=chrome-fleet-karte --app-dir=./cmd/karteserver
	@echo END SWITCH PROD

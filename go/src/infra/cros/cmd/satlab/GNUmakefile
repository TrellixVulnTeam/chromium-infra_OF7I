# Copyright 2021 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

.PHONY: satlab
satlab:
	@if test -f ./devscripts/pre-satlab 2>/dev/null; then ./devscripts/pre-satlab; fi
	go build -o satlab -- .
	@if test -f ./devscripts/post-satlab 2>/dev/null; then ./devscripts/post-satlab; fi

.PHONY: test
test:
	go test ./...

.PHONY: check
check: test

.PHONY: clean
clean:
	$(RM) ./satlab

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...

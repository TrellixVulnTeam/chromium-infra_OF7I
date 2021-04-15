# Copyright 2021 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

default: check

check:
	golint ./...
	go vet ./...
	go test ./...

test:
	go test ./...

gen:
	go generate ./...

# Add a target for removing the generated files.
# It is easier to keep the generated files in sync if there's
# a straightforward way to remove them.
.PHONY: realclean
realclean:
	find . -name '*.pb.go' -delete
	$(RM) ./api/pb.discovery.go

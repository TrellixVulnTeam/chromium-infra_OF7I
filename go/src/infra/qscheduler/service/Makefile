# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
projdir := $(patsubst %/,%,$(dir $(mkfile_path)))
infradir := $(projdir)/../../../../..

test:
	go test ./...

gen:
	go generate ./...

run-local:
	go run $(projdir)/cmd/qscheduler-swarming/main.go \
		-qscheduler-config $(projdir)/devcfg/services/dev/config.cfg \
		-cloud-project qscheduler-swarming-dev \
		-http-addr "0.0.0.0:8082" \
		-admin-addr "0.0.0.0:8900"

# This builds "gcr.io/chops-public-images-dev/qscheduler-swarming" from local
# code in the same way the production builders build production images. Can
# be used to verify changes to Dockefile or qscheduler-swarming.yaml.
build-docker-image:
	cloudbuildhelper build \
		$(infradir)/build/images/deterministic/qscheduler-swarming.yaml

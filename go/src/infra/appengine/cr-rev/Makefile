# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Before running any targets, ensure you run the following command:
# eval `../../../../env.py`

# Generate all go proto+grpc
.PHONY: gen
gen:
	cproto config
	cproto frontend/api/v1
	protoc backend/pubsub/source_repo_event.proto --go_out=../../../

.PHONY: test
test: gen
	go test ./...

# Deploy to development
.PHONY: dev
dev: gen
	gae.py upload --app-id cr-rev-dev  --app-dir .

# Deploy to production
.PHONY: prod
prod: prod
	gae.py upload --app-id cr-rev  --app-dir .


# Copyright 2022 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Before running any targets, ensure you run the following command:
# eval `../../../../env.py`

# Generate all go proto+grpc
.PHONY: gen
gen:
	cproto schema

.PHONY: test
test:
	go test ./...

# Deploy to production
.PHONY: prod
prod:
	gae.py upload --app-id cit-cli-metrics  --app-dir .


# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Several commands here assume that the Go environment is set up,
# e.g. by running `eval $(infra/go/env.py)`.

.PHONY: default help frontend rpcexplorer deploy_prod deploy_dev

default: help

help:
	@echo "Available commands:"
	@sed -n '/^[a-zA-Z0-9_]*:/s/:.*//p' < Makefile

frontend:
	cd frontend/ui && make build

# Building rpcexplorer is optional.
# This appears broken now, see crbug.com/1042421.
rpcexplorer:
	cd ../../../go.chromium.org/luci/web && ./web.py build rpcexplorer

deploy_prod: frontend
	gae.py upload --switch --app-id=tricium-prod

deploy_dev: frontend
	gae.py upload --switch --app-id=tricium-dev

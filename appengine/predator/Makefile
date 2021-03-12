# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

PREDATOR_STAGING = predator-for-me-staging
PREDATOR_PROD = predator-for-me

MODULES= app/app.yaml \
	 app/backend-clusterfuzz.yaml \
	 app/backend-cracas.yaml \
	 app/backend-fracas.yaml \
	 app/backend-process.yaml \
	 app/backend-uma-sampling-profiler.yaml

default: help

help:
	@echo "Available commands:"
	@sed -n '/^[a-zA-Z0-9_.]*:/s/:.*//p' <Makefile

test:
	../../test.py test appengine/predator

deploy_staging:
	gcloud app deploy --project ${PREDATOR_STAGING} ${MODULES}

deploy_prod:
	gcloud app deploy --project ${PREDATOR_PROD} ${MODULES}

# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

.PHONY: build-for-deployment test clean

build-for-deployment: python pylint
	GOOS=linux GOARCH=amd64 go build -o pylint_parser

pylint_parser: python pylint
	go build -o pylint_parser

test: pylint_parser
	./pylint_parser -input=test -output=out

python:
	echo "infra/python/cpython3/linux-amd64 version:3.8.0b1.chromium.1" | \
		cipd ensure -ensure-file - -root python

pylint:
	pip install --ignore-installed --target pylint --no-compile pylint

clean:
	rm -rf pylint_parser pylint python out

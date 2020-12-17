# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

.PHONY: build-for-deployment test clean

build-for-deployment: gosec
	GOOS=linux GOARCH=amd64 go build -o gosec_wrapper

gosec_wrapper: gosec
	go build -o gosec_wrapper

gosec:
	if [ ! -d ./bin ] ; then mkdir bin && curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s latest; fi

test: gosec_wrapper
	./gosec_wrapper -input=test -output=out && go test || echo "TEST FAILED"

clean:
	rm -rf bin out gosec_wrapper

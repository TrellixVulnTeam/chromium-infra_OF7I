
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
.PHONY: build-for-deploy test unit-test clean

# Make sure that the deployed binary is runnable on Linux, even if deploying from a different platform.
build-for-deploy:
	GOOS=linux GOARCH=amd64 go build -o metrics_analyzer

metrics-analyzer:
	go build -o metrics_analyzer

test: metrics-analyzer
	./metrics_analyzer -input=testdata/src -output=out -previous=prevdata/src -patch=prevdata/tricium_generated_diff.patch -enums=enums/enums.xml -- rm/remove_histogram.xml

unit-test:
	go test histograms_check_test.go field_trial_check_test.go metrics_analyzer.go histograms_check.go field_trial_check.go

clean:
	rm -rf out metrics_analyzer

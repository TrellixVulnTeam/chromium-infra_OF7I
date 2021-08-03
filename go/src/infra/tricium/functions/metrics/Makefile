# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

.PHONY: build-for-deployment test unit-test clean

build-for-deployment:
	GOOS=linux GOARCH=amd64 go build -o metrics_analyzer

metrics_analyzer:
	go build -o metrics_analyzer

clean:
	rm -rf out metrics_analyzer

test: clean metrics_analyzer
	./metrics_analyzer -input=testdata/src -output=out -previous=prevdata/src -patch=prevdata/tricium_test_diff.patch -enums=enums/enums.xml -- test/histograms.xml
	./metrics_analyzer -input=testdata/src -output=out -previous=prevdata/src -patch=prevdata/add_new_suffix_diff.patch -enums=enums/enums.xml -- suffixes/histogram_suffixes_list.xml

unit-test:
	go test histograms_check_test.go field_trial_check_test.go metrics_analyzer.go histograms_check.go field_trial_check.go


# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
.PHONY: test clean

metrics-analyzer:
	go build -o metrics_analyzer

test: metrics-analyzer
	./metrics_analyzer -input=testdata -output=out -enums=src/enums/enums.xml

unit-test:
	go test histograms_check_test.go field_trial_check_test.go metrics_analyzer.go histograms_check.go field_trial_check.go

test-arg: metrics-analyzer
	./metrics_analyzer -input=testdata -output=out -previous=prevdata -patch=prevdata/tricium_generated_diff.patch -enums=src/enums/enums.xml -- src/rm/remove_histogram.xml
	
clean:
	rm -rf out metrics_analyzer
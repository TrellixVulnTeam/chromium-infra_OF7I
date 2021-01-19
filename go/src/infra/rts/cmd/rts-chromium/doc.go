// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command rts-chromium is Chromium-specific part of the generic RTS framework.
//
// Install it:
//
//  go install infra/rts/cmd/rts-chromium
//
// Primarily rts-chromium can generate history files:
//
//  rts-chromium presubmit-history \
//    -from 2020-10-04 -to 2020-10-05 \
//    -out cq.hist
//
// It will ask to login on the first run.
//
// Filtering
//
// Flags -builder and -test can be used to narrow the search down to specific
// builder and/or test. The flag values are regexps. The following retrieves
// history of browser_tests on linux-rel:
//
//  rts-chromium presubmit-history \
//    -from 2020-10-04 -to 2020-10-05 \
//    -out linux-rel-browser-tests.hist \
//    -builder linux-rel \
//    -test ninja://chrome/test:browser_tests/.+
//
// Test duration fraction
//
// By default the tool fetches only 0.1% of test durations, because
// Chromium CQ produces 1B+ of test results per day. Fetching them all would be
// excessive.
//
// However, if the date range is short and/or filters are applied, the
// default fraction of 0.1% might be inappropriate. It can be changed using
// -duration-data-frac flag. The following changes the sample size to 10%:
//
//  rts-chromium presubmit-history \
//    -from 2020-10-04 -to 2020-10-05 \
//    -out cq.hist
//    -duration-data-frac 0.1
package main

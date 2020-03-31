# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd


def CheckChange(input_api, output_api):
  results = []
  results += input_api.canned_checks.CheckDoNotSubmit(input_api, output_api)
  results += input_api.canned_checks.CheckChangeHasNoTabs(input_api, output_api)
  return results


def CheckChangeOnUpload(input_api, output_api):
  return CheckChange(input_api, output_api)


def CheckChangeOnCommit(input_api, output_api):
  return CheckChange(input_api, output_api)

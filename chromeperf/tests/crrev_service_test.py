# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json
import unittest

from chromeperf.services import crrev_service


def test_CrrevService_GetNumbering(mocker):
  mock_request = mocker.patch('chromeperf.services.request.request')
  params = {
      'number': '498032',
      'numbering_identifier': 'refs/heads/master',
      'numbering_type': 'COMMIT_POSITION',
      'project': 'chromium',
      'repo': 'chromium/src'
  }
  return_value = {
      'git_sha': '4c9925b198332f5fbb82b3edb672ed55071f87dd',
      'repo': 'chromium/src',
      'numbering_type': 'COMMIT_POSITION',
      'number': '498032',
      'project': 'chromium',
      'numbering_identifier': 'refs/heads/master',
      'redirect_url': 'https://chromium.googlesource.com/chromium/src/+/foo',
      'kind': 'crrev#numberingItem',
      'etag': '"z28iYHtWcY14RRFEUgin0OFGLHY/au8p5YtferYwojQRpsPavK6G5-A"'
  }
  mock_request.return_value = json.dumps(return_value)
  assert crrev_service.get_numbering(**params) == return_value
  mock_request.assert_called_once_with(
      'https://cr-rev.appspot.com/_ah/api/crrev/v1/get_numbering',
      'GET',
      project='chromium',
      repo='chromium/src',
      number='498032',
      numbering_type='COMMIT_POSITION',
      numbering_identifier='refs/heads/master')

# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import json
import pytest

from chromeperf.services import buildbucket_service

_BUILD_PARAMETERS = {
    'builder_name': 'dummy_builder',
    'properties': {
        'bisect_config': {}
    }
}


@pytest.fixture
def request_json(mocker):
  mocked = mocker.patch('chromeperf.services.request.request_json',
                        mocker.MagicMock())
  mocked.return_value = {'build': {'id': 'build id'}}
  return mocked


def test_BuildBucketService_Put(request_json):
  expected_body = {
      'bucket': 'bucket_name',
      'tags': ['buildset:foo'],
      'parameters_json': json.dumps(_BUILD_PARAMETERS, separators=(',', ':')),
  }
  response = buildbucket_service.put('bucket_name', ['buildset:foo'],
                                     _BUILD_PARAMETERS)
  assert response == {'build': {'id': 'build id'}}
  request_json.assert_called_once_with(
      buildbucket_service.API_BASE_URL + 'builds',
      method='PUT',
      body=expected_body)


def test_BuildBucketService_GetJobStatus(request_json):
  response = buildbucket_service.get('job_id')
  assert response == {'build': {'id': 'build id'}}
  request_json.assert_called_once_with(buildbucket_service.API_BASE_URL +
                                       'builds/job_id')

# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json
import mock

from testing_utils import testing
from libs.http import retry_http_client

from common import rotations


class DummyHttpClient(retry_http_client.RetryHttpClient):

  def __init__(self, *args, **kwargs):
    super(DummyHttpClient, self).__init__(*args, **kwargs)
    self.responses = {}
    self.requests = []

  def SetResponse(self, url, result):
    self.responses[url] = result

  def GetBackoff(self, *_):  # pragma: no cover
    """Override to avoid sleep."""
    return 0

  def _Get(self, url, _, headers):
    self.requests.append((url, None, headers))
    response = self.responses.get(url, (404, 'Not Found'))
    return response[0], response[1], {}

  def _Post(self, *_):  # pragma: no cover
    pass

  def _Put(self, *_):  # pragma: no cover
    pass


class RotationsTest(testing.AppengineTestCase):

  def setUp(self):
    super(RotationsTest, self).setUp()
    self.http_client = DummyHttpClient()
    self.http_patcher = mock.patch('common.rotations._HTTP_CLIENT',
                                   self.http_client)
    self.http_patcher.start()

  def tearDown(self):
    self.http_patcher.stop()

  def testCurrentSheriffs(self):
    response = json.dumps({
        'updated_unix_timestamp': 1587957062,
        'emails': ['ham@google.com', 'beef@google.com'],
    })
    self.http_client.SetResponse(rotations._ROTATIONS_URL, (200, response))
    self.assertIn('ham@google.com', rotations.current_sheriffs())
    self.assertIn('beef@google.com', rotations.current_sheriffs())

  def testCurrentSheriffsMalformedResponse(self):
    response = json.dumps({
        'foo': 'bar',
    })
    self.http_client.SetResponse(rotations._ROTATIONS_URL, (200, response))
    with self.assertRaises(Exception):
      rotations.current_sheriffs()

  def testCurrentSheriffsBadHttp(self):
    self.http_client.SetResponse(rotations._ROTATIONS_URL, (403, 'forbidden'))
    with self.assertRaises(Exception):
      rotations.current_sheriffs()

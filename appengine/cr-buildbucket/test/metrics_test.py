# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import datetime

import mock
import gae_ts_mon

from test import test_util
from test.test_util import future
from testing_utils import testing
from go.chromium.org.luci.buildbucket.proto import common_pb2
import config
import metrics
import model


class MetricsTest(testing.AppengineTestCase):

  def setUp(self):
    super(MetricsTest, self).setUp()
    gae_ts_mon.reset_for_unittest(disable=True)

  def test_fields_for(self):
    build = test_util.build(
        builder=dict(project='chromium', bucket='try', builder='linux'),
        status=common_pb2.FAILURE,
        tags=[
            dict(key='user_agent', value='cq'),
            dict(key='something', value='else'),
        ],
        canary=common_pb2.YES,
    )
    expected = {
        'bucket': 'luci.chromium.try',
        'builder': 'linux',
        'canary': True,
        'user_agent': 'cq',
        'status': 'COMPLETED',
        'result': 'FAILURE',
        'failure_reason': 'BUILD_FAILURE',
        'cancelation_reason': '',
    }
    self.assertEqual(set(expected), set(metrics._BUILD_FIELDS))
    actual = metrics._fields_for(build, expected.keys())
    self.assertEqual(expected, actual)

    with self.assertRaises(ValueError):
      metrics._fields_for(build, ['wrong field'])

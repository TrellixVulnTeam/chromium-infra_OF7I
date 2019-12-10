# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock

import endpoints
import gae_ts_mon

from .test_support import test_case

from infra_libs.ts_mon import config
from infra_libs.ts_mon.common import http_metrics
from protorpc import message_types
from protorpc import remote


class FakeTime(object):

  def __init__(self):
    self.timestamp_now = 1000.0

  def __call__(self):
    self.timestamp_now += 0.2
    return self.timestamp_now


@endpoints.api(name='testapi', version='v1')
class TestEndpoint(remote.Service):

  @gae_ts_mon.instrument_endpoint(time_fn=FakeTime())
  @endpoints.method(
      message_types.VoidMessage, message_types.VoidMessage, name='method_good')
  def do_good(self, request):
    return request

  @gae_ts_mon.instrument_endpoint(time_fn=FakeTime())
  @endpoints.method(
      message_types.VoidMessage, message_types.VoidMessage, name='method_bad')
  def do_bad(self, _):
    raise Exception

  @gae_ts_mon.instrument_endpoint(time_fn=FakeTime())
  @endpoints.method(
      message_types.VoidMessage, message_types.VoidMessage, name='method_400')
  def do_400(self, _):
    raise endpoints.BadRequestException('Bad request')


class InstrumentEndpointTest(test_case.EndpointsTestCase):
  api_service_cls = TestEndpoint

  def setUp(self):
    super(InstrumentEndpointTest, self).setUp()
    self.endpoint_name = '/_ah/spi/TestEndpoint.%s'
    config.reset_for_unittest()

  def test_good(self):
    self.call_api('do_good')
    fields = {
        'name': self.endpoint_name % 'method_good',
        'status': 200,
        'is_robot': False
    }
    self.assertEqual(1, http_metrics.server_response_status.get(fields))
    self.assertLessEqual(200, http_metrics.server_durations.get(fields).sum)

  def test_bad(self):
    self.call_api('do_bad', status=500)
    fields = {
        'name': self.endpoint_name % 'method_bad',
        'status': 500,
        'is_robot': False
    }
    self.assertEqual(1, http_metrics.server_response_status.get(fields))
    self.assertLessEqual(200, http_metrics.server_durations.get(fields).sum)

  def test_400(self):
    self.call_api('do_400', status=400)
    fields = {
        'name': self.endpoint_name % 'method_400',
        'status': 400,
        'is_robot': False
    }
    self.assertEqual(1, http_metrics.server_response_status.get(fields))
    self.assertLessEqual(200, http_metrics.server_durations.get(fields).sum)

  @mock.patch(
      'gae_ts_mon.exporter.need_to_flush_metrics',
      autospec=True,
      return_value=False)
  def test_no_flush(self, _fake):
    # For branch coverage.
    self.call_api('do_good')
    fields = {
        'name': self.endpoint_name % 'method_good',
        'status': 200,
        'is_robot': False
    }
    self.assertEqual(1, http_metrics.server_response_status.get(fields))
    self.assertLessEqual(200, http_metrics.server_durations.get(fields).sum)

# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import copy
import os

import gae_ts_mon
import mock
import webapp2

from .test_support import test_case

from infra_libs.ts_mon import config
from infra_libs.ts_mon import shared
from infra_libs.ts_mon.common import http_metrics
from infra_libs.ts_mon.common import interface
from infra_libs.ts_mon.common import monitors
from infra_libs.ts_mon.common import targets


class InitializeTest(test_case.TestCase):
  def setUp(self):
    super(InitializeTest, self).setUp()

    config.reset_for_unittest()
    target = targets.TaskTarget('test_service', 'test_job',
                                'test_region', 'test_host')
    self.mock_state = interface.State(target=target)
    self.mock_state.metrics = copy.copy(interface.state.metrics)
    mock.patch('infra_libs.ts_mon.common.interface.state',
        new=self.mock_state).start()

    mock.patch('infra_libs.ts_mon.common.monitors.HttpsMonitor',
               autospec=True).start()

  def tearDown(self):
    config.reset_for_unittest()
    mock.patch.stopall()
    super(InitializeTest, self).tearDown()

  def test_sets_target(self):
    config.initialize(is_local_unittest=False)

    self.assertEqual('sample-app', self.mock_state.target.service_name)
    self.assertEqual('default', self.mock_state.target.job_name)
    self.assertEqual('appengine', self.mock_state.target.region)
    self.assertEqual('v1a', self.mock_state.target.hostname)

  def test_sets_monitor(self):
    os.environ['SERVER_SOFTWARE'] = 'Production'  # != 'Development'
    config.initialize(is_local_unittest=False)
    self.assertEquals(1, monitors.HttpsMonitor.call_count)

  def test_sets_monitor_dev(self):
    config.initialize(is_local_unittest=False)
    self.assertFalse(monitors.HttpsMonitor.called)
    self.assertIsInstance(self.mock_state.global_monitor, monitors.DebugMonitor)

  def test_initialize_with_enabled_fn(self):
    is_enabled_fn = mock.Mock()
    config.initialize(
        None, is_enabled_fn=is_enabled_fn, is_local_unittest=False)
    self.assertIs(is_enabled_fn, interface.state.flush_enabled_fn)

  @mock.patch('gae_ts_mon.config.instrument_wsgi_application')
  def test_initialize_with_local_unittest(self, mock_inst):
    config.initialize(object(), is_local_unittest=True)
    mock_inst.assert_called()

  @mock.patch('gae_ts_mon.exporter.flush_metrics_if_needed', return_value=True)
  def test_shutdown_hook_flushed(self, _mock_flush):
    time_now = 10000
    id = shared.get_instance_entity().key.id()
    with shared.instance_namespace_context():
      self.assertIsNotNone(shared.Instance.get_by_id(id))
    config._shutdown_hook(time_fn=lambda: time_now)
    with shared.instance_namespace_context():
      self.assertIsNone(shared.Instance.get_by_id(id))

  @mock.patch('gae_ts_mon.exporter.flush_metrics_if_needed', return_value=False)
  def test_shutdown_hook_not_flushed(self, _mock_flush):
    time_now = 10000
    id = shared.get_instance_entity().key.id()
    with shared.instance_namespace_context():
      self.assertIsNotNone(shared.Instance.get_by_id(id))
    config._shutdown_hook(time_fn=lambda: time_now)
    with shared.instance_namespace_context():
      self.assertIsNone(shared.Instance.get_by_id(id))

  def test_internal_callback(self):
    # Smoke test.
    config._internal_callback()


class InstrumentWSGIApplicationTest(test_case.TestCase):

  def setUp(self):
    super(InstrumentWSGIApplicationTest, self).setUp()

  @mock.patch('gae_ts_mon.instrument_webapp2.instrument')
  def testWithWebapp2(self, mock_inst):
    app = webapp2.WSGIApplication()
    config.instrument_wsgi_application(app, time_fn=None)
    mock_inst.assert_called_once_with(app, None)

  def testWithUnsupportedWSGIApp(self):
    with self.assertRaises(NotImplementedError):
      config.instrument_wsgi_application(object())

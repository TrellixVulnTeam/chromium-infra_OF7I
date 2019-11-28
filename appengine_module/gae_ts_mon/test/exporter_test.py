# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import copy
import datetime

import gae_ts_mon
import mock

from .test_support import test_case

from infra_libs.ts_mon import config
from infra_libs.ts_mon import exporter
from infra_libs.ts_mon import shared
from infra_libs.ts_mon.common import interface
from infra_libs.ts_mon.common import targets


class ExporterTest(test_case.TestCase):

  def setUp(self):
    super(ExporterTest, self).setUp()

    config.reset_for_unittest()
    target = targets.TaskTarget('test_service', 'test_job', 'test_region',
                                'test_host')
    self.mock_state = interface.State(target=target)
    self.mock_state.metrics = copy.copy(interface.state.metrics)
    mock.patch(
        'infra_libs.ts_mon.common.interface.state',
        new=self.mock_state).start()

    mock.patch(
        'infra_libs.ts_mon.common.monitors.HttpsMonitor',
        autospec=True).start()

  def tearDown(self):
    config.reset_for_unittest()
    mock.patch.stopall()
    super(ExporterTest, self).tearDown()

  def test_reset_cumulative_metrics(self):
    gauge = gae_ts_mon.GaugeMetric('gauge', 'foo', None)
    counter = gae_ts_mon.CounterMetric('counter', 'foo', None)
    gauge.set(5)
    counter.increment()
    self.assertEqual(5, gauge.get())
    self.assertEqual(1, counter.get())

    exporter._reset_cumulative_metrics()
    self.assertEqual(5, gauge.get())
    self.assertIsNone(counter.get())

  def test_flush_metrics_without_target(self):
    time_now = 10000
    interface.state.target = None
    self.assertFalse(exporter.flush_metrics_if_needed(time_now))

  def test_flush_metrics_no_task_num(self):
    # We are not assigned task_num yet; cannot send metrics.
    time_now = 10000
    datetime_now = datetime.datetime.utcfromtimestamp(time_now)
    more_than_min_ago = datetime_now - datetime.timedelta(seconds=61)
    interface.state.last_flushed = more_than_min_ago
    entity = shared.get_instance_entity()
    entity.task_num = -1
    interface.state.target.task_num = -1
    self.assertFalse(exporter.flush_metrics_if_needed(time_now))

  def test_flush_metrics_no_task_num_too_long(self):
    # We are not assigned task_num for too long; cannot send metrics.
    time_now = 10000
    datetime_now = datetime.datetime.utcfromtimestamp(time_now)
    too_long_ago = datetime_now - datetime.timedelta(
        seconds=shared.INSTANCE_EXPECTED_TO_HAVE_TASK_NUM_SEC + 1)
    interface.state.last_flushed = too_long_ago
    entity = shared.get_instance_entity()
    entity.task_num = -1
    entity.last_updated = too_long_ago
    interface.state.target.task_num = -1
    self.assertFalse(exporter.flush_metrics_if_needed(time_now))

  def test_flush_metrics_purged(self):
    # We lost our task_num; cannot send metrics.
    time_now = 10000
    datetime_now = datetime.datetime.utcfromtimestamp(time_now)
    more_than_min_ago = datetime_now - datetime.timedelta(seconds=61)
    interface.state.last_flushed = more_than_min_ago
    entity = shared.get_instance_entity()
    entity.task_num = -1
    interface.state.target.task_num = 2
    self.assertFalse(exporter.flush_metrics_if_needed(time_now))

  def test_flush_metrics_too_early(self):
    # Too early to send metrics.
    time_now = 10000
    datetime_now = datetime.datetime.utcfromtimestamp(time_now)
    less_than_min_ago = datetime_now - datetime.timedelta(seconds=59)
    interface.state.last_flushed = less_than_min_ago
    entity = shared.get_instance_entity()
    entity.task_num = 2
    self.assertFalse(exporter.flush_metrics_if_needed(time_now))

  @mock.patch('infra_libs.ts_mon.common.interface.flush', autospec=True)
  def test_flush_metrics_successfully(self, mock_flush):
    # We have task_num and due for sending metrics.
    time_now = 10000
    datetime_now = datetime.datetime.utcfromtimestamp(time_now)
    more_than_min_ago = datetime_now - datetime.timedelta(seconds=61)
    interface.state.last_flushed = more_than_min_ago
    entity = shared.get_instance_entity()
    entity.task_num = 2
    # Global metrics must be erased after exporter.
    test_global_metric = gae_ts_mon.GaugeMetric('test', 'foo', None)
    test_global_metric.set(42)
    interface.register_global_metrics([test_global_metric])
    self.assertEqual(42, test_global_metric.get())
    self.assertTrue(exporter.flush_metrics_if_needed(time_now))
    self.assertEqual(None, test_global_metric.get())
    mock_flush.assert_called_once_with()

  @mock.patch('infra_libs.ts_mon.common.interface.flush', autospec=True)
  def test_flush_metrics_disabled(self, mock_flush):
    # We have task_num and due for sending metrics, but ts_mon is disabled.
    time_now = 10000
    datetime_now = datetime.datetime.utcfromtimestamp(time_now)
    more_than_min_ago = datetime_now - datetime.timedelta(seconds=61)
    interface.state.last_flushed = more_than_min_ago
    interface.state.flush_enabled_fn = lambda: False
    entity = shared.get_instance_entity()
    entity.task_num = 2
    self.assertFalse(exporter.flush_metrics_if_needed(time_now))
    self.assertEqual(0, mock_flush.call_count)

  @mock.patch('threading.Thread', autospec=True)
  @mock.patch(
      'gae_ts_mon.exporter.need_to_flush_metrics',
      autospec=True,
      return_value=True)
  def test_parallel_flush_start_thread(self, _, mock_thread):
    time_now = 10000
    with exporter.parallel_flush(time_now) as thread:
      self.assertNotEqual(None, thread)
      # test if the thread instance was created with the expected param values.
      mock_thread.assert_called_once_with(
          args=(time_now,), target=exporter._flush_metrics)
      # test if thread.start() has been invoked.
      thread.start.assert_called_once()

    # thread.join() should be called when the context exits.
    thread.join.assert_called_once()

  @mock.patch('threading.Thread', autospec=True)
  @mock.patch(
      'gae_ts_mon.exporter.need_to_flush_metrics',
      autospec=True,
      return_value=False)
  def test_create_flush_thread_if_needed_skip_creating_thread(
      self, _, mock_thread):
    time_now = 10000
    with exporter.parallel_flush(time_now) as thread:
      self.assertEqual(None, thread)
      mock_thread.assert_not_called()

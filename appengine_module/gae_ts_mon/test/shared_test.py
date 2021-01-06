# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from __future__ import absolute_import
import unittest
import os
import mock

import gae_ts_mon

from .test_support import test_case

from infra_libs.ts_mon import shared


class SharedTest(test_case.TestCase):

  @mock.patch('gae_ts_mon.shared.is_python3_env', return_value=False)
  def test_get_instance_entity(self, _mocked_py3_env):
    entity = shared.get_instance_entity()
    # Save the modification, make sure it sticks.
    entity.task_num = 42
    entity.put()
    entity2 = shared.get_instance_entity()
    self.assertEqual(42, entity2.task_num)

    # Make sure it does not pollute the default namespace.
    self.assertIsNone(shared.Instance.get_by_id(entity.key.id()))

  @mock.patch('gae_ts_mon.shared.is_python3_env', return_value=True)
  def test_instance_key_id_py3(self, _mocked_py3_env):
    with mock.patch.dict(
        'os.environ', {
            'GAE_INSTANCE': 'instance',
            'GAE_VERSION': 'version',
            'GAE_SERVICE': 'default',
        }):
      self.assertEqual(shared.instance_key_id(), 'instance.version.default')

  def test_is_python_3_env(self):
    with mock.patch.dict('os.environ', {
        'GAE_RUNTIME': 'python3',
    }):
      self.assertEqual(shared.is_python3_env(), True)

    with mock.patch.dict('os.environ', {
        'GAE_RUNTIME': 'python37',
    }):
      self.assertEqual(shared.is_python3_env(), True)

    with mock.patch.dict('os.environ', {
        'GAE_RUNTIME': 'python27',
    }):
      self.assertEqual(shared.is_python3_env(), False)

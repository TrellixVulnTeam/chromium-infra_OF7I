# Copyright (c) 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import logging
import os
import shutil
import sys
import tempfile
import unittest

import mock

from infra.services.service_manager import cloudtail_factory


class CloudtailFactoryTest(unittest.TestCase):
  def setUp(self):
    self.tmpdir = tempfile.mkdtemp()
    mock.patch(
        'infra_libs.logs.logs.DEFAULT_LOG_DIRECTORIES', self.tmpdir).start()
    self.mock_popen = mock.patch('subprocess.Popen', autospec=True).start()

  def tearDown(self):
    mock.patch.stopall()
    try:
      shutil.rmtree(self.tmpdir)
    except OSError:  # pragma: no cover
      # Don't raise, as this may prevent from seeing actual test failure.
      logging.exception('failed to rmtree(%r)', self.tmpdir)

  def test_start(self):
    fh = mock.Mock()

    f = cloudtail_factory.CloudtailFactory('/foo', None)
    f.start('log', fh)

    self.assertEqual(1, self.mock_popen.call_count)
    self.assertEqual(
        ['/foo', 'pipe', '--log-id', 'log', '--local-log-level', 'info'],
        self.mock_popen.call_args[0][0])

    kwargs = self.mock_popen.call_args[1]
    self.assertEqual(fh, kwargs['stdin'])
    self.assertTrue(kwargs['stdout'].closed)
    self.assertTrue(kwargs['stderr'].closed)

  def test_start_avoids_log_collisions(self):
    f = cloudtail_factory.CloudtailFactory('/foo', None)
    f.start('invoke1', mock.Mock())
    f.start('invoke2', mock.Mock())
    self.assertEqual(2, self.mock_popen.call_count)
    self.assertEqual(2, f._counter)

    kwargs1 = self.mock_popen.call_args_list[0][1]
    kwargs2 = self.mock_popen.call_args_list[1][1]
    logging.debug('%s\n\n%s\n\n%s', self.mock_popen.call_args_list,
        self.mock_popen.call_args_list[0], kwargs1)
    base = os.path.join(self.tmpdir, 'sm.%d.cloudtail.' % os.getpid())
    self.assertEqual(kwargs1['stdout'].name, base + '1.stdout')
    self.assertEqual(kwargs1['stderr'].name, base + '1.stderr')
    self.assertEqual(kwargs2['stdout'].name, base + '2.stdout')
    self.assertEqual(kwargs2['stderr'].name, base + '2.stderr')

  def test_start_with_credentials(self):
    f = cloudtail_factory.CloudtailFactory('/foo', '/bar')
    f.start('log', mock.Mock())

    self.assertEqual(1, self.mock_popen.call_count)
    self.assertEqual(
        ['/foo', 'pipe', '--log-id', 'log', '--local-log-level', 'info',
         '--ts-mon-credentials', '/bar'],
        self.mock_popen.call_args[0][0])

  def test_start_with_kwargs(self):
    f = cloudtail_factory.CloudtailFactory('/foo', None)
    f.start('log', mock.Mock(), cwd='bar')

    self.assertEqual(1, self.mock_popen.call_count)
    kwargs = self.mock_popen.call_args[1]
    self.assertEqual('bar', kwargs['cwd'])

  def test_choose_log_dir_default_works(self):
    candidates = [os.path.join(self.tmpdir, 'not-exists')]

    if not sys.platform.startswith('win'):  # pragma: no branch cover
      # It's non-trivial to create a read-only (in Unix sense) dir on Windows.
      # https://superuser.com/a/1247851
      # So, just don't do this windows.
      read_only = os.path.join(self.tmpdir, 'ro')
      os.mkdir(read_only, 0444)
      candidates.append(read_only)

    writable = os.path.join(self.tmpdir, 'w')
    os.mkdir(writable, 0777)
    candidates.append(writable)

    with mock.patch('infra_libs.logs.logs.DEFAULT_LOG_DIRECTORIES',
                    os.pathsep.join(candidates)):
      self.assertEqual(cloudtail_factory._choose_log_dir(), writable)

  def test_choose_log_dir_failsafe_to_temp(self):
    not_exists = os.path.join(self.tmpdir, 'not-exists')
    with mock.patch('infra_libs.logs.logs.DEFAULT_LOG_DIRECTORIES', not_exists):
      with mock.patch('tempfile.gettempdir', autospec=True, return_value='/t'):
        self.assertEqual(cloudtail_factory._choose_log_dir(), '/t')

  def test_choose_log_dir_default_smoke(self):
    mock.patch.stopall()  # Use original value for DEFAULT_LOG_DIRECTORIES.
    cloudtail_factory._choose_log_dir()


class DummyCloudtailFactoryTest(unittest.TestCase):
  def test_start(self):
    f = cloudtail_factory.DummyCloudtailFactory()

    with self.assertRaises(OSError):
      f.start('foo', mock.Mock())

# Copyright (c) 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os
import logging
import subprocess
import tempfile
import threading

import infra_libs.logs.logs


class DummyCloudtailFactory(object):
  def start(self, *_args, **_kwargs):
    raise OSError('Cloudtail is not configured')


class CloudtailFactory(object):
  """Object that knows how to start cloudtail processes."""

  def __init__(self, path, ts_mon_credentials):
    self._path = path
    self._ts_mon_credentials = ts_mon_credentials

    # In practice, cloudtail is launched only once per daemonized process by
    # the daemon process itself. So, at the moment, the state here isn't
    # strictly speaking necessary. However, given "Factory" in class name, it's
    # easy to imagine unsuspecting devs expecting to be able to do several
    # factory uses per process lifetime.
    self._lock = threading.Lock()
    self._counter = 0
    self._log_dir = None

  def start(self, log_name, stdin_fh, **popen_kwargs):
    """Starts cloudtail in a subprocess.  Thread-safe.

    Args:
      log_name: --log-id argument passed to cloudtail.
      stdin_fh: File object or descriptor to be connected to cloudtail's stdin.
      popen_kwargs: Any additional keyword arguments to pass to Popen.

    Raises:
      OSError
    """
    args = [
        self._path, 'pipe',
        '--log-id', log_name,
        '--local-log-level', 'info',
        '--local-log-file', self._get_log_file(),
    ]

    if self._ts_mon_credentials:
      args.extend(['--ts-mon-credentials', self._ts_mon_credentials])

    with open(os.devnull, 'w') as null_fh:
      subprocess.Popen(
          args,
          stdin=stdin_fh,
          stdout=null_fh,
          stderr=null_fh,
          **popen_kwargs)

  def _get_log_file(self):
    with self._lock:
      if self._log_dir is None:
        self._log_dir = _choose_log_dir()
      self._counter += 1
      counter = self._counter
    return os.path.join(
        self._log_dir, 'sm.%d.cloudtail.%d.log' % (os.getpid(), counter))


def _choose_log_dir():
  # On Chrome puppet managed machines, one of the log directories should exist.
  # NOTE: DEFAULT_LOG_DIRECTORIES is a string, not a list.
  candidates = infra_libs.logs.logs.DEFAULT_LOG_DIRECTORIES.split(os.pathsep)
  for d in candidates:
    try:
      with tempfile.TemporaryFile(dir=d):
        pass
      return d
    except OSError:
      continue
  d = tempfile.gettempdir()
  logging.warn(
      'default infra log %s dirs not writable, using %s instead', candidates, d)
  return d

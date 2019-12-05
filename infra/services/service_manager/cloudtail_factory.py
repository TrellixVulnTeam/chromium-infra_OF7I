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
    log_out, log_err = self._open_log_files()
    logging.info(
        'cloudtail for %s is itself logged into %s and %s',
        log_name, log_out, log_err)

    args = [
        self._path, 'pipe',
        '--log-id', log_name,
        '--local-log-level', 'info',
    ]

    if self._ts_mon_credentials:
      args.extend(['--ts-mon-credentials', self._ts_mon_credentials])

    with open(log_out, 'w') as fout, open(log_err, 'w') as ferr:
      subprocess.Popen(
          args,
          stdin=stdin_fh,
          stdout=fout,
          stderr=ferr,
          **popen_kwargs)

  def _open_log_files(self):
    with self._lock:
      if self._log_dir is None:
        self._log_dir = _choose_log_dir()
      self._counter += 1
      counter = self._counter
    base = 'sm.%d.cloudtail.%d.std' % (os.getpid(), counter)
    out, err = [os.path.join(self._log_dir, base + e) for e in ['out', 'err']]
    return out, err


def _choose_log_dir():
  # On Chrome puppet managed machines, one of the log directories should exist.
  candidates = infra_libs.logs.logs.DEFAULT_LOG_DIRECTORIES
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

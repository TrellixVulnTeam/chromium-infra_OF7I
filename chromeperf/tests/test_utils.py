# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import contextlib
import subprocess

import psutil

from google.cloud.environment_vars import BIGTABLE_EMULATOR
from google.cloud.environment_vars import GCD_DATASET
from google.cloud.environment_vars import GCD_HOST
from google.cloud.environment_vars import PUBSUB_EMULATOR

BIGTABLE = 'bigtable'
DATASTORE = 'datastore'
PUBSUB = 'pubsub'
EXTRA = {
    DATASTORE: ("--no-store-on-disk", "--consistency=1.0"),
}
_DS_READY_LINE = b'[datastore] Dev App Server is now running.\n'
_PS_READY_LINE_PREFIX = b'[pubsub] INFO: Server started, listening on '
_BT_READY_LINE_PREFIX = b'[bigtable] Cloud Bigtable emulator running on '


def _get_start_command(package, port):
    """Get command line arguments for starting emulator.
    :type package: str
    :param package: The package to start an emulator for.
    :rtype: tuple
    :returns: The arguments to be used, in a tuple.
    """
    result = ('gcloud', 'beta', 'emulators', package, 'start')
    extra = EXTRA.get(package, ())
    return result + extra + (f"--host-port=127.0.0.1:{port}", )


def _datastore_wait_ready(popen):
    """Wait until the datastore emulator is ready to use.
    :type popen: :class:`subprocess.Popen`
    :param popen: An open subprocess to interact with.
    """
    envs = {}
    emulator_ready = False
    while not emulator_ready:
        line = popen.stderr.readline()
        if line.startswith(b'[datastore]   export '):
            k, v = line.decode().strip().split(' ')[-1].split('=')
            envs[k] = v
        emulator_ready = line == _DS_READY_LINE
    return envs


def _wait_ready_prefix(popen, prefix):
    """Wait until the a process encounters a line with matching prefix.
    :type popen: :class:`subprocess.Popen`
    :param popen: An open subprocess to interact with.
    :type prefix: str
    :param prefix: The prefix to match
    """
    emulator_ready = False
    while not emulator_ready:
        emulator_ready = popen.stderr.readline().startswith(prefix)


def _wait_ready(package, popen):
    """Wait until the emulator is ready to use.
    :type package: str
    :param package: The package to check if ready.
    :type popen: :class:`subprocess.Popen`
    :param popen: An open subprocess to interact with.
    :raises: :class:`KeyError` if the ``package`` is not among
             ``datastore``, ``pubsub`` or ``bigtable``.
    """
    if package == DATASTORE:
        return _datastore_wait_ready(popen)
    elif package == PUBSUB:
        return _wait_ready_prefix(popen, _PS_READY_LINE_PREFIX)
    elif package == BIGTABLE:
        return _wait_ready_prefix(popen, _BT_READY_LINE_PREFIX)
    else:
        raise KeyError('Package not supported', package)


def _cleanup_emulator(pid):
    """Cleanup a process (including all of its children).
    :type pid: int
    :param pid: Process ID.
    """
    proc = psutil.Process(pid)
    for child_proc in proc.children(recursive=True):
        try:
            child_proc.kill()
            child_proc.terminate()
        except psutil.NoSuchProcess:
            pass
    proc.terminate()
    proc.kill()


@contextlib.contextmanager
def with_emulator(package, port):
    start_command = _get_start_command(package, port)
    # Ignore stdin and stdout, don't pollute the user's output with them.
    proc_start = subprocess.Popen(
        start_command,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    try:
        yield _wait_ready(package, proc_start)
    finally:
        _cleanup_emulator(proc_start.pid)

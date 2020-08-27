# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Run tests locally with the emulator.
First makes system calls to spawn the emulator and get the local environment
variable needed for it. Then calls the tests.
"""

import argparse
import os
import subprocess

import psutil

from google.cloud.environment_vars import BIGTABLE_EMULATOR
from google.cloud.environment_vars import GCD_DATASET
from google.cloud.environment_vars import GCD_HOST
from google.cloud.environment_vars import PUBSUB_EMULATOR

BIGTABLE = 'bigtable'
DATASTORE = 'datastore'
PUBSUB = 'pubsub'
PACKAGE_INFO = {
    BIGTABLE: (BIGTABLE_EMULATOR,),
    DATASTORE: (GCD_DATASET, GCD_HOST),
    PUBSUB: (PUBSUB_EMULATOR,),
}
EXTRA = {
    DATASTORE: ("--no-store-on-disk",),
}
_DS_READY_LINE = b'[datastore] Dev App Server is now running.\n'
_PS_READY_LINE_PREFIX = b'[pubsub] INFO: Server started, listening on '
_BT_READY_LINE_PREFIX = b'[bigtable] Cloud Bigtable emulator running on '


def get_parser():
  """Get simple ``argparse`` parser to determine package.
    :rtype: :class:`argparse.ArgumentParser`
    :returns: The parser for this script.
    """
  parser = argparse.ArgumentParser(
      description='Run google-cloud system tests against local emulator.')
  parser.add_argument(
      '--package',
      dest='package',
      choices=sorted(PACKAGE_INFO.keys()),
      default=DATASTORE,
      help='Package to be tested.')
  parser.add_argument('commands', nargs=argparse.REMAINDER)
  return parser


def get_start_command(package):
  """Get command line arguments for starting emulator.
    :type package: str
    :param package: The package to start an emulator for.
    :rtype: tuple
    :returns: The arguments to be used, in a tuple.
    """
  result = ('gcloud', 'beta', 'emulators', package, 'start')
  extra = EXTRA.get(package, ())
  return result + extra


def get_env_init_command(package):
  """Get command line arguments for getting emulator env. info.
    :type package: str
    :param package: The package to get environment info for.
    :rtype: tuple
    :returns: The arguments to be used, in a tuple.
    """
  result = ('gcloud', 'beta', 'emulators', package, 'env-init')
  return result


def datastore_wait_ready(popen):
  """Wait until the datastore emulator is ready to use.
    :type popen: :class:`subprocess.Popen`
    :param popen: An open subprocess to interact with.
    """
  emulator_ready = False
  while not emulator_ready:
    emulator_ready = popen.stderr.readline() == _DS_READY_LINE


def wait_ready_prefix(popen, prefix):
  """Wait until the a process encounters a line with matching prefix.
    :type popen: :class:`subprocess.Popen`
    :param popen: An open subprocess to interact with.
    :type prefix: str
    :param prefix: The prefix to match
    """
  emulator_ready = False
  while not emulator_ready:
    emulator_ready = popen.stderr.readline().startswith(prefix)


def wait_ready(package, popen):
  """Wait until the emulator is ready to use.
    :type package: str
    :param package: The package to check if ready.
    :type popen: :class:`subprocess.Popen`
    :param popen: An open subprocess to interact with.
    :raises: :class:`KeyError` if the ``package`` is not among
             ``datastore``, ``pubsub`` or ``bigtable``.
    """
  if package == DATASTORE:
    datastore_wait_ready(popen)
  elif package == PUBSUB:
    wait_ready_prefix(popen, _PS_READY_LINE_PREFIX)
  elif package == BIGTABLE:
    wait_ready_prefix(popen, _BT_READY_LINE_PREFIX)
  else:
    raise KeyError('Package not supported', package)


def cleanup(pid):
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


def run_commands_with_emulator(package, commands):
  """Spawn an emulator instance and run the system tests.
    :type package: str
    :param package: The package to run system tests against.
    """
  # Make sure this package has environment vars to replace.
  env_vars = PACKAGE_INFO[package]

  start_command = get_start_command(package)
  # Ignore stdin and stdout, don't pollute the user's output with them.
  proc_start = subprocess.Popen(
      start_command, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
  try:
    wait_ready(package, proc_start)
    env_init_command = get_env_init_command(package)
    proc_env = subprocess.Popen(
        env_init_command, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    env_status = proc_env.wait()
    if env_status != 0:
      raise RuntimeError(env_status, proc_env.stderr.read())
    env_lines = proc_env.stdout.read().decode().strip().split('\n')
    # Set environment variables before running the system tests.
    for env_var in env_vars:
      line_prefix = 'export ' + env_var + '='
      value, = [
          line.split(line_prefix, 1)[1]
          for line in env_lines
          if line.startswith(line_prefix)
      ]
      os.environ[env_var] = value

    return subprocess.call(commands, env=os.environ)
  finally:
    cleanup(proc_start.pid)


def main():
  """Main method to run this script."""
  parser = get_parser()
  args = parser.parse_args()
  return run_commands_with_emulator(args.package, args.commands)


if __name__ == '__main__':
  exit(main())

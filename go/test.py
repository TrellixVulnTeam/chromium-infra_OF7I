#!/usr/bin/env vpython
# Copyright 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Runs all Go unit tests in a directory.

Expects Go toolset to be in PATH, GOPATH and GOROOT correctly set. Use ./env.py
to set them up.

Usage:
  test.py [root package path]

By default runs all tests for infra/*.
"""

# TODO(vadimsh): Get rid of this and call "go test ./..." directly from recipes.
# This file once had a much more complicated implementation that verified code
# coverage and allowed skipping tests per platform.

from __future__ import absolute_import
from __future__ import print_function
import errno
import json
import os
import subprocess
import sys

# /path/to/infra
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# Result adapter is deployed here by bootstrap.py.
ADAPTER_DIR = os.path.join(ROOT, "cipd", "result_adapter")


def check_go_available():
  """Returns True if go executable is in the PATH."""
  try:
    subprocess.check_output(['go', 'version'], stderr=subprocess.STDOUT)
    return True
  except subprocess.CalledProcessError:
    return False
  except OSError as err:
    if err.errno == errno.ENOENT:
      return False


def clean_go_bin():
  """Removes all files in GOBIN.

  GOBIN is in PATH in our environment. There are some binaries there (like 'git'
  for gitwrapper), that get mistakenly picked up by the tests.
  """
  gobin = os.environ.get('GOBIN')
  if not gobin or not os.path.exists(gobin):
    return
  for p in os.listdir(gobin):
    os.remove(os.path.join(gobin, p))


def use_resultdb():
  """Checks the luci context to determine if resultdb is configured."""
  ctx_filename = os.environ.get("LUCI_CONTEXT")
  if ctx_filename:
    try:
      with open(ctx_filename) as ctx_file:
        ctx = json.load(ctx_file)
        rdb = ctx.get('resultdb', {})
        return rdb.get('hostname') and rdb.get('current_invocation')
    except (OSError, ValueError):
      print(
          "Failed to open LUCI_CONTEXT; skip enabling resultdb integration",
          file=sys.stderr)
      return False
  return False


def get_adapter_path():
  adapter_fname = "result_adapter"
  if sys.platform == "win32":
    adapter_fname += ".exe"
  return os.path.join(ADAPTER_DIR, adapter_fname)


def run_vet(package_root):
  """Runs 'go vet <package_root>/...'

  Returns:
   0 if and only if all tests pass.
  """
  if not check_go_available():
    print('Can\'t find Go executable in PATH.')
    print('Go vet not supported')
    return 1

  # Turn off copylock analysis. Eventually, when we stop copying protobufs
  # in various places, we can turn it on.
  command = ['go', 'vet', '-copylocks=false', '%s/...' % package_root]

  # TODO: adapt results of go vet to resultdb.

  return subprocess.Popen(command).wait()


def run_tests(package_root):
  """Runs 'go test <package_root>/...'.

  Returns:
    0 if all tests pass..
  """
  if not check_go_available():
    print('Can\'t find Go executable in PATH.')
    print('Use ./env.py python test.py')
    return 1
  clean_go_bin()
  command = ['go', 'test', '%s/...' % package_root]

  prev_env = os.environ.copy()
  if use_resultdb():
    # Silence goconvey reporter to avoid interference with result_adapter.
    # https://github.com/smartystreets/goconvey/blob/0fc5ef5371303f55e76d89a57286fb7076777e5b/convey/init.go#L37
    os.environ['GOCONVEY_REPORTER'] = 'silent'
    command = [get_adapter_path(), 'go', '--'] + command
  try:
    return subprocess.Popen(command).wait()
  finally:
    os.environ.clear()
    os.environ.update(prev_env)


def run_all(package_root):
  """Run go vet and then go tests

  Returns:
    0 if and only if all tests pass.
  """

  # Always run every applicable action so we give the user as much information
  # as possible.
  results = [run_vet(package_root), run_tests(package_root)]

  for res in results:
    if res:
      return res
  return 0


def main(args):
  if not args:
    package_root = 'infra'
  elif len(args) == 1:
    package_root = args[0]
  else:
    print(sys.modules['__main__'].__doc__.strip(), file=sys.stderr)
    return 1
  return run_all(package_root)


if __name__ == '__main__':
  sys.exit(main(sys.argv[1:]))

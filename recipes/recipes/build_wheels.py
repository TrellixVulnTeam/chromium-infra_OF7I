# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEPS = [
    'depot_tools/gclient',
    'depot_tools/windows_sdk',
    'depot_tools/osx_sdk',
    'recipe_engine/context',
    'recipe_engine/file',
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/python',
]

from contextlib import contextmanager


def RunSteps(api):
  solution_path = api.path['cache'].join('builder', 'build_wheels')
  api.file.ensure_directory("init cache if it doesn't exist", solution_path)
  with api.context(cwd=solution_path):
    api.gclient.set_config('infra')
    api.gclient.c.solutions[0].revision = 'origin/master'
    api.gclient.checkout(timeout=10 * 60)
    api.gclient.runhooks()

  temp_path = api.path.mkdtemp('.dockerbuild')

  # DISTUTILS_USE_SDK and MSSdk are necessary for distutils to correctly locate
  # MSVC on Windows. They do nothing on other platforms, so we just set them
  # unconditionally.
  with PlatformSdk(api), api.context(
      cwd=solution_path.join('infra'),
      env={
          'DISTUTILS_USE_SDK': '1',
          'MSSdk': '1',
      }):
    api.python('dockerbuild', solution_path.join('infra', 'run.py'), [
        'infra.tools.dockerbuild',
        '--root',
        temp_path,
        '--upload-sources',
        'wheel-build',
        '--upload',
    ])


@contextmanager
def PlatformSdk(api):
  sdk = None
  if api.platform.is_win:
    sdk = api.windows_sdk()
  elif api.platform.is_mac:
    sdk = api.osx_sdk('mac')

  if sdk is None:
    yield
  else:
    with sdk:
      yield


def GenTests(api):
  yield api.test('success')
  yield api.test('win', api.platform('win', 64))
  yield api.test('mac', api.platform('mac', 64))

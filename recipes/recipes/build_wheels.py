# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from contextlib import contextmanager

from recipe_engine.recipe_api import Property
from recipe_engine.config import List

DEPS = [
    'depot_tools/gclient',
    'depot_tools/windows_sdk',
    'depot_tools/osx_sdk',
    'recipe_engine/context',
    'recipe_engine/file',
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/properties',
    'recipe_engine/python',
]

PROPERTIES = {
    'platforms':
        Property(
            help=('The platforms to build wheels for. On Windows, required to '
                  'be set to a list of either 32-bit or 64-bit platforms. '
                  'For other platforms, if empty, builds for all '
                  'platforms which are supported on this host.'),
            kind=List(str),
            default=(),
        ),
    'dry_run':
        Property(
            help='If true, do not upload wheels or source to CIPD.',
            kind=bool,
            default=False,
        ),
    'rebuild':
        Property(
            help=("If true, build all wheels regardless of whether they're "
                  "already in CIPD"),
            kind=bool,
            default=False,
        ),
}


def RunSteps(api, platforms, dry_run, rebuild):
  solution_path = api.path['cache'].join('builder', 'build_wheels')
  api.file.ensure_directory("init cache if it doesn't exist", solution_path)
  with api.context(cwd=solution_path):
    api.gclient.set_config('infra')
    api.gclient.c.solutions[0].revision = 'origin/master'
    api.gclient.checkout(timeout=10 * 60)
    api.gclient.runhooks()

  temp_path = api.path.mkdtemp('.dockerbuild')

  platform_args = []
  for p in platforms:
    platform_args.extend(['--platform', p])

  # DISTUTILS_USE_SDK and MSSdk are necessary for distutils to correctly locate
  # MSVC on Windows. They do nothing on other platforms, so we just set them
  # unconditionally.
  with PlatformSdk(api, platforms), api.context(
      cwd=solution_path.join('infra'),
      env={
          'DISTUTILS_USE_SDK': '1',
          'MSSdk': '1',
      }):
    args = [
        'infra.tools.dockerbuild',
        '--root',
        temp_path,
    ]
    if not dry_run:
      args.append('--upload-sources')
    args.append('wheel-build')
    if not dry_run:
      args.append('--upload')

    if rebuild:
      args.append('--rebuild')

    api.python('dockerbuild', solution_path.join('infra', 'run.py'),
               args + platform_args)


@contextmanager
def PlatformSdk(api, platforms):
  sdk = None
  if api.platform.is_win:
    is_64bit = all((p.startswith('windows-x64') for p in platforms))
    is_32bit = all((p.startswith('windows-x86') for p in platforms))
    if is_64bit == is_32bit:
      raise ValueError(
          'Must specify either 32-bit or 64-bit windows platforms.')
    target_arch = 'x64' if is_64bit else 'x86'
    sdk = api.windows_sdk(target_arch=target_arch)
  elif api.platform.is_mac:
    sdk = api.osx_sdk('mac')

  if sdk is None:
    yield
  else:
    with sdk:
      yield


def GenTests(api):
  yield api.test('success')
  yield api.test(
      'win',
      api.platform('win', 64) +
      api.properties(platforms=['windows-x64', 'windows-x64-py3']))
  yield api.test(
      'win-x86',
      api.platform('win', 64) +
      api.properties(platforms=['windows-x86', 'windows-x86-py3']))
  yield api.test('mac', api.platform(
      'mac', 64)) + api.properties(platforms=['mac-x64', 'mac-x64-cp38'])
  yield api.test('dry-run', api.properties(dry_run=True))

  # Can't build 32-bit and 64-bit Windows wheels on the same invocation.
  yield api.test(
      'win-32and64bit',
      api.platform('win', 64) +
      api.properties(platforms=['windows-x64', 'windows-x86']) +
      api.expect_exception('ValueError'))
  # Must explicitly specify the platforms to build on Windows.
  yield api.test('win-noplatforms',
                 api.platform('win', 64) + api.expect_exception('ValueError'))

  yield api.test('trybot config', api.properties(dry_run=True, rebuild=True))

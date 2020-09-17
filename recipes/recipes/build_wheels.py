# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEPS = [
    'depot_tools/gclient',
    'recipe_engine/context',
    'recipe_engine/file',
    'recipe_engine/path',
    'recipe_engine/python',
]


def RunSteps(api):
  solution_path = api.path['cache'].join('builder', 'solution')
  api.file.ensure_directory("init cache if it doesn't exist", solution_path)
  with api.context(cwd=solution_path):
    api.gclient.set_config('infra')
    api.gclient.c.solutions[0].revision = 'origin/master'
    api.gclient.checkout(timeout=10 * 60)
    api.gclient.runhooks()

  temp_path = api.path.mkdtemp('.dockerbuild')

  # Set MACOSX_DEPLOYMENT_TARGET to 10.13 to ensure compatibility regardless of
  # the OS version the bot is running. This environment variable is unnecessary
  # but harmless on other OSes, so we just set it unconditionally.
  with api.context(
      cwd=solution_path.join('infra'),
      env={'MACOSX_DEPLOYMENT_TARGET': '10.13'}):
    api.python('dockerbuild', solution_path.join('infra', 'run.py'), [
        'infra.tools.dockerbuild',
        '--root',
        temp_path,
        '--upload-sources',
        'wheel-build',
        '--upload',
    ])


def GenTests(api):
  yield api.test('success')

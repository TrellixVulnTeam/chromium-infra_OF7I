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
    api.gclient.c.solutions[0].revision = 'origin/deployed'
    api.gclient.checkout(timeout=10 * 60)
    api.gclient.runhooks()

  api.python('dockerbuild', solution_path.join('infra', 'run.py'), [
      'infra.tools.dockerbuild', '--upload-sources', 'wheel-build', '--upload'
  ])

  # TODO: Update README.wheels.md as well?


def GenTests(api):
  yield api.test('success')

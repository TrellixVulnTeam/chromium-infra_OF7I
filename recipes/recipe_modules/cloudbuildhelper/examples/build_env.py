# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

PYTHON_VERSION_COMPATIBILITY = 'PY2+3'

DEPS = [
    'recipe_engine/path',
    'recipe_engine/step',

    'cloudbuildhelper',
]


def RunSteps(api):
  with api.cloudbuildhelper.build_environment(
      api.path['start_dir'], 'GO_VERSION', 'NODEJS_VERSION'):
    api.step('full env', ['echo', 'hi'])

  with api.cloudbuildhelper.build_environment(api.path['start_dir']):
    api.step('empty env', ['echo', 'hi'])


def GenTests(api):
  yield api.test('full')

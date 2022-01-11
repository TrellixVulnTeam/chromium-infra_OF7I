# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

PYTHON_VERSION_COMPATIBILITY = 'PY2+3'

DEPS = [
  'cloudbuildhelper',
  'recipe_engine/path',
]


def RunSteps(api):
  paths = api.cloudbuildhelper.discover_manifests(
      root=api.path['cache'],
      entries=['stuff/direct.yaml', '1', '2'])
  assert paths == [
      api.path['cache'].join('stuff', 'direct.yaml'),
      api.path['cache'].join('1', 'target.yaml'),
      api.path['cache'].join('2', 'target.yaml'),
  ], paths


def GenTests(api):
  yield api.test('works')

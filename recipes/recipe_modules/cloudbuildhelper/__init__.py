# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

PYTHON_VERSION_COMPATIBILITY = 'PY2+3'

DEPS = [
  'recipe_engine/buildbucket',
  'recipe_engine/cipd',
  'recipe_engine/commit_position',
  'recipe_engine/context',
  'recipe_engine/file',
  'recipe_engine/golang',
  'recipe_engine/json',
  'recipe_engine/nodejs',
  'recipe_engine/path',
  'recipe_engine/raw_io',
  'recipe_engine/step',
  'recipe_engine/time',

  'depot_tools/depot_tools',
  'depot_tools/git',
  'depot_tools/git_cl',
]

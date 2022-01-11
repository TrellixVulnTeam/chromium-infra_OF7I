# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

PYTHON_VERSION_COMPATIBILITY = 'PY2+3'

DEPS = [
    'recipe_engine/path',
    'recipe_engine/properties',
    'recipe_engine/step',

    'cloudbuildhelper',
]


from recipe_engine.recipe_api import Property

PROPERTIES = {
    'commit_position': Property(kind=str),
}


def RunSteps(api, commit_position):
  api.step('label 1', ['echo', api.cloudbuildhelper.get_version_label(
      path=api.path['checkout'],
      revision='abcdefabcdef63ad814cd1dfffe2fcfc9f81299c',
      commit_position=commit_position,
  )])
  api.step('label 2', ['echo', api.cloudbuildhelper.get_version_label(
      path=api.path['checkout'],
      revision='abcdefabcdef63ad814cd1dfffe2fcfc9f81299c',
      commit_position=commit_position,
      template='{rev}-{ref}-{cp}-{date}-{build}',
  )])


def GenTests(api):
  yield api.test(
      'with_commit_position',
      api.properties(commit_position='refs/heads/main@{#45448}'),
  )

  yield api.test(
      'without_commit_position',
      api.properties(commit_position=None),
  )

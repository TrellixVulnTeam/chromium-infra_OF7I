# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

PYTHON_VERSION_COMPATIBILITY = 'PY2+3'

from recipe_engine.recipe_api import Property

DEPS = [
  'cloudbuildhelper',
  'recipe_engine/path',
  'recipe_engine/properties',
]


PROPERTIES = {
    'commit': Property(kind=bool, default=True),
}


def RunSteps(api, commit):
  def roll_cb(_):
    return api.cloudbuildhelper.RollCL(
        message='Title\n\nBody',
        cc=['a@cc.com', 'b@cc.com'],
        tbr=['a@tbr.com', 'b@tbr.com'],
        commit=commit)

  api.cloudbuildhelper.do_roll(
      'https://repo.example.com', api.path['cache'].join('roll'), roll_cb)


def GenTests(api):
  yield api.test('clean')

  yield (
      api.test('dirty') +
      api.step_data('git diff', retcode=1)
  )

  yield (
      api.test('no_commit') +
      api.properties(commit=False) +
      api.step_data('git diff', retcode=1)
  )

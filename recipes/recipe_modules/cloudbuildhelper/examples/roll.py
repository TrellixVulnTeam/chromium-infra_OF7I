# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEPS = [
  'cloudbuildhelper',
  'recipe_engine/path',
]


def RunSteps(api):
  def roll_cb(_):
    return api.cloudbuildhelper.RollCL(
        message='Title\n\nBody',
        cc=['a@cc.com', 'b@cc.com'],
        tbr=['a@tbr.com', 'b@tbr.com'],
        commit=True)

  api.cloudbuildhelper.do_roll(
      'https://repo.example.com', api.path['cache'].join('roll'), roll_cb)


def GenTests(api):
  yield api.test('clean')

  yield (
      api.test('dirty') +
      api.step_data('git diff', retcode=1)
  )

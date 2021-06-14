# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine.post_process import StepCommandRE

DEPS = [
    'windows_adk',
    'recipe_engine/properties',
]


def RunSteps(api):
  api.windows_adk.ensure()


def GenTests(api):
  yield (api.test('basic') +
         api.properties(win_adk_refs='canary', win_adk_winpe_refs='canary') +
         api.post_process(StepCommandRE, 'ensure windows adk present', []) +
         api.post_process(StepCommandRE, 'ensure win-pe add-on present', []))

# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine.post_process import StepCommandRE, DropExpectation

DEPS = [
    'windows_adk',
    'recipe_engine/properties',
    'recipe_engine/file',
    'recipe_engine/path',
]


def RunSteps(api):
  api.windows_adk.ensure()
  api.windows_adk.cleanup()


def GenTests(api):
  yield (api.test('basic') +
         api.properties(win_adk_refs='canary', win_adk_winpe_refs='canary') +
         api.post_process(StepCommandRE, 'ensure windows adk present', []) +
         api.post_process(StepCommandRE, 'ensure win-pe add-on present', []) +
         api.post_process(StepCommandRE, 'Uninstall ADK', [
             '\[START_DIR\]/cipd/3pp/adk/raw_source_0.exe', '/q', '/uninstall',
             '/l', '\[CLEANUP\]/logs/adk-uninstall/adk.log'
         ]) + api.post_process(StepCommandRE, 'Uninstall WinPE', [
             '\[START_DIR\]/cipd/3pp/winpe/raw_source_0.exe', '/q',
             '/uninstall', '/l', '\[CLEANUP\]/logs/winpe-uninstall/winpe.log'
         ]) + api.post_process(DropExpectation))

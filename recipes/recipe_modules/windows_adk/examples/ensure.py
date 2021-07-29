# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine.post_process import StepCommandRE, DropExpectation

DEPS = [
    'windows_adk',
    'recipe_engine/properties',
    'recipe_engine/file',
    'recipe_engine/json',
    'recipe_engine/path',
]


def RunSteps(api):
  api.windows_adk.ensure()
  api.windows_adk.cleanup()


def GenTests(api):
  STEP_INSTALL_ADK_PASS = api.step_data(
      'ensure windows adk present.PowerShell> Install ADK',
      stdout=api.json.output({
          'results': {
              'Success': True
          },
          '[CLEANUP]/logs/adk/adk.log': 'i007: Exit code: 0x0',
      }))
  STEP_INSTALL_WINPE_PASS = api.step_data(
      'ensure win-pe add-on present.PowerShell> Install WinPE',
      stdout=api.json.output({
          'results': {
              'Success': True
          },
          '[CLEANUP]/logs/winpe/winpe.log': 'i007: Exit code: 0x0',
      }))
  STEP_UNINSTALL_ADK_PASS = api.step_data(
      'PowerShell> Uninstall ADK',
      stdout=api.json.output({
          'results': {
              'Success': True
          },
          '[CLEANUP]/logs/adk-uninstall/adk.log': 'i007: Exit code: 0x0',
      }))
  STEP_UNINSTALL_WINPE_PASS = api.step_data(
      'PowerShell> Uninstall WinPE',
      stdout=api.json.output({
          'results': {
              'Success': True
          },
          '[CLEANUP]/logs/winpe-uninstall/winpe.log': 'i007: Exit code: 0x0',
      }))

  yield (api.test('basic') +
         api.properties(win_adk_refs='canary', win_adk_winpe_refs='canary') +
         api.post_process(StepCommandRE, 'ensure windows adk present', []) +
         api.post_process(StepCommandRE, 'ensure win-pe add-on present', []) +
         STEP_INSTALL_ADK_PASS + STEP_INSTALL_WINPE_PASS +
         STEP_UNINSTALL_ADK_PASS + STEP_UNINSTALL_WINPE_PASS +
         api.post_process(DropExpectation))

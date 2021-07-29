# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine.post_process import StepCommandRE, DropExpectation
from recipe_engine.post_process import StatusFailure, StatusSuccess

DEPS = [
    'powershell',
    'recipe_engine/properties',
    'recipe_engine/file',
    'recipe_engine/path',
    'recipe_engine/json',
]


def RunSteps(api):
  api.powershell('basic', 'dir', logs=['logs'], args=['/q', '/4'])


def GenTests(api):
  yield (api.test('fail') + api.step_data(
      'PowerShell> basic',
      stdout=api.json.output({
          'results': {
              'Success': False,
              'Command': 'powershell',
              'ErrorInfo': {
                  'Message': 'Failed step',
              },
          }
      })) + api.post_process(StatusFailure) +  # Failed due to StepFailure
         api.post_process(DropExpectation))

  yield (api.test('pass') + api.step_data(
      'PowerShell> basic',
      stdout=api.json.output({'results': {
          'Success': True,
      }})) + api.post_process(StatusSuccess) +  # No failures
         api.post_process(DropExpectation))

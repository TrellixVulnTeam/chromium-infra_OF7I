# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib

from recipe_engine.post_process import DropExpectation, StatusFailure
from recipe_engine.post_process import StatusSuccess, StepCommandRE

DEPS = [
    'windows_scripts_executor',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
]

PROPERTIES = wib.EditOfflineRegistry


def RunSteps(api, edit_offline_registry_action):
  api.windows_scripts_executor.edit_offline_registry(
    edit_offline_registry_action
  )


def GenTests(api):
  EDIT_OFFLINE_REGISTRY_TAMPER_PROTECTION_PROPERTIES = wib.EditOfflineRegistry(
    name='edit tamper protection',
    reg_hive_file='Windows\\System32\\Config\\software',
    reg_key_path='Microsoft\\Windows Defender\\Features',
    property_name='TamperProtection',
    property_value='0',
    property_type='DWord',
  )

  EDIT_OFFLINE_REGISTRY_TAMPER_PROTECTION_PASS = api.step_data(
      'PowerShell> Edit Offline Registry Key Features and Property ' +
      'TamperProtection',
      stdout=api.json.output({
          'results': {
              'Success': True
          },
      })
    )

  EDIT_OFFLINE_REGISTRY_TAMPER_PROTECTION_FAIL = api.step_data(
      'PowerShell> Edit Offline Registry Key Features and Property ' +
      'TamperProtection',
      stdout=api.json.output(
        {
          'results': {
              'Success': False,
              'Command': 'powershell',
              'ErrorInfo': {
                  'Message': 'Failed step'
              },
          }
        }
      )
    )

  yield (
    # name
    api.test('Edit Offline Registry Action Fail', api.platform('win', 64)) +

    # test properties
    api.properties(EDIT_OFFLINE_REGISTRY_TAMPER_PROTECTION_PROPERTIES) +

    # test expectations
    EDIT_OFFLINE_REGISTRY_TAMPER_PROTECTION_FAIL +

    # test recipe exit status
    api.post_process(StatusFailure) +

    # additional optional params
    api.post_process(DropExpectation)
  )

  yield (
    # name
    api.test('Edit Offline Registry Action Pass', api.platform('win', 64)) +

    # test properties
    api.properties(EDIT_OFFLINE_REGISTRY_TAMPER_PROTECTION_PROPERTIES) +

    # test expectations
    EDIT_OFFLINE_REGISTRY_TAMPER_PROTECTION_PASS +

    # test recipe exit status
    api.post_process(StatusSuccess) +

    # additional optional params
    api.post_process(DropExpectation)
  )

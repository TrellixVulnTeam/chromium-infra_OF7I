# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder import actions

from recipe_engine.post_process import DropExpectation, StatusFailure
from recipe_engine.post_process import StatusSuccess, StepCommandRE
from RECIPE_MODULES.infra.windows_scripts_executor import test_helper as t

DEPS = [
    'windows_scripts_executor',
    'recipe_engine/path',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
    'recipe_engine/raw_io',
]

PROPERTIES = wib.Image

image = 'regedit_test'
customization = 'remove_tamper_protection'
arch = 'x86'
key = '96fe4737ff3346d68755d1359da74003c56d38571669d4c97602fd3f1d59d3f7'


def RunSteps(api, config):
  api.windows_scripts_executor.init(config)
  api.windows_scripts_executor.pin_available_sources()
  api.windows_scripts_executor.gen_canonical_configs(config)
  api.windows_scripts_executor.download_available_packages()
  api.windows_scripts_executor.execute_config(config)


def GenTests(api):
  EDIT_OFFLINE_REGISTRY_TAMPER_PROTECTION_PROPERTIES = actions.Action(
      edit_offline_registry=actions.EditOfflineRegistry(
          name='edit tamper protection',
          reg_hive_file='Windows\\System32\\Config\\software',
          reg_key_path='Microsoft\\Windows Defender\\Features',
          property_name='TamperProtection',
          property_value='0',
          property_type='DWord',
      ))

  yield (
      # name
      api.test('Edit Offline Registry Action Fail', api.platform('win', 64)) +

      # test properties using config for regedit tamper protection
      api.properties(
          t.WPE_IMAGE(image, wib.ARCH_X86, customization, 'regedit',
                      [EDIT_OFFLINE_REGISTRY_TAMPER_PROTECTION_PROPERTIES])) +

      # mock all the init and deinit steps
      t.MOCK_WPE_INIT_DEINIT_FAILURE(api, key, arch, image, customization) +

      # mock registry edit action
      t.EDIT_REGISTRY(api, 'TamperProtection', image, customization, False) +

      # test recipe exit status
      api.post_process(StatusFailure) +

      # additional optional params
      api.post_process(DropExpectation))

  yield (
      # name
      api.test('Edit Offline Registry Action Pass', api.platform('win', 64)) +

      # test properties
      api.properties(
          t.WPE_IMAGE(image, wib.ARCH_X86, customization, 'regedit',
                      [EDIT_OFFLINE_REGISTRY_TAMPER_PROTECTION_PROPERTIES])) +

      # mock all the init and deinit steps
      t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image, customization) +

      # mock registry edit action
      t.EDIT_REGISTRY(api, 'TamperProtection', image, customization) +

      # test recipe exit status
      api.post_process(StatusSuccess) +

      # additional optional params
      api.post_process(DropExpectation))

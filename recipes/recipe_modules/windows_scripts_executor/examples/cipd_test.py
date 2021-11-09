# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import (offline_winpe_customization
                                                    as winpe)
from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder import actions
from PB.recipes.infra.windows_image_builder import sources

from recipe_engine.post_process import DropExpectation, StatusFailure
from recipe_engine.post_process import StatusSuccess, StepCommandRE
from RECIPE_MODULES.infra.windows_scripts_executor import test_helper as t

DEPS = [
    'windows_scripts_executor',
    'recipe_engine/path',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
    'recipe_engine/raw_io'
]

PROPERTIES = wib.Image

image = 'cipd_test'
customization = 'cipd_add_file'
key = '835663538df204d1d6ba072b185850ba502e4520fddbfe2262562596511368af'


def RunSteps(api, config):
  api.windows_scripts_executor.init(config)
  api.windows_scripts_executor.pin_available_sources()
  api.windows_scripts_executor.gen_canonical_configs(config)
  # mock existence of cipd files to avoid failures
  api.path.mock_add_paths(
      '[CACHE]\\Pkgs\\CIPDPkgs\\resolved-instance_id-of-latest----------' +
      '\\infra\\files\\cipd-1\\windows-amd64')
  # mock existence of cipd files to avoid failures
  api.path.mock_add_paths(
      '[CACHE]\\Pkgs\\CIPDPkgs\\resolved-instance_id-of-latest----------' +
      '\\infra\\files\\cipd-2\\windows-amd64')
  api.windows_scripts_executor.download_available_packages()
  api.windows_scripts_executor.execute_config(config)


def GenTests(api):
  # add file from cipd to winpe image action
  ACTION_ADD_CIPD_1 = actions.Action(
      add_file=actions.AddFile(
          name='add cipd-1',
          src=sources.Src(
              cipd_src=sources.CIPDSrc(
                  package='infra/files/cipd-1',
                  refs='latest',
                  platform='windows-amd64',
              ),),
          dst='Windows\\Users\\',
      ))

  # add file from cipd to winpe image action
  ACTION_ADD_CIPD_2 = actions.Action(
      add_file=actions.AddFile(
          name='add cipd-2',
          src=sources.Src(
              cipd_src=sources.CIPDSrc(
                  package='infra/files/cipd-2',
                  refs='latest',
                  platform='windows-amd64',
              ),),
          dst='Windows\\Users\\',
      ))

  yield (
      api.test('Test cipd pin and download package', api.platform('win', 64)) +
      # image with an action to add file from cipd
      api.properties(
          t.WPE_IMAGE(image, wib.ARCH_X86, customization, 'add pkg from cipd',
                      [ACTION_ADD_CIPD_1])) +
      # mock init and deinit steps
      t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, 'x86', image, customization) +
      # mock add cipd file step
      t.ADD_CIPD_FILE(api, 'infra\\files\\cipd-1', 'windows-amd64', image,
                      customization) +
      # assert that the recipe was executed successfully
      api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  yield (
      api.test('Test cipd pin and download packages in single action',
               api.platform('win', 64)) +
      # image with two different actions to add files from cipd
      api.properties(
          t.WPE_IMAGE(image, wib.ARCH_X86, customization, 'add pkg from cipd',
                      [ACTION_ADD_CIPD_1, ACTION_ADD_CIPD_2])) +
      # mock init and deinit steps
      t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, 'x86', image, customization) +
      # mock add cipd file step
      t.ADD_CIPD_FILE(api, 'infra\\files\\cipd-1', 'windows-amd64', image,
                      customization) +
      # mock add cipd file step
      t.ADD_CIPD_FILE(api, 'infra\\files\\cipd-2', 'windows-amd64', image,
                      customization) +
      # assert that the recipe execution was a success
      api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  # image with multiple sub customization
  CIPD_PACKAGE_MULTIPLE_ACTIONS = t.WPE_IMAGE(
      image, wib.ARCH_X86, customization, 'add cipd pkgs',
      [ACTION_ADD_CIPD_1, ACTION_ADD_CIPD_2])
  cust = CIPD_PACKAGE_MULTIPLE_ACTIONS.customizations[0]
  cust.offline_winpe_customization.offline_customization.append(
      actions.OfflineAction(name='add cipd pkg', actions=[ACTION_ADD_CIPD_1]))

  yield (api.test('Test cipd pin and download packages in multiple actions',
                  api.platform('win', 64)) +
         # use the image with multiple actions
         api.properties(CIPD_PACKAGE_MULTIPLE_ACTIONS) +
         # mock all the init and deinit steps
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, 'x86', image, customization) +
         # mock add cipd file step
         t.ADD_CIPD_FILE(api, 'infra\\files\\cipd-1', 'windows-amd64', image,
                         customization) +
         # mock add cipd file step
         t.ADD_CIPD_FILE(api, 'infra\\files\\cipd-1', 'windows-amd64 (2)',
                         image, customization) +
         # mock add cipd file step
         t.ADD_CIPD_FILE(api, 'infra\\files\\cipd-2', 'windows-amd64', image,
                         customization) +
         # assert that the recipe executed successfully
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

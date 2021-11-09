# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder import actions
from PB.recipes.infra.windows_image_builder import (offline_winpe_customization
                                                    as winpe)
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

key = '82db41f183c1addbd1b7a2f9ec5a5e2cac46744a5a68f153dbf7ee76327cc491'
image = 'win10_2013_x64'
customization = 'offline_winpe_2013_x64'


def RunSteps(api, config):
  api.windows_scripts_executor.init(config)
  api.windows_scripts_executor.pin_available_sources()
  api.windows_scripts_executor.gen_canonical_configs(config)
  api.windows_scripts_executor.download_available_packages()
  api.windows_scripts_executor.execute_config(config)


def GenTests(api):
  # install file from cipd without extra args
  INSTALL_FILE_NO_ARGS = actions.Action(
      add_windows_package=actions.AddWindowsPackage(
          name='add cipd',
          src=sources.Src(
              cipd_src=sources.CIPDSrc(
                  package='infra/files/cipd-1',
                  refs='latest',
                  platform='windows-amd64',
              ),),
      ))

  # install file from cipd with args
  INSTALL_FILE_ARGS = actions.Action(
      add_windows_package=actions.AddWindowsPackage(
          name='add cipd',
          src=sources.Src(
              cipd_src=sources.CIPDSrc(
                  package='infra/files/cipd-1',
                  refs='latest',
                  platform='windows-amd64',
              ),),
          args=['-IgnoreCheck']))

  # add windows package without any args
  yield (api.test('Test Add-WindowsPackage no args', api.platform('win', 64)) +
         # input image with install file action without any args
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, customization, 'add pkg no args',
                         [INSTALL_FILE_NO_ARGS])) +
         # mock the install file step
         t.INSTALL_FILE(api, 'add cipd', image, customization) +
         # mock winpe init and deinit steps
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, 'x86', image, customization) +
         # assert that the install file step was executed
         t.CHECK_INSTALL_CAB(api, image, customization, 'add cipd') +
         # assert  that the execution was a success
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  yield (
      api.test('Test Add-WindowsPackage with args', api.platform('win', 64)) +
      # input image with install file action with args
      api.properties(
          t.WPE_IMAGE(image, wib.ARCH_X86, customization, 'add pkg no args',
                      [INSTALL_FILE_ARGS])) +
      # mock all the init and deinit steps
      t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, 'x86', image, customization) +
      # mock install file step
      t.INSTALL_FILE(api, 'add cipd', image, customization) +
      # assert that the install file step was executed with args
      t.CHECK_INSTALL_CAB(api, image, customization, 'add cipd',
                          ['-IgnoreCheck']) +
      # assert recipe execution was a success
      api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  yield (api.test('Test Add-WindowsPackage with args (Failure)',
                  api.platform('win', 64)) +
         # input image with install file action with args
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, customization, 'add pkg',
                         [INSTALL_FILE_ARGS])) +
         # mock all the init and deinit steps
         t.MOCK_WPE_INIT_DEINIT_FAILURE(api, key, 'x86', image, customization) +
         # mock install file step
         t.INSTALL_FILE(api, 'add cipd', image, customization, success=False) +
         # assert that the install file step was executed with args
         t.CHECK_INSTALL_CAB(api, image, customization, 'add cipd',
                             ['-IgnoreCheck']) +
         # assert that the execution was a failure
         api.post_process(StatusFailure) + api.post_process(DropExpectation))

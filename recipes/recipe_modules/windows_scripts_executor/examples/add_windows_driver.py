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
    'recipe_engine/path',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
    'recipe_engine/raw_io',
    'windows_scripts_executor'
]

PROPERTIES = wib.Image

key = '4587561855fcc569fc485e4f74b693870fd0d61eea5f40e9cef2ec9821d240a7'
image = 'add_windows_driver_test'
customization = 'add_windows_driver'


def RunSteps(api, config):
  api.windows_scripts_executor.init(config)
  api.windows_scripts_executor.pin_available_sources()
  api.windows_scripts_executor.gen_canonical_configs(config)
  api.windows_scripts_executor.download_available_packages()
  api.windows_scripts_executor.execute_config(config)


def GenTests(api):
  # install file from cipd without extra args
  INSTALL_FILE_NO_ARGS = actions.Action(
      add_windows_driver=actions.AddWindowsDriver(
          name='add driver from cipd',
          src=sources.Src(
              cipd_src=sources.CIPDSrc(
                  package='infra/files/cipd-1',
                  refs='latest',
                  platform='windows-amd64',
              ),),
      ))

  # install file from cipd with args
  INSTALL_FILE_ARGS = actions.Action(
      add_windows_driver=actions.AddWindowsDriver(
          name='add drivers from cipd',
          src=sources.Src(
              cipd_src=sources.CIPDSrc(
                  package='infra/files/cipd-2',
                  refs='latest',
                  platform='windows-amd64',
              ),),
          args=['-Recurse']))

  # add windows driver without any args
  yield (api.test('Test Add-WindowsDriver no args', api.platform('win', 64)) +
         # input image with install file action without any args
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, customization,
                         'add driver no args', [INSTALL_FILE_NO_ARGS])) +
         # mock the install driver step
         t.INSTALL_DRIVER(api, 'add driver from cipd', image, customization) +
         # mock winpe t and deinit steps
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, 'x86', image, customization) +
         # assert that the install driver step was executed
         t.CHECK_INSTALL_DRIVER(api, image, customization,
                                'add driver from cipd') +
         # assert  that the execution was a success
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  # add windows drivers with args
  yield (api.test('Test Add-WindowsDriver with args', api.platform('win', 64)) +
         # input image with install file action without any args
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, customization,
                         'add drivers with args', [INSTALL_FILE_ARGS])) +
         # mock the install driver step
         t.INSTALL_DRIVER(api, 'add drivers from cipd', image, customization) +
         # mock winpe t and deinit steps
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, 'x86', image, customization) +
         # assert that the install driver step was executed
         t.CHECK_INSTALL_DRIVER(api, image, customization,
                                'add drivers from cipd', ['-Recurse']) +
         # assert  that the execution was a success
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  # add windows drivers with args (failure case)
  yield (
      api.test('Test Add-WindowsDriver with args (Failure)',
               api.platform('win', 64)) +
      # input image with install file action without any args
      api.properties(
          t.WPE_IMAGE(image, wib.ARCH_X86, customization,
                      'add drivers with args', [INSTALL_FILE_ARGS])) +
      # mock the install driver step
      t.INSTALL_DRIVER(
          api, 'add drivers from cipd', image, customization, success=False) +
      # mock winpe t and deinit steps
      t.MOCK_WPE_INIT_DEINIT_FAILURE(api, key, 'x86', image, customization) +
      # assert that the install driver step was executed
      t.CHECK_INSTALL_DRIVER(api, image, customization, 'add drivers from cipd',
                             ['-Recurse']) +
      # assert  that the execution was a failure
      api.post_process(StatusFailure) + api.post_process(DropExpectation))

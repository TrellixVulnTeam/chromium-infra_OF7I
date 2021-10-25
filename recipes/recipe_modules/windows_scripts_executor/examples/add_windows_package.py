# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder import actions
from PB.recipes.infra.windows_image_builder import sources

from recipe_engine.post_process import DropExpectation, StatusFailure
from recipe_engine.post_process import StatusSuccess, StepCommandRE

DEPS = [
    'windows_scripts_executor',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
]

PROPERTIES = actions.AddWindowsPackage


def RunSteps(api, package):
  api.windows_scripts_executor.module_init()
  # Test pin and download cipd pkgs
  api.windows_scripts_executor.add_windows_package(package, 'C:\\Test')


def GenTests(api):
  INSTALL_PASS = api.step_data(
      'PowerShell> Install package add cipd',
      stdout=api.json.output({'results': {
          'Success': True,
      }}))

  INSTALL_FAIL = api.step_data(
      'PowerShell> Install package add cipd',
      stdout=api.json.output({
          'results': {
              'Success': False,
              'Command': 'powershell',
              'ErrorInfo': {
                  'Message': 'Failed step'
              },
          }
      }))

  INSTALL_FILE_NO_ARGS = actions.AddWindowsPackage(
      name='add cipd',
      src=sources.Src(
          cipd_src=sources.CIPDSrc(
              package='infra/files/cipd-1',
              refs='latest',
              platform='windows-amd64',
          ),),
  )

  INSTALL_FILE_ARGS = actions.AddWindowsPackage(
      name='add cipd',
      src=sources.Src(
          cipd_src=sources.CIPDSrc(
              package='infra/files/cipd-1',
              refs='latest',
              platform='windows-amd64',
          ),),
      args=['-IgnoreCheck'])

  yield (api.test('Test Add-WindowsPackage no args', api.platform('win', 64)) +
         api.properties(INSTALL_FILE_NO_ARGS) +  # Pass install file config
         INSTALL_PASS +  # Success on install file
         api.post_process(StatusSuccess) +  # Check if it is success
         api.post_process(DropExpectation))

  yield (
      api.test('Test Add-WindowsPackage with args', api.platform('win', 64)) +
      api.properties(INSTALL_FILE_ARGS) +  # Pass install file config
      INSTALL_PASS +  # Success on install file
      api.post_process(StatusSuccess) +  # Check if it is success
      api.post_process(DropExpectation))

  yield (api.test('Test Add-WindowsPackage with args (Failure)',
                  api.platform('win', 64)) +
         api.properties(INSTALL_FILE_ARGS) +  # Pass install file config
         INSTALL_FAIL +  # Failure on install file
         api.post_process(StatusFailure) +  # Assert failure
         api.post_process(DropExpectation))

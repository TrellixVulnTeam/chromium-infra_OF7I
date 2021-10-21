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

DEPS = [
    'windows_scripts_executor',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
]

PROPERTIES = wib.Image


def RunSteps(api, image):
  # Test pin and download cipd pkgs
  api.windows_scripts_executor.pin_wib_config(image)
  api.windows_scripts_executor.download_wib_artifacts(image)


def GenTests(api):
  PIN_CIPD_1_PASS = api.step_data(
      'Pin all the cipd packages.cipd describe ' +
      'infra/files/cipd-1/windows-amd64',
      stdout=api.json.output({
          'result': {
              'pin': {
                  'instance_id': 'cipd-1-instance-pin',
                  'package': 'infra/files/cipd-1/windows-amd64'
              },
          }
      }))

  PIN_CIPD_2_PASS = api.step_data(
      'Pin all the cipd packages.cipd describe ' +
      'infra/files/cipd-2/windows-amd64',
      stdout=api.json.output({
          'result': {
              'pin': {
                  'instance_id': 'cipd-2-instance-pin',
                  'package': 'infra/files/cipd-2/windows-amd64'
              },
          }
      }))

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
      api.properties(
          wib.Image(
              name='cipd-test',
              arch=wib.ARCH_X86,
              customizations=[
                  wib.Customization(
                      offline_winpe_customization=winpe
                      .OfflineWinPECustomization(
                          name='offline_winpe',
                          offline_customization=[
                              actions.OfflineAction(
                                  name='add files', actions=[ACTION_ADD_CIPD_1])
                          ]))
              ])) + api.post_process(StatusSuccess) +
      api.post_process(DropExpectation))

  yield (api.test(
      'Test cipd pin and download packages in single action',
      api.platform('win', 64)
  ) + api.properties(
      wib.Image(
          name='cipd-test',
          arch=wib.ARCH_X86,
          customizations=[
              wib.Customization(
                  offline_winpe_customization=winpe.OfflineWinPECustomization(
                      name='offline_winpe',
                      offline_customization=[
                          actions.OfflineAction(
                              name='add files',
                              actions=[ACTION_ADD_CIPD_1, ACTION_ADD_CIPD_2])
                      ]))
          ])) + PIN_CIPD_1_PASS + PIN_CIPD_2_PASS +
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  yield (api.test(
      'Test cipd pin and download packages in multiple actions',
      api.platform('win', 64)
  ) + api.properties(
      wib.Image(
          name='cipd-test',
          arch=wib.ARCH_X86,
          customizations=[
              wib.Customization(
                  offline_winpe_customization=winpe.OfflineWinPECustomization(
                      name='offline_winpe',
                      offline_customization=[
                          actions.OfflineAction(
                              name='add files-1',
                              actions=[ACTION_ADD_CIPD_1, ACTION_ADD_CIPD_2]),
                          actions.OfflineAction(
                              name='add files-2', actions=[ACTION_ADD_CIPD_1])
                      ]))
          ])) + PIN_CIPD_1_PASS + PIN_CIPD_2_PASS +
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

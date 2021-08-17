# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib

from recipe_engine.post_process import DropExpectation, StatusFailure
from recipe_engine.post_process import StatusSuccess, StepCommandRE

DEPS = [
    'depot_tools/gitiles',
    'windows_scripts_executor',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
]

PROPERTIES = wib.Image


def RunSteps(api, image):
  api.windows_scripts_executor.pin_wib_config(image)
  api.windows_scripts_executor.download_wib_artifacts(image)


def GenTests(api):
  PIN_FILE_STARTNET_PASS = api.step_data(
      'Pin git artifacts to refs.gitiles log: ' +
      'HEAD/windows/artifacts/startnet.cmd',
      api.gitiles.make_log_test_data('HEAD'),
  )

  FETCH_FILE_STARTNET_PASS = api.step_data(
      'Get all git artifacts.fetch ' +
      'ef70cb069518e6dc3ff24bfae7f195de5099c377:' +
      'windows/artifacts/startnet.cmd',
      api.gitiles.make_encoded_file('Wpeinit'))

  PIN_FILE_DISKPART_PASS = api.step_data(
      'Pin git artifacts to refs.gitiles log: ' +
      'HEAD/windows/artifacts/diskpart.txt',
      api.gitiles.make_log_test_data('HEAD'),
  )

  FETCH_FILE_DISKPART_PASS = api.step_data(
      'Get all git artifacts.fetch ' +
      'ef70cb069518e6dc3ff24bfae7f195de5099c377:' +
      'windows/artifacts/diskpart.txt',
      api.gitiles.make_encoded_file('select volume S'))

  # actions for adding files
  ACTION_ADD_STARTNET = wib.Action(
      add_file=wib.AddFile(
          name='add_startnet_file',
          src=wib.Src(
              git_src=wib.GITSrc(
                  repo='chromium.dev',
                  ref='HEAD',
                  src='windows/artifacts/startnet.cmd'),),
          dst='Windows\\System32',
      ))

  ACTION_ADD_DISKPART = wib.Action(
      add_file=wib.AddFile(
          name='add_diskpart_file',
          src=wib.Src(
              git_src=wib.GITSrc(
                  repo='chromium.dev',
                  ref='HEAD',
                  src='windows/artifacts/diskpart.txt'),),
          dst='Windows\\System32',
      ))

  yield (api.test('Add git src in action', api.platform('win', 64)) +
         api.properties(
             wib.Image(
                 name='win10Img',
                 arch=wib.ARCH_X86,
                 offline_winpe_customization=wib.OfflineCustomization(
                     name='offWpeCust',
                     offline_customization=[
                         wib.OfflineAction(
                             name='action-1', actions=[ACTION_ADD_STARTNET])
                     ]))) + PIN_FILE_STARTNET_PASS + FETCH_FILE_STARTNET_PASS +
         api.post_process(StatusSuccess) +  # recipe should pass
         api.post_process(DropExpectation))

  yield (api.test('Add multiple git src in action', api.platform('win', 64)) +
         api.properties(
             wib.Image(
                 name='win10Img',
                 arch=wib.ARCH_X86,
                 offline_winpe_customization=wib.OfflineCustomization(
                     name='offWpeCust',
                     offline_customization=[
                         wib.OfflineAction(
                             name='action-1',
                             actions=[ACTION_ADD_STARTNET, ACTION_ADD_DISKPART])
                     ]))) + PIN_FILE_STARTNET_PASS + FETCH_FILE_STARTNET_PASS +
         PIN_FILE_DISKPART_PASS + FETCH_FILE_DISKPART_PASS +
         api.post_process(StatusSuccess) +  # recipe should pass
         api.post_process(DropExpectation))

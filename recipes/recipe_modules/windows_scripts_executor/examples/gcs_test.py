# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import actions
from PB.recipes.infra.windows_image_builder import (offline_winpe_customization
                                                    as winpe)
from PB.recipes.infra.windows_image_builder import sources as sources
from PB.recipes.infra.windows_image_builder import (windows_image_builder as
                                                    wib)

from recipe_engine.post_process import DropExpectation, StatusFailure
from recipe_engine.post_process import StatusSuccess, StepCommandRE

DEPS = [
    'depot_tools/gsutil',
    'windows_scripts_executor',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
    'recipe_engine/path',
]

PROPERTIES = wib.Image


def RunSteps(api, image):
  api.windows_scripts_executor.module_init()
  api.windows_scripts_executor.pin_wib_config(image)
  api.windows_scripts_executor.download_wib_artifacts(image)
  api.windows_scripts_executor.execute_wib_config(image)
  api.path.mock_add_paths('[CACHE]\\WinPEImage\\media\\sources\\boot.wim')
  api.windows_scripts_executor.upload_wib_artifacts()


def GenTests(api):
  # actions for adding files
  ACTION_ADD_PING = actions.Action(
      add_file=actions.AddFile(
          name='add ping from GCS',
          src=sources.Src(
              gcs_src=sources.GCSSrc(bucket='WinTools', source='net/ping.exe'),
          ),
          dst='Windows\\System32',
      ))

  GEN_WPE_MEDIA_PASS = api.step_data(
      'execute config win10Img.offline winpe customization ' +
      'offWpeCust.Init WinPE image modification x86 in ' +
      '[CACHE]\\WinPEImage.PowerShell> Gen WinPE media for x86',
      stdout=api.json.output({'results': {
          'Success': True,
      }}))

  MOUNT_WIM_PASS = api.step_data(
      'execute config win10Img.offline winpe customization ' +
      'offWpeCust.Init WinPE image modification x86 in ' +
      '[CACHE]\\WinPEImage.PowerShell> Mount wim to ' +
      '[CACHE]\\WinPEImage\\mount',
      stdout=api.json.output({
          'results': {
              'Success': True
          },
      }))

  UMOUNT_WIM_PASS = api.step_data(
      'execute config win10Img.offline winpe customization ' +
      'offWpeCust.Deinit WinPE image modification.PowerShell> ' +
      'Unmount wim at [CACHE]\\WinPEImage\\mount',
      stdout=api.json.output({
          'results': {
              'Success': True
          },
      }))

  ADD_FILE_CIPD_PASS = api.step_data(
      'execute config win10Img.offline winpe customization ' +
      'offWpeCust.PowerShell> Add file [CACHE]\\GCSPkgs\\' +
      'WinTools\\net\\ping.exe',
      stdout=api.json.output({'results': {
          'Success': True,
      }}))

  DEINIT_WIM_ADD_CFG_TO_ROOT_PASS = api.step_data(
      'execute config win10Img.offline winpe customization ' +
      'offWpeCust.Deinit WinPE image modification.PowerShell> ' +
      'Add cfg [CLEANUP]\\configs\\configuration_key.cfg',
      stdout=api.json.output({'results': {
          'Success': True,
      }}))

  ASSERT_UPLOAD_WIM_TO_GCS = api.post_process(
      StepCommandRE, 'Upload all pending gcs artifacts.gsutil upload', [
          'python', '-u',
          'RECIPE_MODULE\[depot_tools::gsutil\]\\\\resources\\\\' +
          'gsutil_smart_retry.py', '--',
          'RECIPE_REPO\[depot_tools\]\\\\gsutil.py', '----', 'cp',
          '\[CACHE\]\\\\WinPEImage\\\\media\\\\sources\\\\boot.wim',
          'gs://chrome-gce-images/WIB-WIM/configuration_key.wim'
      ])

  yield (api.test('Add gcs src in action', api.platform('win', 64)) +
         api.properties(
             wib.Image(
                 name='win10Img',
                 arch=wib.ARCH_X86,
                 customizations=[
                     wib.Customization(
                         offline_winpe_customization=winpe
                         .OfflineWinPECustomization(
                             name='offWpeCust',
                             offline_customization=[
                                 actions.OfflineAction(
                                     name='action-1', actions=[ACTION_ADD_PING])
                             ]))
                 ])) +
         GEN_WPE_MEDIA_PASS +  # Successfully generate the winpe media
         MOUNT_WIM_PASS +  # mount the wim
         ADD_FILE_CIPD_PASS +  # copy the file to image
         DEINIT_WIM_ADD_CFG_TO_ROOT_PASS +  # add the build cfg to image
         UMOUNT_WIM_PASS +  # unmount the wim
         ASSERT_UPLOAD_WIM_TO_GCS +  # Upload the generated wim to cloud storage
         api.post_process(StatusSuccess) +  # recipe should pass
         api.post_process(DropExpectation))

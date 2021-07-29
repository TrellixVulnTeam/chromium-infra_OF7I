# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib

from recipe_engine.post_process import DropExpectation, StatusFailure
from recipe_engine.post_process import StatusSuccess, StepCommandRE

DEPS = [
    'windows_scripts_executor',
    'recipe_engine/properties',
    'recipe_engine/json',
]

PROPERTIES = wib.Image


def RunSteps(api, image):
  api.windows_scripts_executor.execute_wib_config(image)


def GenTests(api):
  # various step data for testing
  GEN_WPE_MEDIA_FAIL = api.step_data(
      'execute config win10_2013_x64.offline winpe ' +
      'customization offline_winpe_2013_x64.generate ' +
      'windows image folder for x86 in ' +
      '[CACHE]/WinPEImage.PowerShell> Gen WinPE media for x86',
      stdout=api.json.output({
          'results': {
              'Success': False,
              'Command': 'powershell',
              'ErrorInfo': {
                  'Message': 'Failed step'
              },
          }
      }))

  GEN_WPE_MEDIA_PASS = api.step_data(
      'execute config win10_2013_x64.offline winpe ' +
      'customization offline_winpe_2013_x64.generate ' +
      'windows image folder for x86 in ' +
      '[CACHE]/WinPEImage.PowerShell> Gen WinPE media for x86',
      stdout=api.json.output({'results': {
          'Success': True,
      }}))

  MOUNT_WIM_PASS = api.step_data(
      'execute config win10_2013_x64.offline winpe customization ' +
      'offline_winpe_2013_x64.generate windows image folder for ' +
      'x86 in [CACHE]/WinPEImage.PowerShell> Mount wim to ' +
      '[CACHE]/WinPEImage/mount',
      stdout=api.json.output({
          'results': {
              'Success': True
          },
      }))

  UMOUNT_WIM_PASS = api.step_data(
      'execute config win10_2013_x64.offline winpe ' +
      'customization offline_winpe_2013_x64.PowerShell> ' +
      'Unmount wim at [CACHE]/WinPEImage/mount',
      stdout=api.json.output({
          'results': {
              'Success': True
          },
      }))

  ADD_FILE_STARTNET_PASS = api.step_data(
      'execute config win10_2013_x64.offline ' +
      'winpe customization offline_winpe_2013_x64.PowerShell> ' +
      'Add file cipd_startnet_path>',
      stdout=api.json.output({'results': {
          'Success': True,
      }}))

  ADD_FILE_STARTNET_FAIL = api.step_data(
      'execute config win10_2013_x64.offline ' +
      'winpe customization offline_winpe_2013_x64.PowerShell> ' +
      'Add file cipd_startnet_path>',
      stdout=api.json.output({
          'results': {
              'Success': False,
              'Command': 'powershell',
              'ErrorInfo': {
                  'Message': 'Failed step',
              },
          }
      }))

  # Post process check for save and discard options during unmount
  UMOUNT_PP_DISCARD = api.post_process(
      StepCommandRE, 'execute config win10_2013_x64.offline winpe ' +
      'customization offline_winpe_2013_x64.PowerShell> ' +
      'Unmount wim at [CACHE]/WinPEImage/mount', [
          'python', '-u',
          'RECIPE_MODULE\[infra::powershell\]/resources/psinvoke.py',
          '--command', 'Dismount-WindowsImage', '--logs',
          '\[CLEANUP\]/Mount-WindowsImage', '--',
          '-Path "\[CACHE\]/WinPEImage/mount"',
          '-LogPath "\[CLEANUP\]/Mount-WindowsImage/unmount.log"',
          '-LogLevel WarningsInfo', '-Discard'
      ])

  UMOUNT_PP_SAVE = api.post_process(
      StepCommandRE, 'execute config win10_2013_x64.offline winpe ' +
      'customization offline_winpe_2013_x64.PowerShell> ' +
      'Unmount wim at [CACHE]/WinPEImage/mount', [
          'python', '-u',
          'RECIPE_MODULE\[infra::powershell\]/resources/psinvoke.py',
          '--command', 'Dismount-WindowsImage', '--logs',
          '\[CLEANUP\]/Mount-WindowsImage', '--',
          '-Path "\[CACHE\]/WinPEImage/mount"',
          '-LogPath "\[CLEANUP\]/Mount-WindowsImage/unmount.log"',
          '-LogLevel WarningsInfo', '-Save'
      ])

  yield (api.test('Fail win image folder creation') + api.properties(
      wib.Image(
          name='win10_2013_x64',
          offline_winpe_customization=wib.OfflineCustomization(
              name='offline_winpe_2013_x64',
              winpe_arch=wib.ARCH_X86,
              offline_customization=[
                  wib.OfflineAction(
                      name='network_setup',
                      files=[
                          wib.AddFile(
                              name='add_startnet_file',
                              src='cipd_startnet_path>',
                              dst='C:\\Windows\\System32\\startnet.cmd',
                          )
                      ])
              ]))) + GEN_WPE_MEDIA_FAIL +  # Fail to create a winpe media folder
         api.post_process(StatusFailure) +  # recipe should fail
         api.post_process(DropExpectation))

  yield (api.test('Missing arch in config') + api.properties(
      wib.Image(
          name='win10_2013_x64',
          offline_winpe_customization=wib.OfflineCustomization(
              name='offline_winpe_2013_x64',))) +
         api.post_process(StatusFailure) +  # recipe should fail
         api.post_process(DropExpectation))

  yield (api.test('Fail add file step') + api.properties(
      wib.Image(
          name='win10_2013_x64',
          offline_winpe_customization=wib.OfflineCustomization(
              name='offline_winpe_2013_x64',
              winpe_arch=wib.ARCH_X86,
              offline_customization=[
                  wib.OfflineAction(
                      name='network_setup',
                      files=[
                          wib.AddFile(
                              name='add_startnet_file',
                              src='cipd_startnet_path>',
                              dst='C:\\Windows\\System32\\startnet.cmd',
                          )
                      ])
              ]))) + GEN_WPE_MEDIA_PASS + MOUNT_WIM_PASS +
         ADD_FILE_STARTNET_FAIL +  # Fail to add file
         UMOUNT_WIM_PASS +  # Unmount the wim
         UMOUNT_PP_DISCARD +  # Discard the changes made to wim
         api.post_process(StatusFailure) +  # recipe fails
         api.post_process(DropExpectation))

  yield (api.test('Happy path') + api.properties(
      wib.Image(
          name='win10_2013_x64',
          offline_winpe_customization=wib.OfflineCustomization(
              name='offline_winpe_2013_x64',
              winpe_arch=wib.ARCH_X86,
              offline_customization=[
                  wib.OfflineAction(
                      name='network_setup',
                      files=[
                          wib.AddFile(
                              name='add_startnet_file',
                              src='cipd_startnet_path>',
                              dst='C:\\Windows\\System32\\startnet.cmd',
                          )
                      ])
              ]))) + GEN_WPE_MEDIA_PASS + MOUNT_WIM_PASS +
         ADD_FILE_STARTNET_PASS + UMOUNT_WIM_PASS +
         UMOUNT_PP_SAVE +  # Save the changes made to the wim
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

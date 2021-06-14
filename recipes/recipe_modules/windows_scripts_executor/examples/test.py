# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib

from recipe_engine.post_process import DropExpectation

DEPS = [
    'windows_scripts_executor',
    'recipe_engine/properties',
    'recipe_engine/json',
]

PROPERTIES = wib.Image


def RunSteps(api, image):
  api.windows_scripts_executor.execute_wib_config(image)


def GenTests(api):
  yield (api.test('basic') + api.properties(
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
              ]))) + api.step_data(
                  'execute config win10_2013_x64.offline winpe ' +
                  'customization offline_winpe_2013_x64.generate ' +
                  'windows image folder for amd64 in ' +
                  '[CACHE]/WinPEImage.Exec powershell',
                  stdout=api.json.output({
                      'Success': False,
                      'Command': 'pshell',
                      'ErrorInfo': {
                          'Message': 'Failed step'
                      }
                  })) + api.step_data(
                      'execute config win10_2013_x64.offline winpe ' +
                      'customization offline_winpe_2013_x64.performing ' +
                      'action network_setup.Exec powershell',
                      stdout=api.json.output({
                          "Success": False,
                          "Command": "pshell",
                          "ErrorInfo": {
                              "Message": "Failed step"
                          }
                      })) + api.post_process(DropExpectation))

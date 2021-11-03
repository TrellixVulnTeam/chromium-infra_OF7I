# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder import (offline_winpe_customization
                                                    as winpe)
from PB.recipes.infra.windows_image_builder import actions
from PB.recipes.infra.windows_image_builder import sources

from recipe_engine.post_process import DropExpectation, StatusFailure
from recipe_engine.post_process import StatusSuccess, StepCommandRE
from RECIPE_MODULES.infra.windows_scripts_executor import test_helper as t

DEPS = [
    'depot_tools/gitiles',
    'windows_scripts_executor',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
    'recipe_engine/path'
]

PROPERTIES = wib.Image


def RunSteps(api, image):
  api.windows_scripts_executor.module_init()
  api.windows_scripts_executor.pin_wib_config(image)
  api.windows_scripts_executor.save_config_to_disk(image)
  api.windows_scripts_executor.download_wib_artifacts(image)
  api.windows_scripts_executor.execute_wib_config(image)
  api.path.mock_add_paths('[CACHE]\\WinPEImage\\media\\sources\\boot.wim')
  api.windows_scripts_executor.upload_wib_artifacts()


def GenTests(api):
  image = 'general_tests'
  cust_name = 'generic_cust'
  arch = 'x86'
  key = '0f0012e3f5da9daac6c2e6017fc08c696cc36465ec4d11df75e7e1fdd1602c15'

  # actions for adding files
  ACTION_ADD_STARTNET = actions.Action(
      add_file=actions.AddFile(
          name='add_startnet_file',
          src=sources.Src(
              git_src=sources.GITSrc(
                  repo='chromium.dev',
                  ref='HEAD',
                  src='windows/artifacts/startnet.cmd'),),
          dst='Windows\\System32',
      ))

  ACTION_ADD_DOT3SVC = actions.Action(
      add_file=actions.AddFile(
          name='add winpe-dot3svc',
          src=sources.Src(
              cipd_src=sources.CIPDSrc(
                  package='infra_internal/labs/drivers/' +
                  'microsoft/windows_adk/winpe/' + 'winpe-dot3svc',
                  refs='latest',
                  platform='windows-amd64',
              ),),
          dst='Windows\\System32\\',
      ))

  # actions for installing windows packages
  ACTION_INSTALL_WMI = actions.Action(
      add_windows_package=actions.AddWindowsPackage(
          name='install_winpe_wmi',
          src=sources.Src(
              cipd_src=sources.CIPDSrc(
                  package='infra_internal/labs/drivers/' +
                  'microsoft/windows_adk/winpe/winpe-wmi',
                  refs='latest',
                  platform='windows-amd64',
              ),),
          args=['-IgnoreCheck'],
      ))

  yield (api.test('Fail win image folder creation', api.platform('win', 64)) +
         # generate image with add starnet action
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, cust_name, 'network_setup',
                         [ACTION_ADD_STARTNET])) +
         # mock pinning the file to current refs
         t.GIT_PIN_FILE(api, 'HEAD', 'windows/artifacts/startnet.cmd', 'HEAD') +
         # mock git fetch file
         t.GIT_FETCH_FILE(api, 'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                          'windows/artifacts/startnet.cmd', 'Wpeinit') +
         # mock failure in gen winpe media step
         t.GEN_WPE_MEDIA(api, arch, image, cust_name, False) +
         # The recipe execution should fail
         api.post_process(StatusFailure) + api.post_process(DropExpectation))

  yield (api.test('Missing arch in config', api.platform('win', 64)) +
         api.properties(
             wib.Image(
                 name=image,
                 customizations=[
                     wib.Customization(
                         offline_winpe_customization=winpe
                         .OfflineWinPECustomization(name=cust_name,))
                 ])) +
         # recipe execution should fail
         api.post_process(StatusFailure) + api.post_process(DropExpectation))

  yield (api.test('Fail add file step', api.platform('win', 64)) +
         # generate image with add starnet action
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, cust_name, 'network_setup',
                         [ACTION_ADD_STARTNET])) +
         # mock all the init and deinit steps
         t.MOCK_WPE_INIT_DEINIT_FAILURE(api, arch, image, cust_name) +
         # mock git pin execution
         t.GIT_PIN_FILE(api, 'HEAD', 'windows/artifacts/startnet.cmd', 'HEAD') +
         # mock the git fetch execution
         t.GIT_FETCH_FILE(api, 'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                          'windows/artifacts/startnet.cmd', 'Wpeinit') +
         # mock add file from git to image execution
         t.ADD_GIT_FILE(
             api, image, cust_name, 'ef70cb069518e6dc3ff24bfae7f195de5099c377',
             'windows\\artifacts\\startnet.cmd', False) +  # Fail to add file
         api.post_process(StatusFailure) +  # recipe fails
         api.post_process(DropExpectation))

  yield (api.test('Add file from cipd', api.platform('win', 64)) +
         # generate image with add starnet action
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, cust_name, 'network_setup',
                         [ACTION_ADD_DOT3SVC])) +
         # mock init and deinit steps for offline customization
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image, cust_name) +
         # mock add cipd file step
         t.ADD_CIPD_FILE(
             api,
             'infra_internal\\labs\\drivers\\microsoft\\windows_adk\\winpe\\' +
             'winpe-dot3svc', 'windows-amd64', image, cust_name) +
         # assert that recipe completed execution successfully
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  yield (api.test('Add file from git', api.platform('win', 64)) +
         # generate image with add starnet action
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, cust_name, 'network_setup',
                         [ACTION_ADD_STARTNET])) +
         # mock init and deinit steps for offline customization
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image, cust_name) +
         # mock git pin execution
         t.GIT_PIN_FILE(api, 'HEAD', 'windows/artifacts/startnet.cmd', 'HEAD') +
         # mock the git fetch execution
         t.GIT_FETCH_FILE(api, 'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                          'windows/artifacts/startnet.cmd', 'Wpeinit') +
         # mock add file from git step
         t.ADD_GIT_FILE(api, image, cust_name,
                        'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                        'windows\\artifacts\\startnet.cmd') +
         # assert that the recipe completed successfully
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  yield (api.test('Install package from cipd', api.platform('win', 64)) +
         # generate image with add starnet action
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, cust_name, 'wmi_setup',
                         [ACTION_INSTALL_WMI])) +
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image,
                                        cust_name) +  # mock {de}init steps
         t.INSTALL_FILE(api, 'install_winpe_wmi', image,
                        cust_name) +  # install file from cipd
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  yield (api.test('Happy path', api.platform('win', 64)) +
         # generate image with add starnet action
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, cust_name, 'network_setup',
                         [ACTION_ADD_STARTNET, ACTION_ADD_DOT3SVC])) +
         # mock init and deinit steps for offline customization
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image, cust_name) +
         # mock git pin execution
         t.GIT_PIN_FILE(api, 'HEAD', 'windows/artifacts/startnet.cmd', 'HEAD') +
         # mock the git fetch execution
         t.GIT_FETCH_FILE(api, 'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                          'windows/artifacts/startnet.cmd', 'Wpeinit') +
         # mock add file from git step
         t.ADD_GIT_FILE(api, image, cust_name,
                        'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                        'windows\\artifacts\\startnet.cmd') +
         # mock add file from cipd step
         t.ADD_CIPD_FILE(
             api, 'infra_internal\\labs\\drivers\\microsoft\\' +
             'windows_adk\\winpe\\winpe-dot3svc', 'windows-amd64', image,
             cust_name) +
         # assert that the recipe executed successfully
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

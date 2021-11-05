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

# constants for use in test
image = 'general_tests'
cust_name = 'generic_cust'
arch = 'x86'
key = '4fa9269b1b8ebc0cd8d2c1c2415374819838ffb0a4a541a601ec51749b555096'


def RunSteps(api, config):
  api.windows_scripts_executor.init(config)
  api.windows_scripts_executor.pin_available_sources()
  api.windows_scripts_executor.gen_canonical_configs(config)
  api.windows_scripts_executor.download_available_packages()
  api.windows_scripts_executor.execute_config(config)
  # mock existence of customization output to trigger upload
  api.path.mock_add_paths('[CACHE]\\Pkgs\\GCSPkgs\\chrome-gce-images\\' +
                          'WIB-WIM\\{}.wim'.format(key))
  api.windows_scripts_executor.upload_wib_artifacts()


def GenTests(api):

  # actions for adding files from git
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

  # action to copy file from cipd
  ACTION_ADD_DOT3SVC = actions.Action(
      add_file=actions.AddFile(
          name='add winpe-dot3svc',
          src=sources.Src(
              cipd_src=sources.CIPDSrc(
                  package='infra_internal/labs/drivers/' +
                  'microsoft/windows_adk/winpe/winpe-dot3svc',
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

  # generate happy path image with custom destination for upload
  HAPPY_PATH_IMAGE = t.WPE_IMAGE(image, wib.ARCH_X86, cust_name,
                                 'network_setup',
                                 [ACTION_ADD_STARTNET, ACTION_ADD_DOT3SVC])
  HAPPY_PATH_IMAGE.customizations[
      0].offline_winpe_customization.image_dest.bucket = 'test-bucket'
  HAPPY_PATH_IMAGE.customizations[
      0].offline_winpe_customization.image_dest.source = 'out/gce_winpe_rel.wim'

  # Fail the step to gen winpe media folder using copy-pe
  # https://docs.microsoft.com/en-us/windows-hardware/manufacture/desktop/copype-command-line-options
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

  # Missing arch in config. Execution should fail if arch is ARCH_UNSPECIFIED
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

  # Failure in executing action add_step
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

  # Add a file from cipd storage
  yield (api.test('Add file from cipd', api.platform('win', 64)) +
         # generate image with add dot3svc action
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

  # Add a file from git
  yield (api.test('Add file from git', api.platform('win', 64)) +
         # generate image with add starnet action
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, cust_name, 'network_setup',
                         [ACTION_ADD_STARTNET])) +
         # mock all the init and deinit steps for winpe
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image, cust_name) +
         # mock git pin file
         t.GIT_PIN_FILE(api, 'HEAD', 'windows/artifacts/startnet.cmd', 'HEAD') +
         # mock git fetch file
         t.GIT_FETCH_FILE(api, 'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                          'windows/artifacts/startnet.cmd', 'Wpeinit') +
         # mock add file to image mount dir step
         t.ADD_GIT_FILE(api, image, cust_name,
                        'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                        'windows\\artifacts\\startnet.cmd') +
         # recipe execution should pass successfully
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  # install cab file from cipd
  yield (api.test('Install package from cipd', api.platform('win', 64)) +
         # generate image with install wmi action
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, cust_name, 'wmi_setup',
                         [ACTION_INSTALL_WMI])) +
         # mock all the init and deinit steps for winpe
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image, cust_name) +
         # mock install file step
         t.INSTALL_FILE(api, 'install_winpe_wmi', image, cust_name) +
         # recipe should pass successfully
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

  # Generic happy path for recipe execution
  yield (api.test('Happy path', api.platform('win', 64)) +
         # start recipe with happy path image
         api.properties(HAPPY_PATH_IMAGE) +
         # mock all the init and deinit steps
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image, cust_name) +
         # mock git pin file
         t.GIT_PIN_FILE(api, 'HEAD', 'windows/artifacts/startnet.cmd', 'HEAD') +
         # mock git fetch file
         t.GIT_FETCH_FILE(api, 'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                          'windows/artifacts/startnet.cmd', 'Wpeinit') +
         # mock add file to image mount dir step
         t.ADD_GIT_FILE(api, image, cust_name,
                        'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                        'windows\\artifacts\\startnet.cmd') +
         # mock add file to image mount dir step
         t.ADD_CIPD_FILE(
             api, 'infra_internal\\labs\\drivers\\microsoft\\' +
             'windows_adk\\winpe\\winpe-dot3svc', 'windows-amd64', image,
             cust_name) +
         # assert that the generated image was uploaded
         t.CHECK_GCS_UPLOAD(
             api, '\[CACHE\]\\\\Pkgs\\\\GCSPkgs\\\\chrome-gce-images' +
             '\\\\WIB-WIM\\\\{}.wim'.format(key),
             'gs://chrome-gce-images/WIB-WIM/{}.wim'.format(key)) +
         # assert that the generated image was uploaded to custom dest
         t.CHECK_GCS_UPLOAD(
             api,
             '\[CACHE\]\\\\Pkgs\\\\GCSPkgs\\\\chrome-gce-images' +
             '\\\\WIB-WIM\\\\{}.wim'.format(key),
             'gs://test-bucket/out/gce_winpe_rel.wim',
             orig='gs://chrome-gce-images/WIB-WIM/{}.wim'.format(key)) +
         # recipe should pass successfully
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

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
from RECIPE_MODULES.infra.windows_scripts_executor import test_helper as t

DEPS = [
    'depot_tools/gitiles',
    'windows_scripts_executor',
    'recipe_engine/path',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
]

PROPERTIES = wib.Image

image = 'git_src_test'
customization = 'add_file_from_git'
key = '69c31bffdba451b237e80ee933b3667718166beb353bdb7c321ed167c8b51ce7'
arch = 'x86'


def RunSteps(api, config):
  api.windows_scripts_executor.init(config)
  api.windows_scripts_executor.pin_available_sources()
  api.windows_scripts_executor.gen_canonical_configs(config)
  api.windows_scripts_executor.download_available_packages()
  api.windows_scripts_executor.execute_config(config)


def GenTests(api):
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

  ACTION_ADD_DISKPART = actions.Action(
      add_file=actions.AddFile(
          name='add_diskpart_file',
          src=sources.Src(
              git_src=sources.GITSrc(
                  repo='chromium.dev',
                  ref='HEAD',
                  src='windows/artifacts/diskpart.txt'),),
          dst='Windows\\System32',
      ))

  yield (api.test('Add git src in action', api.platform('win', 64)) +
         # run a config for adding startnet file to wim
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, customization,
                         'add_startnet_file', [ACTION_ADD_STARTNET])) +
         # mock all the init and deinit steps
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image, customization) +
         # mock pin of the git src
         t.GIT_PIN_FILE(api, 'HEAD', 'windows/artifacts/startnet.cmd', 'HEAD') +
         # mock adding the file to wim
         t.ADD_GIT_FILE(api, image, customization,
                        'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                        'windows\\artifacts\\startnet.cmd') +
         api.post_process(StatusSuccess) +  # recipe should pass
         api.post_process(DropExpectation))

  # Adding same git src in multiple actions should trigger only one fetch action
  yield (
      api.test('Add same git src in multiple actions', api.platform('win',
                                                                    64)) +
      # run a config for adding startnet file to wim
      api.properties(
          t.WPE_IMAGE(image, wib.ARCH_X86, customization, 'add_startnet_file',
                      [ACTION_ADD_STARTNET, ACTION_ADD_STARTNET])) +
      # mock all the init and deinit steps
      t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image, customization) +
      # mock pin of the git src, should only happen once
      t.GIT_PIN_FILE(api, 'HEAD', 'windows/artifacts/startnet.cmd', 'HEAD') +
      # mock adding the file to wim
      t.ADD_GIT_FILE(api, image, customization,
                     'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                     'windows\\artifacts\\startnet.cmd') +
      # mock adding the file to wim
      t.ADD_GIT_FILE(api, image, customization,
                     'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                     'windows\\artifacts\\startnet.cmd (2)') +
      api.post_process(StatusSuccess) +  # recipe should pass
      api.post_process(DropExpectation))

  yield (api.test('Add multiple git src in action', api.platform('win', 64)) +
         # run a config for adding startnet and diskpart files
         api.properties(
             t.WPE_IMAGE(image, wib.ARCH_X86, customization, 'action-1',
                         [ACTION_ADD_STARTNET, ACTION_ADD_DISKPART])) +
         # mock all the init and deinit steps
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image, customization) +
         # mock pin of the git src
         t.GIT_PIN_FILE(api, 'HEAD', 'windows/artifacts/startnet.cmd', 'HEAD') +
         # mock pin of the git src
         t.GIT_PIN_FILE(api, 'HEAD', 'windows/artifacts/diskpart.txt', 'HEAD') +
         # mock adding the file to wim
         t.ADD_GIT_FILE(api, image, customization,
                        'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                        'windows\\artifacts\\startnet.cmd') +
         # mock adding the file to wim
         t.ADD_GIT_FILE(api, image, customization,
                        'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                        'windows\\artifacts\\diskpart.txt') +
         api.post_process(StatusSuccess) +  # recipe should pass
         api.post_process(DropExpectation))

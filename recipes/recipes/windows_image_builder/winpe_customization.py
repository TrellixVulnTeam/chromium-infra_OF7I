# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import post_process

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder import actions
from PB.recipes.infra.windows_image_builder import sources

from recipe_engine.post_process import DropExpectation, StatusSuccess
from RECIPE_MODULES.infra.windows_scripts_executor import test_helper as t

DEPS = [
    'depot_tools/gitiles',
    'recipe_engine/platform',
    'recipe_engine/properties',
    'recipe_engine/raw_io',
    'recipe_engine/json',
    'windows_adk',
    'windows_scripts_executor',
]

PYTHON_VERSION_COMPATIBILITY = 'PY3'

PROPERTIES = wib.Image


def RunSteps(api, image):
  """ This recipe executes offline_winpe_customization."""
  if not api.platform.is_win:
    raise AssertionError('This recipe can only run on windows')

  # this recipe will only execute the offline winpe customizations
  for cust in image.customizations:
    assert (cust.WhichOneof('customization') == 'offline_winpe_customization')

  # initialize the image to scripts executor
  api.windows_scripts_executor.init(image)

  # pinning all the refs and generating unique keys
  custs = api.windows_scripts_executor.process_customizations()

  # download all the required refs
  api.windows_scripts_executor.download_all_packages(custs)

  # download and install the windows ADK and WinPE packages
  api.windows_adk.ensure()

  # execute the customizations given
  api.windows_scripts_executor.execute_customizations(custs)


wpe_image = 'wpe_image'
wpe_cust = 'generic'
arch = 'x86'
key = '9055a3e678be47d58bb860d27b85adbea41fd2ef3e22c5b7cb3180edf358de90'


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

  yield (api.test('not_run_on_windows', api.platform('linux', 64)) +
         api.expect_exception('AssertionError') +
         api.post_process(DropExpectation))

  yield (api.test('happy path', api.platform('win', 64)) + api.properties(
      t.WPE_IMAGE(wpe_image, wib.ARCH_X86, wpe_cust, 'happy test',
                  [ACTION_ADD_STARTNET])) +
         # mock all the init and deinit steps
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, wpe_image, wpe_cust) +
         # mock git pin file
         t.GIT_PIN_FILE(api, wpe_image, wpe_cust, 'HEAD',
                        'windows/artifacts/startnet.cmd', 'HEAD') +
         # mock add file to wpe_image mount dir step
         t.ADD_GIT_FILE(api, wpe_image, wpe_cust,
                        'ef70cb069518e6dc3ff24bfae7f195de5099c377',
                        'windows\\artifacts\\startnet.cmd') +
         # assert that the generated wpe_image was uploaded
         t.CHECK_GCS_UPLOAD(
             api, wpe_image, wpe_cust,
             '\[CLEANUP\]\\\\{}\\\\workdir\\\\gcs.zip'.format(wpe_cust),
             'gs://chrome-gce-images/WIB-WIM/{}.zip'.format(key)) +
         api.post_process(StatusSuccess) + api.post_process(DropExpectation))

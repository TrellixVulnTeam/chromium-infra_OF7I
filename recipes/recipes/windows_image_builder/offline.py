# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import post_process

from PB.recipes.infra.windows_image_builder import input as input_pb
from PB.recipes.infra.windows_image_builder import windows_image_builder as wib

DEPS = [
    'depot_tools/bot_update',
    'depot_tools/gclient',
    'recipe_engine/context',
    'recipe_engine/file',
    'recipe_engine/json',
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/properties',
    'recipe_engine/step',
    'windows_adk',
    'windows_scripts_executor',
]

PROPERTIES = input_pb.Inputs

def RunSteps(api, inputs):
  """This recipe runs windows offline builder for a given user config."""
  if not api.platform.is_win:
    raise AssertionError("This recipe only runs on Windows")

  if not inputs.config_path:
    raise api.step.StepFailure("`config_path` is a required property")

  builder_named_cache = api.path['cache'].join('builder')
  config = None
  with api.step.nest('read user config'):
    # download the configs repo
    api.gclient.set_config('infradata_config')
    api.gclient.c.solutions[0].revision = 'origin/main'
    with api.context(cwd=builder_named_cache):
      api.bot_update.ensure_checkout()
      api.gclient.runhooks()
      # split the string on '/' as luci scheduler passes a unix path and this
      # recipe is expected to run on windows ('\')
      cfg_path = builder_named_cache.join('infra-data-config',
                                          *inputs.config_path.split('/'))
      config = api.file.read_proto(
          name='Reading ' + inputs.config_path,
          source=cfg_path,
          msg_class=wib.Image,
          codec='TEXTPB')

  # Ensure windows adk is installed
  api.windows_adk.ensure()
  api.windows_scripts_executor.execute_wib_config(config)


def GenTests(api):
  # Step data for use in tests
  STEP_INSTALL_ADK_PASS = api.step_data(
      'ensure windows adk present.PowerShell> Install ADK',
      stdout=api.json.output({
          'results': {
              'Success': True
          },
          '[CLEANUP]\\logs\\adk\\adk.log': 'i007: Exit code: 0x0',
      }))

  STEP_INSTALL_WINPE_PASS = api.step_data(
      'ensure win-pe add-on present.PowerShell> Install WinPE',
      stdout=api.json.output({
          'results': {
              'Success': True
          },
          '[CLEANUP]\\logs\\winpe\\winpe.log': 'i007: Exit code: 0x0',
      }))

  yield (api.test('basic', api.platform('win', 64)) +
         api.properties(input_pb.Inputs(config_path="test_config")) +
         STEP_INSTALL_ADK_PASS + STEP_INSTALL_WINPE_PASS +
         api.post_process(post_process.StatusFailure)
        )  # fails as config is empty

  yield (
      api.test('not_run_on_windows', api.platform('linux', 64)) +
      api.properties(
          input_pb.Inputs(
              config_path="test_config",
          ),
      ) +
      api.expect_exception('AssertionError'))

  yield (api.test('run_without_config_path', api.platform('win', 64)) +
         api.properties(input_pb.Inputs(config_path="",),) +
         api.post_process(post_process.StatusFailure))

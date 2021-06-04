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
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/properties',
    'recipe_engine/step',
    'windows_adk',
]

PROPERTIES = input_pb.Inputs

def RunSteps(api, inputs):
  """This recipe runs windows offline builder for a given user config."""
  if not api.platform.is_win:
    raise AssertionError("This recipe only runs on Windows")

  if not inputs.config_path:
    raise api.step.StepFailure("`config_path` is a required property")

  with api.step.nest('ensure windows adk present'):
    api.windows_adk.ensure()

  builder_named_cache = api.path['cache'].join('builder')
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
      api.file.read_proto(name='Reading '+inputs.config_path, source=cfg_path,
                          msg_class=wib.Image, codec='TEXTPB')

def GenTests(api):

  yield api.test('basic', api.platform('win', 64)) + api.properties(
      input_pb.Inputs(
          config_path="test_config"))

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

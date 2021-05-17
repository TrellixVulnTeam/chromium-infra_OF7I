# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import post_process

from PB.recipes.infra.windows_image_builder import input as input_pb

DEPS = [
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

  with api.step.nest('ensure windows adk present') as r:
    api.windows_adk.ensure()

  with api.step.nest('read user config') as r:
    r.logs['read user config'] = 'succeed'


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

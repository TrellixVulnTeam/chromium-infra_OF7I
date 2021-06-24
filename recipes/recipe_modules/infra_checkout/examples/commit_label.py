# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEPS = [
    'recipe_engine/buildbucket',
    'recipe_engine/properties',
    'recipe_engine/step',

    'depot_tools/bot_update',

    'infra_checkout',
]


def RunSteps(api):
  co = api.infra_checkout.checkout(gclient_config_name='infra')
  api.step('label', ['echo', co.get_commit_label()])


def GenTests(api):
  yield api.test(
      'with_commit_position',
      api.buildbucket.ci_build(
          project='infra',
          bucket='ci',
          git_repo='https://chromium.googlesource.com/infra/infra',
      )
  )

  yield api.test(
      'without_commit_position',
      api.buildbucket.ci_build(
          project='infra',
          bucket='ci',
          git_repo='https://chromium.googlesource.com/infra/infra',
      ),
      api.step_data(
          'bot_update',
          api.bot_update.output_json(
              root='infra',
              first_sln='infra',
              revision_mapping={'got_revision': 'infra'},
              commit_positions=False,
          ),
      )
  )

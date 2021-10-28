# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Recipe to test LUCI CQ/CV itself."""

from PB.recipes.infra.cv_testing import tryjob as pb

PYTHON_VERSION_COMPATIBILITY = 'PY2+3'

DEPS = [
  'recipe_engine/cq',
  'recipe_engine/properties',
  'recipe_engine/step',
]

PROPERTIES = pb.Input


def RunSteps(api, properties):
  api.step('1 step per recipe keeps a recipe engine crash away', cmd=None)
  if properties.reuse_own_mode_only:
    api.cq.allow_reuse_for(api.cq.run_mode)
  if properties.fail:
    raise api.step.StepFailure('tryjob wants to be red')
  if properties.infra_fail:
    raise api.step.InfraFailure('tryjob wants to be purple')


def GenTests(api):
  def test(name, *args):
    return api.test(
        name,
        api.cq(run_mode=api.cq.DRY_RUN),
        *args)

  yield test(
      'any-reuse',
  )
  yield test(
      'reuse-by-the-same-mode-only',
      api.properties(reuse_own_mode_only=True),
  )
  yield test(
      'fail',
      api.properties(fail=True),
  )
  yield test(
      'infra_fail',
      api.properties(infra_fail=True),
  )

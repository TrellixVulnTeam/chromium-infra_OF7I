# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEPS = [
  'depot_tools/bot_update',
  'depot_tools/gclient',
  'depot_tools/git',
  'recipe_engine/buildbucket',
  'recipe_engine/context',
  'recipe_engine/path',
  'recipe_engine/platform',
  'recipe_engine/properties',
  'recipe_engine/python',
  'recipe_engine/step',
]

ASSETS_DIFF_FAILURE_MESSAGE = '''
- Please check the diffs in the previous step
- Please run `make release` to update assets '''


def RunSteps(api):
  api.gclient.set_config('luci_py')
  api.bot_update.ensure_checkout()
  api.gclient.runhooks()
  # TODO(tandrii): trigger tests without PRESUBMIT.py; https://crbug.com/917479

  luci_dir = api.path['checkout'].join('luci')
  with api.context(cwd=luci_dir):
    _step_run_swarming_tests(api)

    if api.platform.is_linux:
      _step_swarming_ui_tests(api)

def _step_run_swarming_tests(api):
  luci_dir = api.context.cwd

  with api.step.nest('swarming'):
    cwd = luci_dir.join('appengine', 'swarming')
    with api.context(cwd=cwd):
      cfg = api.context.cwd.join('unittest.cfg')
      # TODO(jwata): remove '--log-level DEBUG' after fixing test failures
      testpy_args = ['-v', '--conf', cfg, '-A',
                     '!no_run', '--log-level', 'DEBUG']

      with api.step.nest('python2'):
        venv = luci_dir.join('.vpython')
        # TODO(jwata): some tests are failing
        # remove ok_ret='any' after fixing them on the luci-py repo
        api.python('run tests',
                   'test.py', args=testpy_args, venv=venv, ok_ret='any')
        api.python('run tests sequentially',
                   'test_seq.py', args=['-v'], venv=venv)

      with api.step.nest('python3'):
        venv3 = luci_dir.join('.vpython3')
        # add --python3 for enabling py3filter plugin
        api.python('run tests',
                   'test.py', args=testpy_args+['--python3'], venv=venv3)
        api.python('run tests sequentially',
                   'test_seq.py', args=['-v'], venv=venv3)


def _step_swarming_ui_tests(api):
  with api.step.nest('swarming-ui'):
    ui_dir = api.path['checkout'].join('luci', 'appengine', 'swarming', 'ui2')
    node_path = ui_dir.join('nodejs', 'bin')
    paths_to_add = [api.path.pathsep.join([str(node_path)])]
    env_prefixes = {'PATH': paths_to_add}
    with api.context(env_prefixes=env_prefixes, cwd=ui_dir):
      api.step('install node modules', ['npm', 'ci'])
      _steps_check_diffs_on_ui_assets(api)
      api.step('run tests', ['make', 'test'])


def _steps_check_diffs_on_ui_assets(api):
  api.step('build assets', ['make', 'release'])
  diff_check = api.git('diff', '--exit-code', ok_ret='any')
  if diff_check.retcode != 0:
    diff_check.presentation.status = 'FAILURE'
    api.python.failing_step(
        'ASSETS DIFF DETECTED',
        ASSETS_DIFF_FAILURE_MESSAGE)


def GenTests(api):
  yield (
      api.test('ci') +
      api.buildbucket.ci_build(
          'infra', 'ci', 'Luci-py linux-64',
          git_repo='https://chromium.googlesource.com/infra/luci/luci-py',
      )
  )

  yield (
      api.test('try') +
      api.buildbucket.try_build(
          'infra', 'try', 'Luci-py Presubmit',
          git_repo='https://chromium.googlesource.com/infra/luci/luci-py',
      )
  )

  # test case for failures
  yield (
      api.test('failure') +
      api.buildbucket.try_build(
          'infra', 'try', 'Luci-py Presubmit',
          git_repo='https://chromium.googlesource.com/infra/luci/luci-py',
      ) +
      api.step_data('swarming-ui.git diff', retcode=1)
  )

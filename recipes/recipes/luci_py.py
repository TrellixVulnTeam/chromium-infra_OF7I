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

  luci_dir = api.path['checkout'].join('luci')
  with api.context(cwd=luci_dir):
    appeng_dir = luci_dir.join('appengine')

    if api.platform.is_linux:
      with api.step.nest('auth_service'):
        _step_run_py_tests(api, appeng_dir.join('auth_service'))

      with api.step.nest('config_service'):
        _step_run_py_tests(api, appeng_dir.join('config_service'))

      with api.step.nest('components'):
        components_dir = appeng_dir.join('components')
        _step_run_py_tests(api, components_dir)
        _step_run_py_tests(api, components_dir, python3=True)

      with api.step.nest('isolate'):
        _step_run_py_tests(api, appeng_dir.join('isolate'))

    # TODO(crbug.com/1017545): enable on Windows and Mac.
    # clients tests run in python2/3, but it ignores failures on Windows
    # in python2, and ignores failures on Windows and Mac in python3.
    with api.step.nest('client'):
      ok_ret = 'any' if api.platform.is_win else (0,)
      _step_run_py_tests(api, luci_dir.join('client'), ok_ret=ok_ret)
      ok_ret = (0,) if api.platform.is_linux else 'any'
      _step_run_py_tests(api, luci_dir.join('client'), ok_ret=ok_ret,
                         python3=True)

    with api.step.nest('swarming bot'):
      bot_dir = appeng_dir.join('swarming', 'swarming_bot')
      # TODO(crbug.com/1019105): remove this timeout.
      if api.platform.is_mac:
        timeout = 120
      else:
        timeout = None
      _step_run_py_tests(api, bot_dir, timeout=timeout)
      # TODO(crbug.com/1017545): enable python3 on Windows and Mac.
      # swarming bot tests run in python3, but it ignores failures on Windows
      # and Mac.
      ok_ret = (0,) if api.platform.is_linux else 'any'
      _step_run_py_tests(api, bot_dir, timeout=timeout, ok_ret=ok_ret,
                         python3=True)

    # swarming server
    if api.platform.is_linux:
      with api.step.nest('swarming'):
        _step_run_py_tests(api, appeng_dir.join('swarming'))

    # swarming ui
    if api.platform.is_linux:
      _step_swarming_ui_tests(api)


def _step_run_py_tests(api, cwd, python3=False, timeout=None, ok_ret=(0,)):
  luci_dir = api.context.cwd
  with api.context(cwd=cwd):
    cfg = api.context.cwd.join('unittest.cfg')
    testpy_args = ['-v', '--conf', cfg]

    if python3:
      venv = luci_dir.join('.vpython3')
      py = 'python3'
    else:
      venv = luci_dir.join('.vpython')
      py = 'python2'

    api.python('run tests %s' % py, 'test.py', args=testpy_args, venv=venv,
               timeout=timeout, ok_ret=ok_ret)


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
          'infra', 'try', 'Luci-py Linux',
          git_repo='https://chromium.googlesource.com/infra/luci/luci-py',
      )
  )

  yield (
      api.test('try-mac') +
      api.platform.name('mac') +
      api.buildbucket.try_build(
          'infra', 'try', 'Luci-py Mac',
          git_repo='https://chromium.googlesource.com/infra/luci/luci-py',
      )
  )

  yield (
      api.test('try-win') +
      api.platform.name('win') +
      api.buildbucket.try_build(
          'infra', 'try', 'Luci-py Windows',
          git_repo='https://chromium.googlesource.com/infra/luci/luci-py',
      )
  )

  # test case for failures
  yield (
      api.test('try-failure') +
      api.buildbucket.try_build(
          'infra', 'try', 'Luci-py Presubmit',
          git_repo='https://chromium.googlesource.com/infra/luci/luci-py',
      ) +
      api.step_data('swarming-ui.git diff', retcode=1)
  )

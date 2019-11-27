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
    # auth server
    if api.platform.is_linux:
      _step_run_tests(api, 'auth_service',
                      luci_dir.join('appengine', 'auth_service'))

    # config server
    if api.platform.is_linux:
      _step_run_tests(api, 'config_service',
                      luci_dir.join('appengine', 'config_service'))

    # components
    if api.platform.is_linux:
      _step_run_tests(api, 'components',
                      luci_dir.join('appengine', 'components'),
                      run_test_seq=True,
                      run_python3=True)

    # isolate server
    if api.platform.is_linux:
      _step_run_tests(api, 'isolate',
                      luci_dir.join('appengine', 'isolate'))

    # TODO(crbug.com/1017545): enable python3 on windows and mac
    # client
    ok_ret = (0,)
    if not api.platform.is_linux:
      ok_ret = 'any'
    _step_run_tests(api, 'client',
                    luci_dir.join('client'),
                    run_test_seq=True,
                    run_python3=True,
                    ok_ret=ok_ret)

    # TODO(crbug.com/1019105): remove this timeout.
    if api.platform.is_mac:
      timeout = 60
    else:
      timeout = None

    # TODO(crbug.com/1017545): enable python3 on windows and mac
    # swarming bot
    _step_run_tests(api, 'swarming bot',
                    luci_dir.join('appengine', 'swarming', 'swarming_bot'),
                    run_test_seq=True,
                    run_python3=api.platform.is_linux,
                    timeout=timeout)

    # swarming server
    if api.platform.is_linux:
      _step_run_tests(api, 'swarming',
                      luci_dir.join('appengine', 'swarming'),
                      run_test_seq=True)

    # swarming ui
    if api.platform.is_linux:
      _step_swarming_ui_tests(api)


def _step_run_tests(
    api, name, cwd, run_test_seq=False, run_python3=False, timeout=None,
    ok_ret=(0,)):
  luci_dir = api.context.cwd
  with api.step.nest(name):
    with api.context(cwd=cwd):
      cfg = api.context.cwd.join('unittest.cfg')
      testpy_args = ['-v', '--conf', cfg, '-A', '!no_run']

      if run_python3:
        # python3
        venv3 = luci_dir.join('.vpython3')
        api.python('run tests python3',
                   'test.py', args=testpy_args, venv=venv3,
                   timeout=timeout, ok_ret=ok_ret)
        if run_test_seq:
          api.python('run tests seq python3',
                     'test_seq.py', args=['-v'], venv=venv3,
                     timeout=timeout, ok_ret=ok_ret)

      # python2
      venv = luci_dir.join('.vpython')
      api.python('run tests python2',
                 'test.py', args=testpy_args, venv=venv,
                 timeout=timeout, ok_ret=ok_ret)
      if run_test_seq:
        api.python('run tests seq python2',
                   'test_seq.py', args=['-v'], venv=venv,
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

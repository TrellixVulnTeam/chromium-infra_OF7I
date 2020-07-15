# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEPS = [
    'infra_checkout',
    'depot_tools/bot_update',
    'depot_tools/gclient',
    'depot_tools/git',
    'recipe_engine/buildbucket',
    'recipe_engine/context',
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/python',
    'recipe_engine/raw_io',
    'recipe_engine/step',
]

ASSETS_DIFF_FAILURE_MESSAGE = '''
- Please check the diffs in the previous step
- Please run `make release` to update assets '''


def RunSteps(api):
  co = api.infra_checkout.checkout('luci_py', patch_root='infra/luci')
  co.gclient_runhooks()

  luci_dir = api.path['checkout'].join('luci')
  with api.context(cwd=luci_dir):
    appeng_dir = luci_dir.join('appengine')

    with api.step.nest('check changes') as presentation:
      changes = _check_changes(api)
      presentation.logs['changes'] = [
          '%s: %s' % (p, j) for p, j in changes.items()
      ]

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

    # TODO(crbug.com/1017545): enable on Windows.
    # clients tests run in python2/3, but it ignores failures on Windows.
    with api.step.nest('client'):
      ok_ret = 'any' if api.platform.is_win else (0,)
      _step_run_py_tests(api, luci_dir.join('client'), ok_ret=ok_ret)
      _step_run_py_tests(api, luci_dir.join('client'), ok_ret=ok_ret,
                         python3=True)

    with api.step.nest('swarming bot'):
      bot_dir = appeng_dir.join('swarming', 'swarming_bot')
      _step_run_py_tests(api, bot_dir)
      # TODO(crbug.com/1017545): enable python3 on Windows.
      # swarming bot tests run in python3, but it ignores failures on Windows.
      ok_ret = 'any' if api.platform.is_win else (0,)
      _step_run_py_tests(api, bot_dir, ok_ret=ok_ret, python3=True)

    # swarming server
    if api.platform.is_linux:
      with api.step.nest('swarming'):
        _step_run_py_tests(api, appeng_dir.join('swarming'))

    # swarming ui
    if api.platform.is_linux:
      _step_swarming_ui_tests(api, changes)


def _check_changes(api):
  return {
      'DEPS':
          _has_changed_files(api, 'DEPS'),
      'client':
          _has_changed_files(api, 'client'),
      'auth_service':
          _has_changed_files(api, 'appengine/auth_service'),
      'config_service':
          _has_changed_files(api, 'appengine/config_service'),
      'components':
          _has_changed_files(api, 'appengine/components'),
      'isolate':
          _has_changed_files(api, 'appengine/isolate'),
      'swarming':
          _has_changed_files(
              api, 'appengine/swarming', exclude_dir='appengine/swarming/ui2'),
      'swarming_ui':
          _has_changed_files(api, 'appengine/swarming/ui2'),
  }


def _has_changed_files(api, subdir, exclude_dir=None):
  result = api.m.git(
      'diff',
      '--name-only',
      '--cached',
      subdir,
      name='get change list on %s' % subdir,
      stdout=api.m.raw_io.output())
  files = result.stdout.splitlines()

  # exclude files if exclude_dir is specified.
  if exclude_dir:
    filtered = []
    for f in files:
      if not f.startswith(exclude_dir):
        filtered.append(f)
    files = filtered

  result.presentation.logs['change list'] = sorted(files)
  return len(files) > 0


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


def _step_swarming_ui_tests(api, changes):
  deps = ['DEPS', 'swarming_ui']
  if not any([changed for k, changed in changes.items() if k in deps]):
    # skip tests when no changes on the dependencies.
    return
  with api.step.nest('swarming-ui'):
    ui_dir = api.path['checkout'].join('luci', 'appengine', 'swarming', 'ui2')
    node_path = ui_dir.join('nodejs', 'bin')
    paths_to_add = [api.path.pathsep.join([str(node_path)])]
    env_prefixes = {'PATH': paths_to_add}
    with api.context(env_prefixes=env_prefixes, cwd=ui_dir):
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

  yield (api.test('try') + api.buildbucket.try_build(
      'infra',
      'try',
      'Luci-py Linux',
      git_repo='https://chromium.googlesource.com/infra/luci/luci-py',
  ) + api.step_data(
      'check changes.get change list on appengine/swarming',
      api.raw_io.stream_output(
          '\n'.join([
              'appengine/swarming/foo.py',
              'appengine/swarming/ui2/bar.js',
          ]),
          stream='stdout'),
  ) + api.step_data(
      'check changes.get change list on appengine/swarming/ui2',
      api.raw_io.stream_output(
          '\n'.join([
              'appengine/swarming/ui2/bar.js',
          ]), stream='stdout'),
  ))

  yield (api.test('try-mac') + api.platform.name('mac') +
         api.buildbucket.try_build(
             'infra',
             'try',
             'Luci-py Mac',
             git_repo='https://chromium.googlesource.com/infra/luci/luci-py',
         ))

  yield (
      api.test('try-win') +
      api.platform.name('win') +
      api.buildbucket.try_build(
          'infra', 'try', 'Luci-py Windows',
          git_repo='https://chromium.googlesource.com/infra/luci/luci-py',
      )
  )

  # test case for failures
  yield (api.test('try-ui-diff-check-failure') + api.buildbucket.try_build(
      'infra',
      'try',
      'Luci-py Presubmit',
      git_repo='https://chromium.googlesource.com/infra/luci/luci-py',
  ) + api.step_data(
      'check changes.get change list on appengine/swarming/ui2',
      api.raw_io.stream_output(
          '\n'.join([
              'appengine/swarming/ui2/bar.js',
          ]), stream='stdout'),
  ) + api.step_data('swarming-ui.git diff', retcode=1))

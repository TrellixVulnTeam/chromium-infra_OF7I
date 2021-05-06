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
  with api.context(cwd=api.path['start_dir']):
    if api.platform.is_win:
      # Need to enable support for symlinks on Windows for unittest in luci-py
      api.git(
          'config', '--global', 'core.symlinks', 'true', name='set symlinks')
  co = api.infra_checkout.checkout('luci_py', patch_root='infra/luci')
  co.gclient_runhooks()

  luci_dir = api.path['checkout'].join('luci')
  with api.context(cwd=luci_dir):

    with api.step.nest('check changes') as presentation:
      changes = _check_changes(api)
      presentation.logs['changes'] = [
          '%s: %s' % (p, j) for p, j in changes.items()
      ]

    _step_auth_tests(api, changes)

    _step_config_tests(api, changes)

    _step_components_tests(api, changes)

    _step_isolate_tests(api, changes)

    _step_client_tests(api, changes)

    _step_swarming_bot_tests(api, changes)

    _step_swarming_tests(api, changes)

    _step_swarming_ui_tests(api, changes)


def _check_changes(api):
  return {
      'DEPS':
          _has_changed_files(api, 'DEPS'),
      'vpython':
          _has_changed_files(api, '.vpython'),
      'vpython3':
          _has_changed_files(api, '.vpython3'),
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
      'appengine_third_party':
          _has_changed_files(api, 'appengine/third_party'),
  }


def _has_changed_files(api, path, exclude_dir=None):
  result = api.m.git(
      'diff',
      '--name-only',
      '--cached',
      path,
      name='get change list on %s' % path,
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


def _step_run_py_tests(api, cwd, python3=False, timeout=None):
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

    api.python(
        'run tests %s' % py,
        'test.py',
        args=testpy_args,
        venv=venv,
        timeout=timeout)


def _step_auth_tests(api, changes):
  if not api.platform.is_linux:
    return

  deps = ['auth_service', 'components', 'vpython', 'appengine_third_party']
  if not any([changes[d] for d in deps]):
    # skip tests when no changes on the dependencies.
    return

  auth_dir = api.path['checkout'].join('luci', 'appengine', 'auth_service')
  with api.step.nest('auth_service'):
    _step_run_py_tests(api, auth_dir)


def _step_components_tests(api, changes):
  if not api.platform.is_linux:
    return

  deps = ['components', 'vpython', 'appengine_third_party']
  if not any([changes[d] for d in deps]):
    # skip tests when no changes on the dependencies.
    return

  components_dir = api.path['checkout'].join('luci', 'appengine', 'components')
  with api.step.nest('components'):
    _step_run_py_tests(api, components_dir)
    _step_run_py_tests(api, components_dir, python3=True)


def _step_config_tests(api, changes):
  if not api.platform.is_linux:
    return

  deps = ['config_service', 'components', 'vpython', 'appengine_third_party']
  if not any([changes[d] for d in deps]):
    # skip tests when no changes on the dependencies.
    return

  config_dir = api.path['checkout'].join('luci', 'appengine', 'config_service')
  with api.step.nest('config_service'):
    _step_run_py_tests(api, config_dir)


def _step_isolate_tests(api, changes):
  if not api.platform.is_linux:
    return

  deps = ['isolate', 'client', 'components', 'appengine_third_party']
  if not any([changes[d] for d in deps]):
    # skip tests when no changes on the dependencies.
    return

  isolate_dir = api.path['checkout'].join('luci', 'appengine', 'isolate')
  with api.step.nest('isolate'):
    _step_run_py_tests(api, isolate_dir)


def _step_client_tests(api, changes):
  deps = ['client', 'components', 'vpython3', 'vpython']
  if not any([changes[d] for d in deps]):
    # skip tests when no changes on the dependencies.
    return

  luci_dir = api.path['checkout'].join('luci')
  with api.step.nest('client'):
    _step_run_py_tests(api, luci_dir.join('client'))
    _step_run_py_tests(api, luci_dir.join('client'), python3=True)


def _step_swarming_tests(api, changes):
  deps = [
      'swarming', 'client', 'components', 'vpython3', 'vpython',
      'appengine_third_party'
  ]
  if not any([changes[d] for d in deps]):
    # skip tests when no changes on the dependencies.
    return
  if not api.platform.is_linux:
    # the server runs only on linux.
    return

  swarming_dir = api.path['checkout'].join('luci', 'appengine', 'swarming')
  with api.step.nest('swarming'):
    _step_run_py_tests(api, swarming_dir)


def _step_swarming_bot_tests(api, changes):
  deps = ['swarming', 'client', 'components', 'vpython3', 'vpython']
  if not any([changes[d] for d in deps]):
    # skip tests when no changes on the dependencies.
    return

  bot_dir = api.path['checkout'].join('luci', 'appengine', 'swarming',
                                      'swarming_bot')
  with api.step.nest('swarming bot'):
    _step_run_py_tests(api, bot_dir)
    _step_run_py_tests(api, bot_dir, python3=True)


def _step_swarming_ui_tests(api, changes):
  if not api.platform.is_linux:
    return
  deps = ['DEPS', 'swarming_ui']
  if not any([changes[d] for d in deps]):
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

  def _ci_build():
    return api.buildbucket.ci_build(
        project='infra',
        git_repo='https://chromium.googlesource.com/infra/luci/luci-py')

  def _try_build():
    return api.buildbucket.try_build(
        project='infra',
        git_repo='https://chromium.googlesource.com/infra/luci/luci-py')

  def _step_data_changed_files(directory, files):
    return api.step_data(
        'check changes.get change list on %s' % directory,
        api.raw_io.stream_output('\n'.join(files)),
        stream='stdout')

  yield (api.test('ci') + _ci_build() +
         _step_data_changed_files('client', ['client/foo.py']) +
         _step_data_changed_files('appengine/auth_service',
                                  ['appengine/auth_service/foo.py']) +
         _step_data_changed_files('appengine/config_service',
                                  ['appengine/config_service/foo.py']) +
         _step_data_changed_files('appengine/components',
                                  ['appengine/components/foo.py']) +
         _step_data_changed_files('appengine/isolate',
                                  ['appengine/isolate/foo.py']) +
         _step_data_changed_files('appengine/swarming',
                                  ['appengine/swarming/foo.py']) +
         _step_data_changed_files('appengine/swarming/ui2',
                                  ['appengine/swarming/ui2/foo.js']))

  yield (api.test('try') + _try_build() +
         _step_data_changed_files('client', ['client/foo.py']) +
         _step_data_changed_files('appengine/auth_service',
                                  ['appengine/auth_service/foo.py']) +
         _step_data_changed_files('appengine/config_service',
                                  ['appengine/config_service/foo.py']) +
         _step_data_changed_files('appengine/components',
                                  ['appengine/components/foo.py']) +
         _step_data_changed_files('appengine/isolate',
                                  ['appengine/isolate/foo.py']) +
         _step_data_changed_files(
             'appengine/swarming',
             ['appengine/swarming/foo.py', 'appengine/swarming/ui2/bar.js']) +
         _step_data_changed_files('appengine/swarming/ui2',
                                  ['appengine/swarming/ui2/bar.js']))

  yield (api.test('try-mac') + api.platform.name('mac') + _try_build() +
         _step_data_changed_files('client', ['client/foo.py']))

  yield (api.test('try-win') + api.platform.name('win') + _try_build() +
         _step_data_changed_files('client', ['client/foo.py']))

  # test case for failures
  yield (
    api.test('try-ui-diff-check-failure') + _try_build() +
    _step_data_changed_files(
        'appengine/swarming/ui2',
        ['appengine/swarming/ui2/bar.js']) +
    api.step_data('swarming-ui.git diff', retcode=1))

  # test case for skipping test steps.
  yield api.test('try-skip') + _try_build()

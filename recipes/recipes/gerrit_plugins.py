# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEPS = [
    'zip',
    'recipe_engine/buildbucket',
    'recipe_engine/file',
    'recipe_engine/platform',
    'recipe_engine/step',
    'recipe_engine/path',
    'recipe_engine/cipd',
    'recipe_engine/context',
    'depot_tools/bot_update',
    'depot_tools/gclient',
    'depot_tools/gsutil',
]

PYTHON_VERSION_COMPATIBILITY = 'PY3'


def _getNode(api):
  with api.step.nest('get node'):
    packages_dir = api.path['start_dir'].join('packages')
    ensure_file = api.cipd.EnsureFile()
    ensure_file.add_package('infra/nodejs/nodejs/${platform}',
                            'node_version:12.13.0')
    api.cipd.ensure(packages_dir, ensure_file)
    return api.path['start_dir'].join('packages', 'bin')


def _getChrome(api):
  with api.step.nest('get chrome'):
    chrome = api.path.mkdtemp(prefix='chrome')
    gs_bucket = 'chromium-browser-snapshots'
    gs_path = 'Linux_x64'
    version_file = 'LAST_CHANGE'
    chrome_zip = 'chrome-linux.zip'
    api.gsutil.download(gs_bucket, '%s/%s' % (gs_path, version_file), chrome)
    version = api.file.read_text('read latest chrome version',
                                 chrome.join(version_file))
    api.gsutil.download(gs_bucket, '%s/%s/%s' % (gs_path, version, chrome_zip),
                        chrome)
    api.zip.unzip('unzip chrome', chrome.join(chrome_zip), chrome.join('zip'))
    return chrome.join('zip', 'chrome-linux')


def _getBazel(api):
  with api.step.nest('get bazel'):  # pragma: no cover
    bazel_path = api.path.mkdtemp(prefix='bazel')
    bazel_bin = bazel_path.join('bazel')
    api.gsutil.download('bazel', '4.2.1/release/bazel-4.2.1-linux-x86_64',
                        bazel_bin)
    api.step('make bazel executable', ['chmod', '+x', bazel_bin])
    return bazel_path


def RunSteps(api):
  assert api.platform.is_linux, 'Unsupported platform, only Linux is supported.'
  cl = api.buildbucket.build.input.gerrit_changes[0]
  project_name = cl.project
  assert project_name.startswith('infra/gerrit-plugins/'), (
      'unknown project: "%s"' % project_name)
  plugin = project_name[len('infra/gerrit-plugins/'):]
  test_name = 'gerrit_plugins_%s' % plugin.replace('-', '_')
  api.gclient.set_config(test_name)
  api.bot_update.ensure_checkout(patch_root=project_name)
  test_dir = api.path['start_dir'].join(test_name)

  node_path = _getNode(api)

  chrome_path = _getChrome(api)
  chrome_bin = chrome_path.join('chrome')

  # Check if the plugin uses JavaScript or TypeScript.
  # TODO(crbug.com/1196790): Clean up once all plugins use TypeScript.
  tsconfig_path = test_dir.join('web', 'tsconfig.json')
  if api.path.exists(tsconfig_path):  # pragma: no cover
    # Karma requires the binary be named chromium, not chrome when running
    # ChromiumHeadless.
    chromium_bin = chrome_path.join('chromium')
    api.step('rename to chromium', ['mv', chrome_bin, chromium_bin])

    # TypeScript plugin tests require that the plugin be located within the
    # Gerrit repo. Move and rename the plugin.
    plugins_dir = None
    with api.step.nest('set up plugin layout'):
      plugins_dir = api.path['start_dir'].join('gerrit', 'plugins')
      api.step('move test repo', ['mv', test_dir, plugins_dir])
      api.step('rename test repo',
               ['mv', plugins_dir.join(test_name), plugins_dir.join(plugin)])

    bazel_path = _getBazel(api)

    env = {
        'CHROMIUM_BIN':
            str(chromium_bin),
        'PATH':
            api.path.pathsep.join(
                [str(bazel_path),
                 str(node_path),
                 str(chrome_path), '%(PATH)s']),
    }

    with api.context(env=env, cwd=plugins_dir.join(plugin)):
      api.step('run karma tests', [
          'bazel', 'test', '--test_output=all', 'web:karma_test',
          '--test_arg=ChromiumHeadless'
      ])
  else:
    env = {
        'LAUNCHPAD_CHROME': chrome_bin,
        'PATH': api.path.pathsep.join([str(node_path), '%(PATH)s'])
    }

    with api.context(env=env, cwd=test_dir):
      api.step('npm install', ['npm', 'install'])
      api.step('run wct tests', ['npx', 'wct', '--expanded'])


def GenTests(api):
  for plugin in (
      'binary-size',
      'buildbucket',
      'chromium-behavior',
      'chromium-binary-size',
      'chromium-style',
      'chumpdetector',
      'code-coverage',
      'git-numberer',
      'landingwidget',
      'tricium'):
    yield (
      api.test(plugin) +
      api.platform.name('linux') +
      api.buildbucket.try_build(project='infra/gerrit-plugins/%s' % plugin)
    )

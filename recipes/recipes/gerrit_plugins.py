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


def RunSteps(api):
  assert api.platform.is_linux, 'Unsupported platform, only Linux is supported.'
  cl = api.buildbucket.build.input.gerrit_changes[0]
  project_name = cl.project
  assert project_name.startswith('infra/gerrit-plugins/'), (
      'unknown project: "%s"' % project_name)
  api.gclient.set_config('gerrit_plugins')
  api.bot_update.ensure_checkout(patch_root=project_name)

  # Get node from CIPD.
  packages_dir = api.path['start_dir'].join('packages')
  ensure_file = api.cipd.EnsureFile()
  ensure_file.add_package('infra/nodejs/nodejs/${platform}',
                          'node_version:12.13.0')
  api.cipd.ensure(packages_dir, ensure_file)
  node_path = api.path['start_dir'].join('packages', 'bin')

  # Get the latest version of Chrome from Google Storage.
  gs_dir = api.path.mkdtemp(prefix='gs_dir')
  gs_bucket = 'chromium-browser-snapshots'
  gs_path = 'Linux_x64'
  version_file = 'LAST_CHANGE'
  chrome_zip = 'chrome-linux.zip'
  api.gsutil.download(gs_bucket, '%s/%s' % (gs_path, version_file), gs_dir)
  version = api.file.read_text('read latest chrome version',
                               gs_dir.join(version_file))
  api.gsutil.download(gs_bucket, '%s/%s/%s' % (gs_path, version, chrome_zip),
                      gs_dir)
  api.zip.unzip('unzip chrome', gs_dir.join(chrome_zip), gs_dir.join('zip'))
  chrome_path = gs_dir.join('zip', 'chrome-linux', 'chrome')

  env = {
      'LAUNCHPAD_CHROME': chrome_path,
      'PATH': api.path.pathsep.join([str(node_path), '%(PATH)s'])
  }

  with api.context(env=env, cwd=api.path['start_dir'].join('gerrit_plugins')):
    # TODO(gavinmak) Support typescript plugins and tests.
    api.step('npm install', ['npm', 'install'])
    api.step('run wct tests', ['npx', 'wct', '--expanded'])


def GenTests(api):
  yield (api.test('linux') + api.platform.name('linux') +
         api.buildbucket.try_build(project='infra/gerrit-plugins/tricium'))

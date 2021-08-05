# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from contextlib import contextmanager

from recipe_engine.recipe_api import Property
from recipe_engine.config import List

DEPS = [
    'depot_tools/gclient',
    'depot_tools/git',
    'depot_tools/windows_sdk',
    'depot_tools/osx_sdk',
    'depot_tools/tryserver',
    'recipe_engine/buildbucket',
    'recipe_engine/context',
    'recipe_engine/file',
    'recipe_engine/json',
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/properties',
    'recipe_engine/python',
    'recipe_engine/raw_io',
    'recipe_engine/step',
]

PROPERTIES = {
    'platforms':
        Property(
            help=('The platforms to build wheels for. On Windows, required to '
                  'be set to a list of either 32-bit or 64-bit platforms. '
                  'For other platforms, if empty, builds for all '
                  'platforms which are supported on this host.'),
            kind=List(str),
            default=(),
        ),
    'dry_run':
        Property(
            help='If true, do not upload wheels or source to CIPD.',
            kind=bool,
            default=False,
        ),
    'rebuild':
        Property(
            help=("If true, build all wheels regardless of whether they're "
                  "already in CIPD"),
            kind=bool,
            default=False,
        ),
}


def RunSteps(api, platforms, dry_run, rebuild):
  solution_path = api.path['cache'].join('builder', 'build_wheels')
  api.file.ensure_directory("init cache if it doesn't exist", solution_path)

  ref = 'origin/main'
  if api.tryserver.is_tryserver:
    ref = api.tryserver.gerrit_change_fetch_ref

  with api.context(cwd=solution_path):
    api.gclient.set_config('infra')
    api.gclient.c.solutions[0].revision = ref
    api.gclient.checkout(timeout=10 * 60)
    api.gclient.runhooks()

  # DISTUTILS_USE_SDK and MSSdk are necessary for distutils to correctly locate
  # MSVC on Windows. They do nothing on other platforms, so we just set them
  # unconditionally.
  with PlatformSdk(api, platforms), api.context(
      cwd=solution_path.join('infra'),
      env={
          'DISTUTILS_USE_SDK': '1',
          'MSSdk': '1',
      }):
    wheels = None
    if api.tryserver.is_tryserver:
      files = api.git(
          '-c',
          'core.quotePath=false',
          'diff',
          '--name-only',
          'HEAD~',
          name='git diff to find changed files',
          stdout=api.raw_io.output()).stdout.split()
      assert (files != [])
      # Avoid rebuilding everything if only the wheel specs have changed.
      if all(api.path.basename(p) in {'wheels.py', 'wheels.md'} for p in files):
        run_wheel_json = lambda step_name: \
          api.step(step_name,
                   ['vpython', '-vpython-spec', '.vpython',
                    '-m', 'infra.tools.dockerbuild', 'wheel-json'],
                   stdout=api.json.output()).stdout

        new_wheels = run_wheel_json('compute new wheels.json')

        patch_commit = api.git(
            'rev-parse', 'HEAD', stdout=api.raw_io.output()).stdout.strip()
        api.git('checkout', 'HEAD~', name='git checkout previous revision')

        old_wheels = run_wheel_json('compute old wheels.json')

        api.git('checkout', patch_commit, name='git checkout back to HEAD')

        wheels = []
        for wheel in new_wheels:
          if wheel not in old_wheels:
            spec = wheel['spec']

            # Compute the tag in the same way as in dockerbuild's Spec.tag.
            tag = '%s-%s' % (spec['name'], spec['version'])
            if spec['version_suffix']:
              tag += spec['version_suffix']
            if spec['pyversions']:
              tag += '-' + '.'.join(sorted(spec['pyversions']))
            wheels.append(tag)

    temp_path = api.path.mkdtemp('.dockerbuild')
    args = [
        '--root',
        temp_path,
    ]
    if not dry_run:
      args.append('--upload-sources')
    args.append('wheel-build')
    if not dry_run:
      args.append('--upload')

    if rebuild:
      args.append('--rebuild')

    for p in platforms:
      args.extend(['--platform', p])

    if wheels is not None:
      # If this is a wheel config-only change, but there are no new or changed
      # wheels, then don't bother running dockerbuild. This is the case if
      # there's a non-functional change in wheels.py, or if we removed wheels.
      if wheels == []:
        return

      for wheel in wheels:
        args.extend(['--wheel', wheel])

    api.step('dockerbuild', [
        'vpython', '-vpython-spec', '.vpython', '-m', 'infra.tools.dockerbuild'
    ] + args)


@contextmanager
def PlatformSdk(api, platforms):
  sdk = None
  if api.platform.is_win:
    is_64bit = all((p.startswith('windows-x64') for p in platforms))
    is_32bit = all((p.startswith('windows-x86') for p in platforms))
    if is_64bit == is_32bit:
      raise ValueError(
          'Must specify either 32-bit or 64-bit windows platforms.')
    target_arch = 'x64' if is_64bit else 'x86'
    sdk = api.windows_sdk(target_arch=target_arch)
  elif api.platform.is_mac:
    sdk = api.osx_sdk('mac')

  if sdk is None:
    yield
  else:
    with sdk:
      yield


def GenTests(api):
  yield api.test('success')
  yield api.test(
      'win',
      api.platform('win', 64) +
      api.properties(platforms=['windows-x64', 'windows-x64-py3']))
  yield api.test(
      'win-x86',
      api.platform('win', 64) +
      api.properties(platforms=['windows-x86', 'windows-x86-py3']))
  yield api.test('mac', api.platform(
      'mac', 64)) + api.properties(platforms=['mac-x64', 'mac-x64-cp38'])
  yield api.test('dry-run', api.properties(dry_run=True))

  # Can't build 32-bit and 64-bit Windows wheels on the same invocation.
  yield api.test(
      'win-32and64bit',
      api.platform('win', 64) +
      api.properties(platforms=['windows-x64', 'windows-x86']) +
      api.expect_exception('ValueError'))
  # Must explicitly specify the platforms to build on Windows.
  yield api.test('win-noplatforms',
                 api.platform('win', 64) + api.expect_exception('ValueError'))

  yield api.test(
      'trybot non-wheels file CL',
      api.properties(dry_run=True, rebuild=True) +
      api.buildbucket.try_build('infra') +
      api.tryserver.gerrit_change_target_ref('refs/branch-heads/foo') +
      api.override_step_data(
          'git diff to find changed files',
          stdout=api.raw_io.output('infra/tools/dockerbuild/wheel_wheel.py')))

  yield api.test(
      'trybot wheels only CL',
      api.properties(dry_run=True, rebuild=True) +
      api.buildbucket.try_build('infra') +
      api.tryserver.gerrit_change_target_ref('refs/branch-heads/foo') +
      api.override_step_data(
          'git diff to find changed files',
          stdout=api.raw_io.output('infra/tools/dockerbuild/wheels.py')) +
      api.override_step_data(
          'compute old wheels.json',
          stdout=api.json.output([{
              "spec": {
                  "name": "old-wheel",
                  "patch_version": None,
                  "pyversions": ["py3"],
                  "version": "3.2.0",
                  "version_suffix": None,
              }
          }])) + api.override_step_data(
              'compute new wheels.json',
              stdout=api.json.output([{
                  "spec": {
                      "name": "entirely-new",
                      "patch_version": 'chromium.1',
                      "pyversions": ["py3"],
                      "version": "3.3.0",
                      "version_suffix": ".chromium.1",
                  }
              }])))

  yield api.test(
      'trybot wheel removed CL',
      api.properties(dry_run=True, rebuild=True) +
      api.buildbucket.try_build('infra') +
      api.tryserver.gerrit_change_target_ref('refs/branch-heads/foo') +
      api.override_step_data(
          'git diff to find changed files',
          stdout=api.raw_io.output('infra/tools/dockerbuild/wheels.py')) +
      api.override_step_data(
          'compute old wheels.json',
          stdout=api.json.output([{
              "spec": {
                  "name": "old-wheel",
                  "patch_version": None,
                  "pyversions": ["py3"],
                  "version": "3.2.0",
                  "version_suffix": None,
              }
          }])) + api.override_step_data(
              'compute new wheels.json', stdout=api.json.output([])))

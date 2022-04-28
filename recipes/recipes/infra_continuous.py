# Copyright 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

PYTHON_VERSION_COMPATIBILITY = "PY2+3"

DEPS = [
    'depot_tools/bot_update',
    'depot_tools/depot_tools',
    'depot_tools/gclient',
    'depot_tools/osx_sdk',
    'recipe_engine/buildbucket',
    'recipe_engine/context',
    'recipe_engine/file',
    'recipe_engine/json',
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/properties',
    'recipe_engine/python',
    'recipe_engine/resultdb',
    'recipe_engine/runtime',
    'recipe_engine/step',
    'infra_checkout',
    'infra_cipd',
]


# Mapping from a builder name to a list of GOOS-GOARCH variants it should build
# CIPD packages for. 'native' means "do not cross-compile, build for the host
# platform". Targeting 'native' will also usually build non-go based packages.
#
# Additionally, a variant may have a sequence of options appended to it,
# separated by colons. e.g. 'VARIANT:option:option'. Currently the supported
# options are:
#   * 'test' - Run the tests. By default no tests are run.
#   * 'legacy' - Switch the builder that builds this variant to use go "legacy"
#     Go version 1.15.* instead of the "bleeding_edge" version. This is
#     primarily needed by builders targeting OSX amd64 that need to produce
#     binaries that can run on OSX 10.10 and OSX 10.11. What exact versions
#     correspond to "legacy" and "bleeding_edge" is defined in bootstrap.py in
#     infra.git. Note that this option applies to the entire builder (not only
#     the individual build variant).
#
# If the builder is not in this set, or the list of GOOS-GOARCH for it is empty,
# it won't be used for building CIPD packages.
#
# Only builders named '*-packager-*' builders will actually upload CIPD
# packages, while '*-continuous-*' builders merely verify that CIPD packages can
# be built.
#
# TODO(iannucci): make packager role explicit with `package=cipd_prefix` option.
# TODO(iannucci): remove this dict and put this all configuration as explicit
#    property inputs to the recipe :)
CIPD_PACKAGE_BUILDERS = {
    # trusty-64 is the primary builder for linux-amd64, and the rest just
    # cross-compile to different platforms (to speed up the overall cycle time
    # by doing stuff in parallel).
    'infra-continuous-trusty-64': [
        'native:test',
        'linux-386',
    ],
    'infra-continuous-xenial-64': [
        'linux-arm64',
        'linux-mips64',
        'linux-mips64le',
    ],
    'infra-continuous-bionic-64': [
        'linux-mipsle',
        'linux-ppc64',
        'linux-s390x',
    ],

    # 10.13 is the primary builder for darwin-amd64.
    'infra-continuous-mac-10.13-64': ['native:test:legacy',],
    'infra-continuous-mac-10.14-64': [],
    'infra-continuous-mac-10.15-64': ['darwin-arm64',],

    # Windows 64 bit builder runs and tests for both 64 && 32 bit.
    'infra-continuous-win10-64': [
        'native:test',
        'windows-386:test',
    ],
    'infra-continuous-win11-64': [
        'native:test',
        'windows-386:test',
    ],

    # Internal builders, they use exact same recipe.
    'infra-internal-continuous-trusty-64': [
        'native:test',
        'linux-arm',
    ],
    'infra-internal-continuous-xenial-64': [
        'linux-arm64',
        'darwin-arm64',  # note: can't do it on mac-10.15 since we need >=go1.16
    ],
    'infra-internal-continuous-win-64': [
        'native:test',
        'windows-386:test',
    ],
    'infra-internal-continuous-mac-11-64': ['native:test:legacy',],

    # Builders that upload CIPD packages.
    #
    # In comments is approximate runtime for building and testing packages, per
    # platform (as of Feb 23 2021). We try to balance xc1 and xc2.
    'infra-packager-linux-64': [
        'native',  # ~120 sec
    ],
    'infra-packager-linux-xc1': [
        'linux-386',  # ~60 sec
        'linux-arm',  # ~60 sec
        'linux-arm64',  # ~60 sec
    ],
    'infra-packager-linux-xc2': [
        'linux-mips64',  # ~40 sec
        'linux-mips64le',  # ~40 sec
        'linux-mipsle',  # ~40 sec
        'linux-ppc64',  # ~40 sec
        'linux-ppc64le',  # ~40 sec
        'linux-s390x',  # ~40 sec
        'aix-ppc64',  # ~5 sec
    ],
    'infra-packager-mac-64': [
        'native:legacy',  # ~150 sec
        'darwin-arm64',  # ~60 sec
    ],
    'infra-packager-win-64': [
        'native',  # ~60 sec
        'windows-386',  # ~100 sec
    ],
    'infra-internal-packager-linux-64': [
        'native',  # ~60 sec
        'linux-arm',  # ~30 sec
        'linux-arm64',  # ~30 sec
        'darwin-arm64',  # ~30 sec (note: need go 1.16)
    ],
    'infra-internal-packager-mac-64': [
        'native:legacy',  # ~40 sec
    ],
    'infra-internal-packager-win-64': [
        'native',  # ~60 sec
        'windows-386',  # ~40 sec
    ],
}


INTERNAL_REPO = 'https://chrome-internal.googlesource.com/infra/infra_internal'
PUBLIC_REPO = 'https://chromium.googlesource.com/infra/infra'


def RunSteps(api):

  buildername = api.buildbucket.builder_name
  if (buildername.startswith('infra-internal-continuous') or
      buildername.startswith('infra-internal-packager')):
    project_name = 'infra_internal'
    repo_url = INTERNAL_REPO
  elif (buildername.startswith('infra-continuous') or
      buildername.startswith('infra-packager')):
    project_name = 'infra'
    repo_url = PUBLIC_REPO
  else:  # pragma: no cover
    raise ValueError(
        'This recipe is not intended for builder %s. ' % buildername)

  # Use the latest bleeding edge version of Go unless asked for the legacy one.
  go_version_variant = 'bleeding_edge'
  for variant in CIPD_PACKAGE_BUILDERS.get(buildername, []):
    if 'legacy' in variant.split(':'):
      go_version_variant = 'legacy'
      break

  co = api.infra_checkout.checkout(
      gclient_config_name=project_name,
      internal=(project_name == 'infra_internal'),
      generate_env_with_system_python=True,
      go_version_variant=go_version_variant)
  co.gclient_runhooks()

  # Whatever is checked out by bot_update. It is usually equal to
  # api.buildbucket.gitiles_commit.id except when the build was triggered
  # manually (commit id is empty in that case).
  rev = co.bot_update_step.presentation.properties['got_revision']
  build_main(api, co, buildername, project_name, repo_url, rev)


def build_main(api, checkout, buildername, project_name, repo_url, rev):
  is_packager = 'packager' in buildername

  # Do not run python tests on packager builders, since most of them are
  # irrelevant to the produced packages. Relevant portion of tests will be run
  # from api.infra_cipd.test() below, when testing packages that pack python
  # code.
  if api.platform.arch != 'arm' and not is_packager:
    run_python_tests(api, project_name)

  # Some third_party go packages on OSX rely on cgo and thus a configured
  # clang toolchain.
  with api.osx_sdk('mac'), checkout.go_env():
    if not is_packager:
      with api.depot_tools.on_path():
        # Some go tests test interactions with depot_tools binaries, so put
        # depot_tools on the path.
        api.step(
            'infra go tests',
            api.resultdb.wrap(
                ['vpython', '-u', api.path['checkout'].join('go', 'test.py')]))

    for plat in CIPD_PACKAGE_BUILDERS.get(buildername, []):
      options = plat.split(':')
      plat = options.pop(0)

      if plat == 'native':
        goos, goarch = None, None
      else:
        goos, goarch = plat.split('-', 1)

      with api.infra_cipd.context(api.path['checkout'], goos, goarch):
        if api.platform.is_mac:
          api.infra_cipd.build_without_env_refresh(
              api.properties.get('signing_identity'))
        else:
          api.infra_cipd.build_without_env_refresh()
        if 'test' in options:
          api.infra_cipd.test()
        if is_packager:
          if api.runtime.is_experimental:
            api.step('no CIPD package upload in experimental mode', cmd=None)
          else:
            api.infra_cipd.upload(api.infra_cipd.tags(repo_url, rev))


def run_python_tests(api, project_name):
  with api.step.defer_results():
    with api.context(cwd=api.path['checkout']):
      # Run Linux tests everywhere, Windows tests only on public CI.
      if api.platform.is_linux or project_name == 'infra':
        api.python('infra python tests', 'test.py', ['test'])

      # Validate ccompute configs.
      if api.platform.is_linux and project_name == 'infra_internal':
        api.python(
            'ccompute config test',
            'ccompute/scripts/ccompute_config.py', ['test'])


def GenTests(api):

  def test(name, builder, repo, project, bucket, plat, is_experimental=False,
           arch='intel'):
    return (api.test(name) + api.platform(plat, 64, arch) +
            api.runtime(is_experimental=is_experimental) +
            api.buildbucket.ci_build(
                project, bucket, builder, git_repo=repo, build_number=123))

  yield test('public-ci-linux', 'infra-continuous-trusty-64',
             PUBLIC_REPO, 'infra', 'ci', 'linux')
  yield test('public-ci-linux-arm64', 'infra-continuous-trusty-64',
             PUBLIC_REPO, 'infra', 'ci', 'linux', arch='arm')
  yield test('public-ci-win', 'infra-continuous-win10-64',
             PUBLIC_REPO, 'infra', 'ci', 'win')

  yield test('internal-ci-linux', 'infra-internal-continuous-trusty-64',
             INTERNAL_REPO, 'infra-internal', 'ci', 'linux')
  yield test('internal-ci-mac', 'infra-internal-continuous-mac-11-64',
             INTERNAL_REPO, 'infra-internal', 'ci', 'mac')

  yield test('public-packager-mac', 'infra-packager-mac-64',
             PUBLIC_REPO, 'infra', 'prod', 'mac')
  yield test('public-packager-mac_experimental', 'infra-packager-mac-64',
             PUBLIC_REPO, 'infra', 'prod', 'mac',
             is_experimental=True)
  yield test('public-packager-mac_codesign', 'infra-packager-mac-64',
             PUBLIC_REPO, 'infra', 'prod', 'mac') + api.properties(
                 signing_identity='AAAAAAAAAAAAABBBBBBBBBBBBBXXXXXXXXXXXXXX')

  yield test('internal-packager-linux', 'infra-internal-packager-linux-64',
             INTERNAL_REPO, 'infra-internal', 'prod', 'linux')

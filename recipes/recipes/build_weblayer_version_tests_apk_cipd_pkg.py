# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import contextlib
import re

DEPS = [
  'build/chromium',
  'build/zip',
  'depot_tools/bot_update',
  'depot_tools/gclient',
  'depot_tools/git',
  'depot_tools/git_cl',
  'recipe_engine/cipd',
  'recipe_engine/context',
  'recipe_engine/file',
  'recipe_engine/json',
  'recipe_engine/path',
  'recipe_engine/python',
  'recipe_engine/url'
]

CL_DESC = """weblayer: Add skew tests for new versions released in beta

Skew tests are being added for the following versions:
%s

R=rmhasan@google.com, estaab@chromium.org
Bug:1041619
"""

# CIPD package path.
# https://chrome-infra-packages.appspot.com/p/chromium/testing/weblayer-x86/+/
CIPD_PKG_NAME='chromium/testing/weblayer-x86'

CHROMIUM_VERSION_REGEX = r'\d+\.\d+\.\d+\.\d+$'

CHROMIUMDASH = 'https://chromiumdash.appspot.com'
FETCH_RELEASES = '/fetch_releases?'
FETCH_COMMIT = '/fetch_commit?'

# Number of releases to fetch
NUM_RELEASES = 20

# Weblayer variants.pyl identifierr templates
WEBLAYER_NTH_TMPL = 'WEBLAYER_%s_SKEW_TESTS_NTH_MILESTONE'
WEBLAYER_NTH_MINUS_ONE_TMPL = (
    'WEBLAYER_%s_SKEW_TESTS_NTH_MINUS_ONE_MILESTONE')
WEBLAYER_NTH_MINUS_TWO_TMPL = (
    'WEBLAYER_%s_SKEW_TESTS_NTH_MINUS_TWO_MILESTONE')

# Client and Implementation templates for skew test configurations
CLIENT = 'CLIENT'
IMPL = 'IMPL'
CLIENT_ARGS_TMPL = """'args': [
  '--test-runner-outdir',
  '.',
  '--client-outdir',
  '../../weblayer_instrumentation_test_M{milestone}/out/Release',
  '--implementation-outdir',
  '.',
  '--test-expectations',
  '../../weblayer/browser/android/javatests/skew/expectations.txt',
  '--client-version={milestone}',
],
"""
IMPL_ARGS_TMPL = """'args': [
  '--test-runner-outdir',
  '.',
  '--client-outdir',
  '.',
  '--implementation-outdir',
  '../../weblayer_instrumentation_test_M{milestone}/out/Release',
  '--test-expectations',
  '../../weblayer/browser/android/javatests/skew/expectations.txt',
  '--impl-version={milestone}',
],
"""
CLIENT_ID_TMPL = "'identifier': 'M{milestone}_Client_Library_Tests',"
IMPL_ID_TMPL = "'identifier': 'M{milestone}_Implementation_Library_Tests',"

# Swarming arguments templates for all skew tests
SWARMING_ARGS_TMPL = """'swarming': {{
  'cipd_packages': [
    {{
      'cipd_package': 'chromium/testing/weblayer-x86',
      'location': 'weblayer_instrumentation_test_M{milestone}',
      'revision': 'version:{chromium_version}',
    }}
  ],
}},
"""


def generate_skew_test_config_lines(library, version):
  lines = []
  milestone = version[:version.index('.')]
  args_tmpl = CLIENT_ARGS_TMPL if library == CLIENT else IMPL_ARGS_TMPL
  id_tmpl = CLIENT_ID_TMPL if library == CLIENT else IMPL_ID_TMPL
  INDENT_SIZE = 4
  lines.extend(
      ' ' * INDENT_SIZE + v.rstrip()
      for v in  args_tmpl.format(milestone=milestone).splitlines())
  lines.append(' ' * INDENT_SIZE + id_tmpl.format(milestone=milestone))
  lines.extend(
      ' ' * INDENT_SIZE + v.rstrip()
      for v in SWARMING_ARGS_TMPL.format(
          milestone=milestone, chromium_version=version).splitlines())
  return lines


def releases_url(platform, channel, num, api):
  return CHROMIUMDASH + FETCH_RELEASES + api.url.urlencode(
      {'platform': platform, 'channel': channel, 'num': num})


def commit_url(hash_value, api):
  return CHROMIUMDASH + FETCH_COMMIT + api.url.urlencode(
      {'commit': hash_value})


def get_chromium_version(api, hash_value):
  commit_info = api.url.get_json(
      commit_url(hash_value, api),
      step_name='Fetch information on commit %s' % hash_value).output
  version =  commit_info['deployment']['beta']
  assert re.match(CHROMIUM_VERSION_REGEX, version)
  return version


def is_higher_version(version, query_version):
  for p1, p2 in zip(version.split('.'), query_version.split('.')):
    if int(p2) > int(p1):
      return True
  return False


@contextlib.contextmanager
def checkout_chromium_version_and_sync_3p_repos(api, version):
  api.git('checkout', version)
  api.git('clean', '-fd')
  try:
    check_correct_version(api, version)
    src_cfg = api.gclient.make_config('android')
    api.gclient.sync(src_cfg)
    api.gclient.runhooks()
    yield
  finally:
    api.git('checkout', 'origin/master')
    api.git('clean', '-fd')
    # TODO(rmhasan): Use api.gn.clean to clean up old build files
    api.file.rmtree(
        'Cleaning %s build files' % version,
        api.path['checkout'].join('out', 'Release'))


def check_correct_version(api, version):
  version_file = api.path['checkout'].join('chrome', 'VERSION')
  lines = api.file.read_text(
      'Reading //chrome/VERSION', version_file).splitlines()
  src_version = '.'.join(line[line.index('=') + 1:] for line in lines)
  if not re.match(CHROMIUM_VERSION_REGEX, src_version):
    raise ValueError("Chromium version, '%s', is not in proper format" %
                     src_version)
  assert version == src_version, (
      'Version %s does not match src/ version %s' % (version, src_version))


def get_chromium_versions_to_add(api):
  # Get Chromium project commits released into the beta channel
  # Get at least the last 20 commits released so that we can possibly
  # get 3 milestone versions below
  beta_releases = api.url.get_json(
      releases_url('Android', 'Beta', NUM_RELEASES, api),
      step_name='Getting Android beta channel releases').output
  # Convert hashes into chromium version numbers
  chromium_versions = [get_chromium_version(
                           api, beta_release['hashes']['chromium'])
                       for beta_release in beta_releases]

  # Map milestone number to milestone version numbers
  milestones_to_versions = {}
  for version in chromium_versions:
    milestone = int(version[:version.index('.')])
    milestones_to_versions.setdefault(milestone, '0.0.0.0')
    if is_higher_version(milestones_to_versions[milestone], version):
      milestones_to_versions[milestone] = version

  # Sort milestone versions by milestone number
  recent_milestone_versions = [
      milestones_to_versions[milestone] for milestone in
      sorted(milestones_to_versions, reverse=True)]
  return recent_milestone_versions


def maybe_update_variants_pyl(api, variants_lines, variants_pyl_path):
  # Get recent milestone versions
  recent_milestone_versions = get_chromium_versions_to_add(api)

  # Map Chromium versions to variants.pyl identifiers
  versions_to_variants_id_tmpl = zip(
      recent_milestone_versions,
      [WEBLAYER_NTH_TMPL, WEBLAYER_NTH_MINUS_ONE_TMPL,
       WEBLAYER_NTH_MINUS_TWO_TMPL])

  # TODO(crbug.com/1041619): Add presubmit check for variants.pyl
  # that checks if variants.pyl follows a format which allows the code
  # below to overwrite skew test configurations
  new_variants_lines = []
  cipd_pkgs_to_create = set()
  lineno = 0
  while lineno < len(variants_lines):
    for version, tmpl in versions_to_variants_id_tmpl:
      for library in [CLIENT, IMPL]:
        variants_id = tmpl % library
        if variants_id in variants_lines[lineno]:
          new_variants_lines.append(variants_lines[lineno])
          new_variants_lines.extend(
              generate_skew_test_config_lines(library, version))
          contains_current_version = False
          open_bracket_count = 1
          lineno += 1
          while open_bracket_count:
            contains_current_version |= version in variants_lines[lineno]
            for c in variants_lines[lineno]:
              if c == '{':
                open_bracket_count += 1
              elif c == '}':
                open_bracket_count -= 1
            if open_bracket_count:
              lineno += 1
          if not contains_current_version:
            cipd_pkgs_to_create.add(version)
    new_variants_lines.append(variants_lines[lineno])
    lineno += 1

  if cipd_pkgs_to_create:
    # Build CIPD packages for new versions that were released in beta
    build_cipd_pkgs(api, cipd_pkgs_to_create)
    # Upload changes to variants.pyl to Gerrit
    upload_changes(api, new_variants_lines, variants_pyl_path,
                   cipd_pkgs_to_create)


def build_cipd_pkgs(api, cipd_pkgs_to_create):
  # Need to save mb_config.pyl for skew tests APK committed at ToT
  curr_path = api.path['checkout'].join(
     'weblayer', 'browser', 'android', 'javatests',
     'skew', 'mb_config.pyl')
  mb_config_path = api.path.mkdtemp().join('mb_config.pyl')
  api.file.copy('Copying mb_config.pyl from main branch',
                curr_path, mb_config_path)
  extract_dir = api.path['cleanup'].join('apk_build_files')

  # Create CIPD packages for all chromium versions added to variants.pyl
  for version in cipd_pkgs_to_create:

    # Checkout chromium version
    with api.context(cwd=api.path['checkout']), \
      checkout_chromium_version_and_sync_3p_repos(api, version):
      zip_path = api.path.mkdtemp().join('build_config.zip')
      if api.path.exists(str(extract_dir)):
        api.file.rmtree('Cleaning up build files from last build', extract_dir)
      cipd_out = api.path['checkout'].join(
          'out', 'Release', 'skew_test_apk_%s.cipd' % version.replace('.', '_'))

      # Generate build files for weblayer instrumentation tests APK - x86
      mb_path = api.path['checkout'].join('tools', 'mb')
      mb_py_path = mb_path.join('mb.py')
      mb_args = [
         'zip', '--master=dummy.master', '--builder=dummy.builder',
         #'--goma-dir', str(api.path['cache'].join('goma', 'client')),
         #'--luci-auth',
         '--config-file=%s' % str(mb_config_path), 'out/Release',
         'weblayer_instrumentation_test_apk', str(zip_path)]
      api.python('Generating build files for weblayer_instrumentation_test_apk',
                 mb_py_path, mb_args)

      # Build weblayer instrumentation tests APK - x86 CIPD package
      api.zip.unzip('Uncompressing build files for version %s' % version,
                    zip_path, extract_dir)
      api.cipd.build(extract_dir, cipd_out, CIPD_PKG_NAME)
      api.cipd.register(CIPD_PKG_NAME, cipd_out, tags={'version': version},
                        refs=['m%s' % version[:version.index('.')]])


def upload_changes(api, new_variants_lines, variants_pyl_path,
                   cipd_pkgs_to_create):
  # New chromium versions were added to variants.pyl so we need to write
  # new changes to src/testing/buildbot/variants.pyl and commit them to the
  # main branch
  api.git('branch', '-D', 'new-skew-tests', ok_ret='any')
  api.git.new_branch('new-skew-tests')
  api.file.write_text(
      'Write to variants.py', variants_pyl_path,
      '\n'.join(new_variants_lines))
  api.python('Running generate_buildbot_json.py',
             api.path['checkout'].join(
                 'testing', 'buildbot', 'generate_buildbot_json.py'))
  api.git('commit', '-am', 'Adding skew tests for new versions')
  description = CL_DESC % '\n'.join(
      '%d, %s' % (idx + 1, ver)
      for idx, ver in enumerate(cipd_pkgs_to_create))
  # TODO(rmhasan): Add code to automatically submit CL
  api.git_cl.upload(description, upload_args=['--force'])


def RunSteps(api):
  # Set Gclient config to android
  api.gclient.set_config('chromium')
  # Turn it into a Android checkout
  api.gclient.apply_config('android')
  # Checkout chromium/src at ToT
  api.bot_update.ensure_checkout(with_tags=True)

  # TODO(rmhasan): Add back goma compilation
  # Ensure that goma is installed
  #api.chromium.set_config('android', TARGET_PLATFORM='android')
  #api.chromium.ensure_goma()
  #api.goma.start()

  # Set up git config
  api.git('config', 'user.name', 'Weblayer Skew Tests Version Updates',
          name='set git config user.name')
  # Configure git cl
  api.git_cl.set_config('basic')
  api.git_cl.c.repo_location = api.path['checkout']
  # Checkout origin.master
  # Also's cd's into src/
  api.git('checkout', 'origin/master')

  # Read variants.pyl
  variants_pyl_path = api.path['checkout'].join(
      'testing', 'buildbot', 'variants.pyl')
  variants_lines = api.file.read_text(
      'Read variants.pyl', variants_pyl_path).splitlines()

  maybe_update_variants_pyl(api, variants_lines, variants_pyl_path)


def GenTests(api):
  TEST_VARIANTS_PYL = """{
  'WEBLAYER_CLIENT_SKEW_TESTS_NTH_MILESTONE': {
    'args': [
      '--test-runner-outdir',
      '.',
    ],
    'swarming': {
      'cipd_packages': [
        {
          'cipd_package': 'chromium/testing/weblayer-x86',
          'revision': 'version:83.0.4103.56',
        },
      ],
    },
    'identifier': 'M83_Client_Library_Tests',
  },
  'WEBLAYER_CLIENT_SKEW_TESTS_NTH_MINUS_ONE_MILESTONE': {
    'args': [
      '--test-runner-outdir',
      '.',
    ],
    'swarming': {
      'cipd_packages': [
        {
          'cipd_package': 'chromium/testing/weblayer-x86',
          'revision': 'version:82.0.6666.45',
        },
      ],
    },
    'identifier': 'M82_Client_Library_Tests',
  },
}
"""
  yield (api.test('basic') +
         api.step_data('Read variants.pyl',
             api.file.read_text(TEST_VARIANTS_PYL)) +
         api.url.json('Getting Android beta channel releases',
                      [{'hashes':{'chromium':'abcd'}},
                       {'hashes':{'chromium':'efgh'}},
                       {'hashes':{'chromium':'ijkl'}}]) +
         api.url.json('Fetch information on commit abcd',
                      {'deployment': {'beta': '84.0.4147.89'}}) +
         api.url.json('Fetch information on commit efgh',
                      {'deployment': {'beta': '84.0.4147.56'}}) +
         api.url.json('Fetch information on commit ijkl',
                      {'deployment': {'beta': '83.0.4103.96'}}) +
         api.step_data('Reading //chrome/VERSION',
             api.file.read_text('a=83\nb=0\nc=4103\nd=96'))  +
         api.path.exists(api.path['cleanup'].join('apk_build_files')) +
         api.step_data('Reading //chrome/VERSION (2)',
             api.file.read_text('a=84\nb=0\nc=4147\nd=89')))

  yield (api.test('malformed-chromium-version') +
         api.expect_exception('ValueError') +
         api.step_data('Read variants.pyl',
             api.file.read_text(TEST_VARIANTS_PYL)) +
         api.url.json('Getting Android beta channel releases',
                      [{'hashes':{'chromium':'abcd'}}]) +
         api.url.json('Fetch information on commit abcd',
                      {'deployment': {'beta': '84.0.4147.89'}}) +
         api.step_data('Reading //chrome/VERSION',
             api.file.read_text('a=abd\nb=0\nc=4147\nd=89\n')))

  yield (api.test('build-no-cipd-pkgs') +
         api.step_data('Read variants.pyl',
             api.file.read_text(TEST_VARIANTS_PYL)) +
         api.url.json('Getting Android beta channel releases',
                      [{'hashes':{'chromium':'abcd'}}]) +
         api.url.json('Fetch information on commit abcd',
                      {'deployment': {'beta': '83.0.4103.56'}}))

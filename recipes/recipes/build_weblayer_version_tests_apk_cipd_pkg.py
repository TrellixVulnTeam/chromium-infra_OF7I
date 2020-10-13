# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import ast
import contextlib
import re

from recipe_engine import post_process

DEPS = [
  'build/chromium',
  'build/goma',
  'build/zip',
  'depot_tools/bot_update',
  'depot_tools/depot_tools',
  'depot_tools/gclient',
  'depot_tools/git',
  'depot_tools/git_cl',
  'recipe_engine/cipd',
  'recipe_engine/context',
  'recipe_engine/file',
  'recipe_engine/json',
  'recipe_engine/path',
  'recipe_engine/properties',
  'recipe_engine/python',
  'recipe_engine/raw_io',
  'recipe_engine/step',
  'recipe_engine/time',
  'recipe_engine/url'
]

CL_DESC = """weblayer: Add skew tests for new versions released in beta

Skew tests are being added for the following versions:
%s

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

VARIANTS_PYL_PATH = 'src/testing/buildbot/variants.pyl'


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
    api.gclient.sync(api.gclient.c)
    api.gclient.runhooks()
    yield
  finally:
    api.git('checkout', 'origin/master')
    api.git('clean', '-fd')
    # TODO(rmhasan): Use api.gn.clean to clean up old build files
    api.file.rmtree(
        'Removing src/out/Release',
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
  with api.step.nest('Converting hashes released in Beta to Chromium versions'):
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


def get_existing_cipd_tags(variants_pyl_content, variant_name_tmpls):
  variants = ast.literal_eval(variants_pyl_content)
  tags = set()
  for tmpl in variant_name_tmpls:
    for lib in [CLIENT, IMPL]:
      variant_name = tmpl % lib
      if variant_name in variants:
        cipd_tag = variants[
          variant_name]['swarming']['cipd_packages'][0]['revision']
        tags.add(cipd_tag)
  return tags


def maybe_update_variants_pyl(api, variants_pyl_content, variants_pyl_path):
  # Get recent milestone versions
  recent_milestone_versions = get_chromium_versions_to_add(api)
  variants_lines = variants_pyl_content.splitlines()

  variants_name_tmpls = [
      WEBLAYER_NTH_TMPL, WEBLAYER_NTH_MINUS_ONE_TMPL,
      WEBLAYER_NTH_MINUS_TWO_TMPL]
  # Map Chromium versions to variants.pyl identifiers
  versions_to_variants_id_tmpl = zip(
      recent_milestone_versions, variants_name_tmpls)

  existing_cipd_tags = get_existing_cipd_tags(variants_pyl_content,
                                              variants_name_tmpls)
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
          contains_recent_milestone_version = False
          open_bracket_count = 1
          lineno += 1
          current_config_lines = []

          while open_bracket_count:
            contains_recent_milestone_version |= (
                version in variants_lines[lineno])
            for c in variants_lines[lineno]:
              if c == '{':
                open_bracket_count += 1
              elif c == '}':
                open_bracket_count -= 1
            if open_bracket_count:
              current_config_lines.append(variants_lines[lineno])
              lineno += 1

          if not contains_recent_milestone_version:
            cipd_pkgs_to_create.add(version)
            new_variants_lines.extend(
                generate_skew_test_config_lines(library, version))
          else:
            new_variants_lines.extend(current_config_lines)

    new_variants_lines.append(variants_lines[lineno])
    lineno += 1

  if cipd_pkgs_to_create:
    # Build CIPD packages for new versions that were released in beta
    maybe_build_cipd_pkgs(api, cipd_pkgs_to_create, existing_cipd_tags)
    # Upload changes to variants.pyl to Gerrit
    upload_changes(api, new_variants_lines, variants_pyl_path,
                   cipd_pkgs_to_create)


def env_with_depot_tools(api):
  return {'PATH': api.path.pathsep.join(
      ('%(PATH)s', str(api.depot_tools.root)))}


def maybe_build_cipd_pkgs(api, cipd_pkgs_to_create, existing_cipd_tags):
  # Need to save mb_config.pyl for skew tests APK committed at ToT
  curr_path = api.path['checkout'].join(
     'weblayer', 'browser', 'android', 'javatests',
     'skew', 'mb_config.pyl')
  mb_config_path = api.path.mkdtemp().join('mb_config.pyl')
  api.file.copy('Copying mb_config.pyl from main branch',
                curr_path, mb_config_path)
  extract_dir = api.path['cleanup'].join('binaries')

  # Create CIPD packages for all chromium versions added to variants.pyl
  for version in cipd_pkgs_to_create:
    with api.step.nest('Maybe build and upload CIPD package for %s' % version):
      # Check if CIPD package exists before building it and uploading it.
      # If a previous build failed or timed out right after uploading CIPD
      # packages, then this recipe would upload duplicate CIPD packages if
      # there was no check.
      tag = 'version:%s' % version
      if tag in existing_cipd_tags or api.cipd.search(CIPD_PKG_NAME, tag):
        continue

      # Checkout chromium version
      with api.context(cwd=api.path['checkout'],
                       env=env_with_depot_tools(api)), \
        checkout_chromium_version_and_sync_3p_repos(api, version):
        zip_path = api.path.mkdtemp().join('binaries.zip')
        if api.path.exists(str(extract_dir)):
          api.file.rmtree('Cleaning up binaries from the last build',
                          extract_dir)
        cipd_out = api.path['checkout'].join(
            'out', 'Release',
            'skew_test_apk_%s.cipd' % version.replace('.', '_'))

        # Generate build files for weblayer instrumentation tests APK - x86
        mb_path = api.path['checkout'].join('tools', 'mb')
        mb_py_path = mb_path.join('mb.py')
        mb_args = [
           'zip', '--master=dummy.master', '--builder=dummy.builder',
           '--goma-dir', str(api.path['cache'].join('goma', 'client')),
           '--luci-auth',
           '--config-file=%s' % str(mb_config_path), 'out/Release',
           'weblayer_instrumentation_test_apk', str(zip_path)]
        try:
          api.python('Building weblayer_instrumentation_test_apk',
                     mb_py_path, mb_args)
        except api.step.StepFailure as e:
          api.goma.stop(e.retcode)
          raise e

        # Build weblayer instrumentation tests APK - x86 CIPD package
        api.zip.unzip('Uncompressing binaries', zip_path, extract_dir)
        api.cipd.build(extract_dir, cipd_out, CIPD_PKG_NAME)
        api.cipd.register(CIPD_PKG_NAME, cipd_out, tags={'version': version},
                          refs=['m%s' % version[:version.index('.')]])


def upload_changes(api, new_variants_lines, variants_pyl_path,
                   cipd_pkgs_to_create):
  # New chromium versions were added to variants.pyl so we need to write
  # new changes to src/testing/buildbot/variants.pyl and commit them to the
  # main branch
  with api.step.nest('Submit changes to %s' % VARIANTS_PYL_PATH):
    api.git('branch', '-D', 'new-skew-tests', ok_ret='any')
    api.git.new_branch('new-skew-tests')
    api.file.write_text(
        'Write changes to %s' % VARIANTS_PYL_PATH, variants_pyl_path,
        '\n'.join(new_variants_lines))
    api.python('Running generate_buildbot_json.py',
               api.path['checkout'].join(
                   'testing', 'buildbot', 'generate_buildbot_json.py'))
    api.git('commit', '-am', 'Adding skew tests for new versions')
    description = CL_DESC % '\n'.join(
        '%d, %s' % (idx + 1, ver)
        for idx, ver in enumerate(cipd_pkgs_to_create))

    upload_args = ['--force', '--r-owners']
    land_cl_arg = '--use-commit-queue'
    if api.properties.get('submit_cl'):
      upload_args.append(land_cl_arg)
    api.git_cl.upload(description, upload_args=upload_args)

    if land_cl_arg in upload_args:
      with api.step.nest('Landing CL'):
        wait_for_cl_to_land(api)


def wait_for_cl_to_land(api):
  total_checks = api.properties.get('total_cq_checks')
  # Time sleeping in between CL status checks
  interval = api.properties.get('interval_between_checks_in_secs')
  # Attempt to land the CL in total_cq_checks * interval_between_checks_in_secs
  # seconds
  for _ in range(total_checks):
    status = api.git_cl('status', ['--field', 'status'], 'git cl status',
                        stdout=api.git_cl.m.raw_io.output()).stdout.strip()
    # TODO(rmhasan) Add code to differentiate between CL's that were
    # merged and CL's that were abandoned.
    if status == 'closed':
      return
    api.time.sleep(interval)
  api.git_cl('set-close', [])
  api.python.failing_step('CQ timed out', 'CQ timed out; Abandoned CL')


def RunSteps(api):
  # Set gclient config
  api.gclient.set_config('chromium_no_telemetry_dependencies')
  api.gclient.apply_config('android')
  # Checkout chromium/src at ToT
  api.bot_update.ensure_checkout(with_tags=True)
  # Ensure GOMA is installed
  api.goma.ensure_goma()
  # start the GOMA proxy
  api.goma.start()

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
  variants_pyl_content = api.file.read_text(
      'Read %s' % VARIANTS_PYL_PATH, variants_pyl_path)

  maybe_update_variants_pyl(api, variants_pyl_content, variants_pyl_path)
  api.goma.stop(0)


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
  'WEBLAYER_IMPL_SKEW_TESTS_NTH_MINUS_ONE_MILESTONE': {
    'args': [
      '--test-runner-outdir',
      '.',
    ],
    'swarming': {
      'cipd_packages': [
        {
          'cipd_package': 'chromium/testing/weblayer-x86',
          'revision': 'version:83.0.4103.96',
        },
      ],
    },
    'identifier': 'M83_Implementation_Library_Tests',
  },
  'WEBLAYER_CLIENT_SKEW_TESTS_NTH_MINUS_TWO_MILESTONE': {
    'args': [
      '--test-runner-outdir',
      '.',
    ],
    'swarming': {
      'cipd_packages': [
        {
          'cipd_package': 'chromium/testing/weblayer-x86',
          'revision': 'version:81.0.1111.40',
        },
      ],
    },
    'identifier': 'M81_Client_Library_Tests',
  }
}
"""

  def cipd_step_data(version, instances=0):
    return api.step_data(
        ('Maybe build and upload CIPD package for %s.'
         'cipd search %s version:%s') % (
             version, CIPD_PKG_NAME, version),
         api.cipd.example_search(CIPD_PKG_NAME, instances=instances))

  def chrome_version_step_data(version):
    return api.step_data(('Maybe build and upload CIPD package for %s.'
                          'Reading //chrome/VERSION') % version,
                         api.file.read_text(
                             '\n'.join('='+str(v) for v in version.split('.'))))

  def land_cl_step_datas(statuses):
    def _create_step_data(status, iteration):
      name = ('Submit changes to %s.'
              'Landing CL.git cl status') % VARIANTS_PYL_PATH
      if iteration:
        name += ' (%d)' % (iteration + 1)
      return api.step_data(
          name,
          api.raw_io.stream_output(status, stream='stdout'))

    return reduce(lambda x, y: x + y,
                  [_create_step_data(s, itr)
                   for itr, s in enumerate(statuses)])

  def url_lib_json_step_datas(versions):
    hashes = [str(hash(s)) for s in versions]
    s1 = api.url.json('Getting Android beta channel releases',
                      [{'hashes':{'chromium': h}} for h in hashes])
    def _create_step_data(h, v):
      name = ('Converting hashes released in Beta to Chromium versions.'
              'Fetch information on commit %s') % h
      return api.url.json(name, {'deployment': {'beta': v}})

    return reduce(lambda x, y: x + y,
                  [s1] + [_create_step_data(h, v)
                          for h, v in zip(hashes, versions)])

  yield api.test(
      'basic',
      api.properties(submit_cl=True,
                     total_cq_checks=2,
                     interval_between_checks_in_secs=60) +
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
          api.file.read_text(TEST_VARIANTS_PYL)) +
      url_lib_json_step_datas(
          ['84.0.4147.89', '84.0.4147.56', '83.0.4103.96', '81.0.1111.40']) +
      api.path.exists(api.path['cleanup'].join('binaries')) +
      cipd_step_data('84.0.4147.89') +
      chrome_version_step_data('84.0.4147.89') +
      land_cl_step_datas(['commit', 'closed']))

  yield api.test(
      'only-build-new-milestone-released',
      api.properties(submit_cl=True,
                     total_cq_checks=2,
                     interval_between_checks_in_secs=60) +
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
          api.file.read_text(TEST_VARIANTS_PYL)) +
      url_lib_json_step_datas(['84.0.4147.89']) +
      cipd_step_data('84.0.4147.89') +
      chrome_version_step_data('84.0.4147.89') +
      api.path.exists(api.path['cleanup'].join('binaries')) +
      land_cl_step_datas(['commit', 'closed']))

  yield api.test(
      'version-already-exists-only-upload-cl',
      api.properties(submit_cl=True,
                     total_cq_checks=2,
                     interval_between_checks_in_secs=60) +
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
          api.file.read_text(TEST_VARIANTS_PYL)) +
      url_lib_json_step_datas(['84.0.4147.89']) +
      cipd_step_data('84.0.4147.89', 1) +
      land_cl_step_datas(['commit', 'closed']))

  yield api.test(
      'build-fails',
      api.properties(submit_cl=True,
                     total_cq_checks=2,
                     interval_between_checks_in_secs=60) +
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
          api.file.read_text(TEST_VARIANTS_PYL)) +
      url_lib_json_step_datas(['84.0.4147.89']) +
      cipd_step_data('84.0.4147.89') +
      chrome_version_step_data('84.0.4147.89') +
      api.step_data(('Maybe build and upload CIPD package for %s.'
                     'Building weblayer_instrumentation_test_apk') %
                    '84.0.4147.89',
                    retcode=1)  +
      api.path.exists(api.path['cleanup'].join('binaries')))

  yield api.test(
      'abandon-cl-after-time-out',
      api.properties(submit_cl=True,
                     total_cq_checks=2,
                     interval_between_checks_in_secs=60) +
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
          api.file.read_text(TEST_VARIANTS_PYL)) +
      url_lib_json_step_datas(['84.0.4147.89']) +
      cipd_step_data('84.0.4147.89') +
      chrome_version_step_data('84.0.4147.89') +
      api.path.exists(api.path['cleanup'].join('binaries')))

  yield api.test(
      'malformed-chromium-version',
      api.expect_exception('ValueError') +
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
          api.file.read_text(TEST_VARIANTS_PYL)) +
      url_lib_json_step_datas(['84.0.4147.89']) +
      cipd_step_data('84.0.4147.89') +
      api.step_data(('Maybe build and upload CIPD package for %s.'
                     'Reading //chrome/VERSION') % '84.0.4147.89',
                    api.file.read_text('=abd\n=0\n=4147\n=89')) +
      api.post_process(post_process.DropExpectation))

  yield api.test(
      'build-no-cipd-pkgs',
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
          api.file.read_text(TEST_VARIANTS_PYL)) +
      url_lib_json_step_datas(['83.0.4103.56']))

# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import ast
import contextlib
import re

from recipe_engine import post_process

DEPS = [
  'build/chromium',
  'build/chromiumdash',
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
  'recipe_engine/url'
]

CL_DESC = """[weblayer, skew tests] Refresh skew tests for M%s

This CL will add skew tests for version %s.

Bug:1041619
"""

# CIPD package path.
# https://chrome-infra-packages.appspot.com/p/chromium/testing/weblayer-x86/+/
CIPD_PKG_NAME='chromium/testing/weblayer-x86'

CHROMIUM_VERSION_REGEX = r'\d+\.\d+\.\d+\.\d+$'

# number of milestones to fetch
NUM_MILESTONES = 4

# Weblayer variants.pyl identifierr templates
WEBLAYER_NTH_TMPL = 'WEBLAYER_%s_SKEW_TESTS_NTH_MILESTONE'
WEBLAYER_NTH_MINUS_ONE_TMPL = (
    'WEBLAYER_%s_SKEW_TESTS_NTH_MINUS_ONE_MILESTONE')
WEBLAYER_NTH_MINUS_TWO_TMPL = (
    'WEBLAYER_%s_SKEW_TESTS_NTH_MINUS_TWO_MILESTONE')
WEBLAYER_NTH_MINUS_THREE_TMPL = (
    'WEBLAYER_%s_SKEW_TESTS_NTH_MINUS_THREE_MILESTONE')

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
CLIENT_ID_TMPL = "'identifier': 'Client Tests For {chromium_version}',"
IMPL_ID_TMPL = "'identifier': 'Implementation Tests For {chromium_version}',"

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
  lib_args_tmpl = CLIENT_ARGS_TMPL if library == CLIENT else IMPL_ARGS_TMPL
  lib_id_tmpl = CLIENT_ID_TMPL if library == CLIENT else IMPL_ID_TMPL
  INDENT_SIZE = 4
  lines.extend(
      ' ' * INDENT_SIZE + v.rstrip()
      for v in  lib_args_tmpl.format(milestone=milestone).splitlines())
  lines.append(' ' * INDENT_SIZE + lib_id_tmpl.format(chromium_version=version))
  lines.extend(
      ' ' * INDENT_SIZE + v.rstrip()
      for v in SWARMING_ARGS_TMPL.format(
          milestone=milestone, chromium_version=version).splitlines())
  return lines


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
    version_sanity_check(api, version)
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


def version_sanity_check(api, version):
  version_file = api.path['checkout'].join('chrome', 'VERSION')
  lines = api.file.read_text(
      'Reading //chrome/VERSION', version_file).splitlines()
  src_version = '.'.join(line[line.index('=') + 1:] for line in lines)
  if not re.match(CHROMIUM_VERSION_REGEX, src_version):
    raise ValueError("Chromium version, '%s', is not in proper format" %
                     src_version)
  assert version == src_version, (
      'Version %s does not match src/ version %s' % (version, src_version))


def get_existing_cipd_tags(variants_pyl_ast, variant_name_tmpls):
  tags = set()
  for tmpl in variant_name_tmpls:
    for lib in [CLIENT, IMPL]:
      variant_name = tmpl % lib
      if variant_name in variants_pyl_ast:
        cipd_tag = variants_pyl_ast[
          variant_name]['swarming']['cipd_packages'][0]['revision']
        tags.add(cipd_tag)
  return tags


def maybe_update_variants_pyl(api, variants_pyl_content,
                              variants_pyl_path, version, milestones):
  variants_lines = variants_pyl_content.splitlines()
  variants_pyl_ast = ast.literal_eval(variants_pyl_content)
  variants_name_tmpls = [
      WEBLAYER_NTH_TMPL, WEBLAYER_NTH_MINUS_ONE_TMPL,
      WEBLAYER_NTH_MINUS_TWO_TMPL, WEBLAYER_NTH_MINUS_THREE_TMPL]

  branch = version.split('.')[2]
  variants_name_tmpl = variants_name_tmpls[
      sorted([int(m['chromium_branch']) for m in milestones],
             reverse=True).index(int(branch))]

  existing_cipd_tags = get_existing_cipd_tags(variants_pyl_ast,
                                              variants_name_tmpls)

  # TODO(crbug.com/1041619): Add presubmit check for variants.pyl
  # that checks if variants.pyl follows a format which allows the code
  # below to overwrite skew test configurations
  new_variants_lines = []
  lineno = 0
  refresh_skew_test = False
  while lineno < len(variants_lines):
    for library in [CLIENT, IMPL]:
      variants_id = variants_name_tmpl % library

      if variants_id in variants_lines[lineno]:
        variant_cipd_tag = variants_pyl_ast[
            variants_id]['swarming']['cipd_packages'][0]['revision']
        variant_version = variant_cipd_tag.replace('version:', '')

        new_variants_lines.append(variants_lines[lineno])
        open_bracket_count = 1
        lineno += 1
        current_variant_lines = []

        while open_bracket_count:
          for c in variants_lines[lineno]:
            if c == '{':
              open_bracket_count += 1
            elif c == '}':
              open_bracket_count -= 1
          if open_bracket_count:
            current_variant_lines.append(variants_lines[lineno])
            lineno += 1

        if is_higher_version(variant_version, version):
          new_variants_lines.extend(
              generate_skew_test_config_lines(library, version))
          refresh_skew_test = True
        else:
          new_variants_lines.extend(current_variant_lines)

    new_variants_lines.append(variants_lines[lineno])
    lineno += 1

  if refresh_skew_test:
    # Upload a CIPD package if it does not already exist.
    if not 'version:%s' % version in existing_cipd_tags:
      # Build CIPD packages for inputted
      maybe_build_cipd_pkgs(api, version)

    # upload changes to variants.pyl
    upload_changes(
        api, new_variants_lines, variants_pyl_path, version)


def env_with_depot_tools(api):
  return {'PATH': api.path.pathsep.join(
      ('%(PATH)s', str(api.depot_tools.root)))}


def maybe_build_cipd_pkgs(api, version):
  # Need to save mb_config.pyl for skew tests APK committed at ToT
  curr_path = api.path['checkout'].join(
     'weblayer', 'browser', 'android', 'javatests',
     'skew', 'mb_config.pyl')
  mb_config_path = api.path.mkdtemp().join('mb_config.pyl')
  api.file.copy('Copying mb_config.pyl from main branch',
                curr_path, mb_config_path)
  extract_dir = api.path['cleanup'].join('binaries')

  # Create CIPD packages for all chromium versions added to variants.pyl
  with api.step.nest('Maybe build and upload CIPD package for %s' % version):
    # Check if CIPD package exists before building it and uploading it.
    # If a previous build failed or timed out right after uploading CIPD
    # packages, then this recipe would upload duplicate CIPD packages if
    # there was no check.
    tag = 'version:%s' % version
    if api.cipd.search(CIPD_PKG_NAME, tag):
      api.python.succeeding_step(
          'Skip building CIPD package for %s' % version,
          'CIPD package for %s is already in the CIPD server' % version)
      return

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


def upload_changes(api, new_variants_lines, variants_pyl_path, version):
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
    description = CL_DESC % (version.split('.')[0], version)

    upload_args = ['--force', '--r-owners', '--cq-dry-run',
                   '--send-mail']
    api.git_cl.upload(description, upload_args=upload_args)


def should_create_skew_test(version, milestones):
  input_branch = version.split('.')[2]
  milestone_branches = [m['chromium_branch'] for m in milestones]
  return input_branch in milestone_branches


def RunSteps(api):
  # Get Chromium version to build skew tests for
  version = api.properties.get('chromium_version')
  assert version, (
      'Recipe needs to be initialized with chromium'
      ' version in input properties')
  assert re.match(CHROMIUM_VERSION_REGEX, version), ('%r does not match the'
                                                     ' chromium version regex.')
  # Get milestones
  milestones = api.chromiumdash.milestones(
      NUM_MILESTONES, step_name='Fetching last %d milestones' % NUM_MILESTONES)
  # Check if chromium version inputted is for one of the milestone branches
  # being skew tested.
  if not should_create_skew_test(version, milestones):
    api.python.succeeding_step(
        'Exiting recipe early without creating skew tests',
        ('Chromium version %r is not a tag in any of the last'
         ' %d milestone branches') % (version, NUM_MILESTONES))
    return

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

  maybe_update_variants_pyl(
      api, variants_pyl_content, variants_pyl_path, version, milestones)
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
          'revision': 'version:82.0.4000.45',
        },
      ],
    },
    'identifier': 'M82_Client_Library_Tests',
  },
  'WEBLAYER_IMPL_SKEW_TESTS_NTH_MILESTONE': {
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
  'WEBLAYER_IMPL_SKEW_TESTS_NTH_MINUS_ONE_MILESTONE': {
    'args': [
      '--test-runner-outdir',
      '.',
    ],
    'swarming': {
      'cipd_packages': [
        {
          'cipd_package': 'chromium/testing/weblayer-x86',
          'revision': 'version:82.0.4000.45',
        },
      ],
    },
    'identifier': 'M82_Client_Library_Tests',
  },
}
"""

  def cipd_step_data(version, instances=0):
    return api.step_data(
        ('Maybe build and upload CIPD package for %s.'
         'cipd search %s version:%s') % (
             version, CIPD_PKG_NAME, version),
         api.cipd.example_search(CIPD_PKG_NAME, instances=instances))

  def chrome_version_step_data(version, actual_version=''):
    actual_version = actual_version or version
    return api.step_data(('Maybe build and upload CIPD package for %s.'
                          'Reading //chrome/VERSION') % version,
                         api.file.read_text(
                             '\n'.join('='+str(v)
                                       for v in actual_version.split('.'))))

  def milestone_json_step_datas(branches):
    return api.url.json('Fetching last %d milestones' % NUM_MILESTONES,
                        [{'chromium_branch': branch} for branch in branches])

  yield api.test(
      'basic',
      api.properties(chromium_version='82.0.4000.90') +
      milestone_json_step_datas(['4000', '4103', '3900', '3300']) +
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
          api.file.read_text(TEST_VARIANTS_PYL)) +
      api.path.exists(api.path['cleanup'].join('binaries')) +
      cipd_step_data('82.0.4000.90') +
      chrome_version_step_data('82.0.4000.90'))

  yield api.test(
      'build_fails',
      api.properties(chromium_version='82.0.4000.90') +
      milestone_json_step_datas(['4000', '4103', '3900', '3300']) +
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
          api.file.read_text(TEST_VARIANTS_PYL)) +
      api.path.exists(api.path['cleanup'].join('binaries')) +
      cipd_step_data('82.0.4000.90') +
      chrome_version_step_data('82.0.4000.90')+
      api.step_data(('Maybe build and upload CIPD package for 82.0.4000.90.'
                     'Building weblayer_instrumentation_test_apk'), retcode=1))

  yield api.test(
      'version_is_less_than_curr_tested',
      api.properties(chromium_version='82.0.4000.10') +
      milestone_json_step_datas(['4000', '4103', '3900', '3300']) +
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
                    api.file.read_text(TEST_VARIANTS_PYL)))

  yield api.test(
      'poorly_written_chromium_version',
      api.properties(chromium_version='82.0.4000.90') +
      api.expect_exception('ValueError') +
      milestone_json_step_datas(['4000', '4103', '3900', '3300']) +
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
                    api.file.read_text(TEST_VARIANTS_PYL)) +
      cipd_step_data('82.0.4000.90') +
      chrome_version_step_data('82.0.4000.90', 'abc.def.ghi.jkl') +
      api.post_process(post_process.DropExpectation))

  yield api.test(
      'cipd_package_already_exists',
      api.properties(chromium_version='82.0.4000.90') +
      milestone_json_step_datas(['4000', '4103', '3900', '3300']) +
      api.step_data('Read %s' % VARIANTS_PYL_PATH,
                    api.file.read_text(TEST_VARIANTS_PYL)) +
      cipd_step_data('82.0.4000.90', 1))

  yield api.test(
      'branch_is_not_in_milestones',
      api.properties(chromium_version='82.0.4003.90') +
      milestone_json_step_datas(['4000', '4103', '3900', '3300']))

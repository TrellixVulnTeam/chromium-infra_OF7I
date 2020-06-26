# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Rolls recipes.cfg dependencies for public projects."""

DEPS = [
    'recipe_autoroller',
    'recipe_engine/json',
    'recipe_engine/properties',
    'recipe_engine/proto',
    'recipe_engine/runtime',
    'recipe_engine/time',
]

from recipe_engine import recipe_api
from recipe_engine.post_process import MustRun

from PB.recipe_engine.recipes_cfg import RepoSpec, DepRepoSpecs

PROPERTIES = {
    'projects':
        recipe_api.Property(),
    'db_gcs_bucket':
        recipe_api.Property(
            kind=str,
            help=('GCS bucket in which to store metadata for historical roll '
                  'attempts'),
            default='recipe-mega-roller-crappy-db'),
}


def RunSteps(api, projects, db_gcs_bucket):
  api.recipe_autoroller.roll_projects(projects, db_gcs_bucket)
  return


def GenTests(api):
  # For conciseness.
  repo_spec = api.recipe_autoroller.repo_spec

  def test(name):
    return (
        api.test(name) +
        api.runtime(is_luci=True, is_experimental=False) +
        api.properties(projects=[
          ('build', 'https://example.com/build.git'),
        ])
    )

  yield (test('basic') + api.properties(db_gcs_bucket='somebucket') +
         api.recipe_autoroller.roll_data('build'))

  yield (test('multiple_commits') + api.properties(db_gcs_bucket='somebucket') +
         api.recipe_autoroller.roll_data('build', num_commits=3))

  yield (
      test('nontrivial') +
      api.recipe_autoroller.roll_data('build', trivial=False)
  )

  yield (
      test('empty') +
      api.recipe_autoroller.roll_data('build', empty=True)
  )

  yield (
      test('failure') +
      api.recipe_autoroller.roll_data('build', success=False)
  )

  yield (
      test('failed_upload') +
      api.recipe_autoroller.roll_data('build') +
      api.override_step_data(
          'build.git cl issue',
          api.json.output({'issue': None, 'issue_url': None}))
  )

  yield (
      test('repo_data_trivial_cq') +
      api.recipe_autoroller.recipe_cfg('build') +
      api.recipe_autoroller.repo_data(
          'build', trivial=True, status='commit',
          timestamp='2016-02-01T01:23:45') +
      api.time.seed(1451606400)
  )

  yield (
      test('repo_data_trivial_cq_stale') +
      api.recipe_autoroller.recipe_cfg('build') +
      api.recipe_autoroller.repo_data(
          'build', trivial=True, status='commit',
          timestamp='2016-02-01T01:23:45') +
      api.time.seed(1454371200)
  )

  yield (
      test('repo_data_trivial_open') +
      api.recipe_autoroller.repo_data(
          'build', trivial=True, status='open',
          timestamp='2016-02-01T01:23:45') +
      api.recipe_autoroller.roll_data('build') +
      api.time.seed(1451606400) +
      api.post_process(MustRun, 'build.git cl set-close')
  )

  yield (
      test('repo_data_trivial_closed') +
      api.recipe_autoroller.repo_data(
          'build', trivial=True, status='closed',
          timestamp='2016-02-01T01:23:45') +
      api.recipe_autoroller.roll_data('build') +
      api.time.seed(1451606400)
  )

  yield (
      test('repo_data_nontrivial_open') +
      api.recipe_autoroller.recipe_cfg('build') +
      api.recipe_autoroller.repo_data(
          'build', trivial=False, status='waiting',
          timestamp='2016-02-01T01:23:45') +
      api.time.seed(1451606400)
  )

  yield (
      test('repo_data_nontrivial_open_stale') +
      api.recipe_autoroller.recipe_cfg('build') +
      api.recipe_autoroller.repo_data(
          'build', trivial=False, status='waiting',
          timestamp='2016-02-01T01:23:45') +
      api.time.seed(1454371200)
  )

  yield (
      test('trivial_custom_tbr_no_dryrun') +
      api.recipe_autoroller.roll_data('build', repo_spec(trivial_commit=False))
  )

  yield (
      test('trivial_custom_tbr_dryrun') +
      api.recipe_autoroller.roll_data(
          'build', repo_spec(trivial_commit=False, trivial_dryrun=True))
  )

  yield (
      test('repo_disabled') +
      api.recipe_autoroller.roll_data(
          'build', repo_spec(disable_reason='I am a water buffalo.'))
  )

  # The recipe shouldn't crash if the autoroller options are not specified.
  yield (
      test('trivial_no_autoroll_options') +
      api.recipe_autoroller.roll_data(
          'build', repo_spec(include_autoroll_options=False), trivial=True)
  )

  yield (
      test('nontrivial_no_autoroll_options') +
      api.recipe_autoroller.roll_data(
          'build', repo_spec(include_autoroll_options=False), trivial=False)
  )

  no_cc_authors_spec = RepoSpec()
  no_cc_authors_spec.autoroll_recipe_options.no_cc_authors = True

  yield (
      test('no_cc_authors') + api.recipe_autoroller.roll_data('build') +
      api.override_step_data(
          'build.get deps',
          api.proto.output_stream(
              DepRepoSpecs(repo_specs={'recipe_engine': no_cc_authors_spec}))))

  # TODO(fxbug.dev/54380): delete this testcase after crrev.com/c/2252547 has
  # rolled into all downstream repos that are rolled by an autoroller.
  yield (test('failed_get_deps') + api.recipe_autoroller.roll_data('build') +
         api.override_step_data('build.get deps', retcode=1))

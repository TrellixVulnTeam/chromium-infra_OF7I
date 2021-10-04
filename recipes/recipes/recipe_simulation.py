# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""A continuous builder which runs recipe tests."""

import json

from recipe_engine.recipe_api import Property
from recipe_engine import post_process

DEPS = [
    'depot_tools/bot_update',
    'depot_tools/gclient',
    'recipe_engine/buildbucket',
    'recipe_engine/context',
    'recipe_engine/file',
    'recipe_engine/json',
    'recipe_engine/path',
    'recipe_engine/properties',
    'recipe_engine/python',
    'recipe_engine/raw_io',
    'recipe_engine/step',
]

PROPERTIES = {
  'git_repo': Property(
      kind=str,
      default=None,
      help='Git repo URL in which to simulate recipes. '
           'It must contain infra/config/recipes.cfg file. '
           'If not specified, uses buildbucket.gitiles_commit.'),
}


# Assumption of current recipe world.
CFG_PATH = 'infra/config/recipes.cfg'


def RunSteps(api, git_repo):
  if api.buildbucket.gitiles_commit.project:
    triggered_on = 'https://%s/%s' % (
        api.buildbucket.gitiles_commit.host,
        api.buildbucket.gitiles_commit.project)
    if git_repo is None:
      git_repo = triggered_on
    elif git_repo != triggered_on:
      raise api.step.InfraFailure(
          'Conflicting git repo URLs:\n'
          '  `git_repo` property %r\n'
          '  but triggered on Gitiles commit {%s}' %
          (git_repo, api.buildbucket.gitiles_commit))

  safe_project_name = ''.join(
      c if c.isalnum() else '_'
      for c in git_repo.replace('.googlesource.com', ''))
  root_dir = api.path['cache'].join('builder', safe_project_name)
  api.file.ensure_directory('ensure cache dir', root_dir)
  c = api.gclient.make_config()
  soln = c.solutions.add()
  soln.name = 's'
  soln.url = git_repo
  soln.revision = 'HEAD'

  with api.context(cwd=root_dir):
    api.bot_update.ensure_checkout(gclient_config=c)

  recipes_cfg_path = root_dir.join('s', *CFG_PATH.split('/'))
  cfg = json.loads(api.file.read_raw('read %s' % CFG_PATH, recipes_cfg_path))
  cfg_recipes_path = cfg.get('recipes_path', '')  # default to the repo's root.
  recipes_py_path = root_dir.join(
      's',
      *(cfg_recipes_path.split('/') + ['recipes.py']))
  try:
    api.step(
        'recipe simulation test', [
            recipes_py_path,
            'test',
            'run',
        ],
        stdout=api.raw_io.output_text())
  except api.step.StepFailure as e:
    # Run tests again to eliminate the coverage flakiness.
    # TODO(crbug.com/1147793): Remove it after Py3 migration is fully done.
    api.step.active_result.presentation.logs['stdout'] = e.result.stdout
    if 'Insufficient total coverage' not in e.result.stdout:
      raise e
    api.step('recipe simulation test (eliminate flakiness)', [
        recipes_py_path,
        'test',
        'run',
    ])


def GenTests(api):
  yield (
      api.test('gitiles_commit') +
      api.buildbucket.ci_build(
          # NOTE: this git_repo doesn't become a property, it's to simulate
          # api.buildbucket.gitiles_commit.
          git_repo='https://chromium.googlesource.com/chromium/tools/build',
      ) +
      api.step_data('read %s' % CFG_PATH, api.file.read_raw('''
        {
          "api_version": 2,
          "project_id": "build",
          "recipes_path": "scripts/slave",
          "repo_name": "build"
        }
      '''))
  )
  yield (
      api.test('tip_of_tree') +
      api.properties(
          git_repo='https://chromium.googlesource.com/infra/infra',
      ) +
      api.step_data('read %s' % CFG_PATH, api.file.read_raw('''
        {
          "api_version": 2,
          "project_id": "infra",
          "recipes_path": "recipes",
          "repo_name": "infra"
        }
      '''))
  )
  yield (
      api.test('recipes_path defaults to repo root') +
      api.properties(
          git_repo='https://chromium.googlesource.com/infra/infra',
      ) +
      api.step_data('read %s' % CFG_PATH, api.file.read_raw('''
        {
          "api_version": 2,
          "project_id": "recipes-py",
          "repo_name": "recipes-py"
        }
      ''')) +
      api.post_check(lambda check, steps: check(
          '[CACHE]/builder/https___chromium_infra_infra/s/recipes.py' in
          steps['recipe simulation test'].cmd)) +
      api.post_process(post_process.DropExpectation)
  )
  yield (
      api.test('conflicting_repo_urls') +
      api.buildbucket.ci_build(
          git_repo='https://chromium.googlesource.com/chromium/tools/build',
      ) +
      api.properties(
          git_repo='https://chromium.googlesource.com/infra/infra.GIT',
      )
  )
  yield (
      api.test('py2_py3_coverage_flakiness') + api.properties(
          git_repo='https://chromium.googlesource.com/infra/infra',) +
      api.step_data(
          'read %s' % CFG_PATH,
          api.file.read_raw('''
        {
          "api_version": 2,
          "project_id": "infra",
          "recipes_path": "recipes",
          "repo_name": "infra"
        }
      ''')) + api.step_data(
              'recipe simulation test',
              stdout=api.raw_io.output_text(
                  'FATAL: Insufficient total coverage for py2+py3 (99.99%)'),
              retcode=1) +
      api.step_data('recipe simulation test (eliminate flakiness)', retcode=0))
  yield (api.test('simulation_test_error') + api.properties(
      git_repo='https://chromium.googlesource.com/infra/infra',) +
         api.step_data(
             'read %s' % CFG_PATH,
             api.file.read_raw('''
        {
          "api_version": 2,
          "project_id": "infra",
          "recipes_path": "recipes",
          "repo_name": "infra"
        }
      ''')) + api.step_data(
                 'recipe simulation test',
                 stdout=api.raw_io.output_text('FAILED'),
                 retcode=1) +
         api.post_process(post_process.StepFailure, 'recipe simulation test') +
         api.post_process(post_process.DropExpectation))

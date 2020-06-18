# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Definitions of infra.git CI resources."""

load('//lib/build.star', 'build')
load('//lib/infra.star', 'infra')
load('//lib/recipes.star', 'recipes')


infra.console_view(name = 'infra', title = 'infra/infra repository console')
infra.cq_group(name = 'infra', tree_status_host = 'infra-status.appspot.com')


def ci_builder(
      name,
      os,
      cpu=None,
      recipe=None,
      console_category=None,
      properties=None,
      extra_dimensions=None,
      schedule=None,
      infra_triggered=True,
      experimental=False,
      tree_closing=False,
  ):
  infra.builder(
      name = name,
      bucket = 'ci',
      executable = infra.recipe(recipe or 'infra_continuous'),
      os = os,
      cpu = cpu,
      triggered_by = [infra.poller()] if infra_triggered else None,
      schedule = schedule,
      properties = properties,
      gatekeeper_group = '' if experimental else 'chromium.infra',
      extra_dimensions=extra_dimensions,
      notifies = infra.tree_closing_notifiers() if tree_closing else None,
  )
  luci.console_view_entry(
      builder = name,
      console_view = 'infra',
      category = console_category or infra.category_from_os(os, short=True),
  )


def try_builder(
      name,
      os,
      cpu=None,
      recipe=None,
      experiment_percentage=None,
      properties=None
  ):
  infra.builder(
      name = name,
      bucket = 'try',
      executable = infra.recipe(recipe or 'infra_repo_trybot'),
      os = os,
      cpu = cpu,
      properties = properties,
  )
  luci.cq_tryjob_verifier(
      builder = name,
      cq_group = 'infra',
      experiment_percentage=experiment_percentage,
  )


# CI Linux.
ci_builder(name = 'infra-continuous-zesty-64', os = 'Ubuntu-17.04')
ci_builder(name = 'infra-continuous-yakkety-64', os = 'Ubuntu-16.10')
ci_builder(name = 'infra-continuous-xenial-64', os = 'Ubuntu-16.04', tree_closing = True)
ci_builder(name = 'infra-continuous-xenial-arm64', os = 'Ubuntu-16.04', cpu = 'arm64', experimental = True)
ci_builder(name = 'infra-continuous-trusty-64', os = 'Ubuntu-14.04', tree_closing = True)

# CI OSX.
ci_builder(name = 'infra-continuous-mac-10.11-64', os = 'Mac-10.11')
ci_builder(name = 'infra-continuous-mac-10.12-64', os = 'Mac-10.12')
ci_builder(name = 'infra-continuous-mac-10.13-64', os = 'Mac-10.13', tree_closing = True)
ci_builder(name = 'infra-continuous-mac-10.14-64', os = 'Mac-10.14')
ci_builder(name = 'infra-continuous-mac-10.15-64', os = 'Mac-10.15')

# CI Win.
ci_builder(name = 'infra-continuous-win7-64', os = 'Windows-7')
ci_builder(name = 'infra-continuous-win10-64', os = 'Windows-10')

# CI for building docker images.
ci_builder(
    name = 'infra-continuous-images',
    os = 'Ubuntu-16.04',  # note: exact Linux version doesn't really matter
    recipe = 'images_builder',
    console_category = 'misc',
    properties = {
        'mode': 'MODE_CI',
        'project': 'PROJECT_INFRA',
        'infra': 'ci',
        'manifests': ['infra/build/images/deterministic'],
    },
)

# Builds arm64-flavored docker images for swarm_docker.
ci_builder(
    name = 'swarm-docker-arm64-image-builder',
    os = 'Ubuntu-16.04',
    cpu = 'arm64',
    # Make sure we're not building Docker images inside another container.
    extra_dimensions = {'inside_docker': '0'},
    infra_triggered = False,  # No need to build at every commit.
    schedule = 'triggered',
    recipe = 'docker_image_builder',
    console_category = 'misc',
    properties = {
        'container_name': 'swarm_docker_arm64',
        'dir_name': 'swarm_docker',
    }
)


# All trybots.
try_builder(name = 'infra-try-xenial-64', os = 'Ubuntu-16.04')
try_builder(name = 'infra-try-xenial-arm64', os = 'Ubuntu-16.04', cpu = 'arm64',
            experiment_percentage=100)
try_builder(name = 'infra-try-trusty-64', os = 'Ubuntu-14.04')
try_builder(name = 'infra-try-mac', os = 'Mac-10.13')
try_builder(name = 'infra-try-win', os = 'Windows')
try_builder(name = 'infra-try-frontend', os = 'Ubuntu-16.04', recipe = 'infra_frontend_tester')

# Experimental trybot for building docker images out of infra.git CLs.
try_builder(
    name = 'infra-try-images',
    os = 'Ubuntu-16.04',
    recipe = 'images_builder',
    experiment_percentage = 100,
    properties = {
        'mode': 'MODE_CL',
        'project': 'PROJECT_INFRA',
        'infra': 'try',
        'manifests': ['infra/build/images/deterministic'],
    },
)

# Presubmit trybot.
build.presubmit(name = 'infra-try-presubmit', cq_group = 'infra', repo_name = 'infra')

# Recipes ecosystem.
recipes.simulation_tester(
    name = 'infra-continuous-recipes-tests',
    project_under_test = 'infra',
    triggered_by = infra.poller(),
    console_view = 'infra',
    console_category = 'misc',
    gatekeeper_group = 'chromium.infra',
)

luci.cq_tryjob_verifier(
    builder = 'infra-internal:try/build_limited Roll Tester (infra)',
    cq_group = 'infra',
)

luci.cq_tryjob_verifier(
    builder = 'infra-internal:try/chrome_release Roll Tester (infra)',
    cq_group = 'infra',
)

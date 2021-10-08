# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Rolls recipes.cfg dependencies for public projects."""

DEPS = [
    'recipe_autoroller',
    'recipe_engine/buildbucket',
    'recipe_engine/json',
    'recipe_engine/properties',
    'recipe_engine/proto',
    'recipe_engine/time',
]

from recipe_engine import recipe_api

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


def GenTests(api):
  yield api.test('basic',
        api.properties(projects=[
            ('build', 'https://example.com/build.git'),
        ]))

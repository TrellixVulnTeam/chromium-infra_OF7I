# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Fixes builds in the datastore.

This code changes each time something needs to be migrated once.
"""

import datetime

from google.appengine.ext import ndb

from components import utils

from go.chromium.org.luci.buildbucket.proto import build_pb2

import bulkproc
import logging
import model

PROC_NAME = 'fix_builds'

bulkproc.register(
    PROC_NAME,
    lambda keys, _payload: _fix_builds(keys),
    keys_only=True,
)


def launch():  # pragma: no cover
  bulkproc.start(PROC_NAME)


def _fix_builds(build_keys):  # pragma: no cover
  res_iter = utils.async_apply(build_keys, _fix_build_async, unordered=True)
  # async_apply returns an iterator. We need to traverse it, otherwise nothing
  # will happen.
  for _ in res_iter:
    pass


@ndb.transactional_tasklet
def _fix_build_async(build_key):  # pragma: no cover
  steps_key = model.BuildSteps.key_for(build_key)
  build, steps = yield ndb.get_multi_async([build_key, steps_key])
  if not build or not build.is_ended:
    return

  to_put = []

  if steps and not steps.step_container_bytes_zipped:
    # Prior to the introduction of step_container_bytes_zipped, steps were
    # compressed using ndb's compression instead of explicit zipping. Go doesn't
    # understand this compression, so fix these entities (all entities created
    # before an arbitrary date by which the change was hopefully released).
    # See 96753cfd8462af82ea365a4c8918539786f4c3bd for more details.
    if build.create_time < datetime.datetime(2019, 4, 1):
      p = build_pb2.Build()
      steps.read_steps(p)
      steps.write_steps(p)
      to_put.append(steps)

  if to_put:
    logging.info('fixing %s' % build.key.id())
    yield ndb.put_multi_async(to_put)

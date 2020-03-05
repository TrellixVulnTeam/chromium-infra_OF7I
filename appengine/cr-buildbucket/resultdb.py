# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""This module integrates buildbucket with resultdb."""

import logging

from google.appengine.ext import ndb

from components import net
from components.prpc import client
from go.chromium.org.luci.resultdb.proto.rpc.v1 import recorder_pb2
from go.chromium.org.luci.resultdb.proto.rpc.v1 import recorder_prpc_pb2
from go.chromium.org.luci.resultdb.proto.rpc.v1 import invocation_pb2

import model


def sync(build):
  """Syncs the build with resultdb.

  Currently, only creates an invocation for the build if none exists yet.

  Returns a boolean indicating whether datastore changes were made.
  """
  rdb = build.proto.infra.resultdb
  if not rdb.hostname or rdb.invocation:
    return False

  new_invocation, update_token = _create_invocation(build)
  assert new_invocation
  assert new_invocation.name, 'Empty invocation name in %s' % (new_invocation)
  assert update_token

  @ndb.transactional
  def txn():
    bundle = model.BuildBundle.get(build, infra=True)
    assert bundle and bundle.infra, bundle
    with bundle.infra.mutate() as infra:
      rdb = infra.resultdb
      if rdb.invocation:
        logging.warning('build already has an invocation %r', rdb.invocation)
        assert bundle.build.resultdb_update_token
        return False
      rdb.invocation = new_invocation.name
    assert not bundle.build.resultdb_update_token
    bundle.build.resultdb_update_token = update_token
    bundle.put()
    return True

  return txn()


def _create_invocation(build):
  """Creates an invocation in resultdb for |build|."""
  # TODO(crbug.com/1056006): Populate bigquery_exports
  # TODO(crbug.com/1056007): Create an invocation like
  #     "build:<project>/<bucket>/<builder>/<number>" if number is available,
  #     and make it include the "build:<id>" inv.
  invocation_id = 'build:%d' % build.proto.id
  response_metadata = {}
  recorder = client.Client(
      build.proto.infra.resultdb.hostname,
      recorder_prpc_pb2.RecorderServiceDescription,
  )
  ret = recorder.CreateInvocation(
      recorder_pb2.CreateInvocationRequest(
          invocation_id=invocation_id,
          invocation=invocation_pb2.Invocation(),
          request_id=invocation_id,
      ),
      credentials=client.service_account_credentials(),
      response_metadata=response_metadata,
  )
  return ret, response_metadata['update-token']

# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""This module integrates buildbucket with resultdb."""

import json
import logging

from google.appengine.ext import ndb
import webapp2

from components import decorators
from components import net
from components.prpc import client
from components.prpc import codes
from go.chromium.org.luci.buildbucket.proto import common_pb2
from go.chromium.org.luci.resultdb.proto.rpc.v1 import recorder_pb2
from go.chromium.org.luci.resultdb.proto.rpc.v1 import recorder_prpc_pb2
from go.chromium.org.luci.resultdb.proto.rpc.v1 import invocation_pb2

import config
import model
import tq


@ndb.tasklet
def create_invocations_async(builds_and_configs):
  """Creates resultdb invocations for each build.

  Only create invocations if ResultDB hostname is globally set.

  Args:
    builds_and_configs: a list of (build, builder_cfg) tuples.
  """
  if not builds_and_configs:  # pragma: no cover
    return
  settings = yield config.get_settings_async()
  resultdb_host = settings.resultdb.hostname
  if not resultdb_host:
    # resultdb host needs to be enabled at service level, i.e. globally per
    # buildbucket deployment.
    return

  resp = yield _create_invocations_async(builds_and_configs, resultdb_host)
  assert (
      len(resp.invocations) == len(resp.update_tokens) ==
      len(builds_and_configs)
  )
  for inv, tok, (build, _) in zip(resp.invocations, resp.update_tokens,
                                  builds_and_configs):
    build.proto.infra.resultdb.invocation = inv.name
    build.resultdb_update_token = tok


@ndb.tasklet
def _create_invocations_async(builds_and_configs, hostname):
  """Creates a batch of invocations in resultdb for the given NewBuilds."""
  # TODO(crbug.com/1056007): Create an invocation like
  #     "build:<project>/<bucket>/<builder>/<number>" if number is available,
  #     and make it include the "build:<id>" inv.

  # build:<first build id>+<number of other builds in the batch>
  request_id = 'build:%d+%d' % (
      builds_and_configs[0][0].proto.id, len(builds_and_configs) - 1
  )
  req = recorder_pb2.BatchCreateInvocationsRequest(request_id=request_id)
  for build, cfg in builds_and_configs:
    req.requests.add(
        invocation_id='build:%d' % build.proto.id,
        invocation=invocation_pb2.Invocation(
            bigquery_exports=cfg.resultdb.bq_exports,
        ),
    )
  response_metadata = {}
  recorder = client.Client(
      hostname,
      recorder_prpc_pb2.RecorderServiceDescription,
  )
  ret = yield recorder.BatchCreateInvocationsAsync(
      req,
      credentials=client.service_account_credentials(),
      response_metadata=response_metadata,
  )
  assert ret.update_tokens
  raise ndb.Return(ret)


def enqueue_invocation_finalization_async(build):
  """Enqueues a task to call ResultDB to finalize the build's invocation."""
  assert ndb.in_transaction()
  assert build
  assert build.is_ended

  task_def = {
      'url': '/internal/task/resultdb/finalize/%d' % build.key.id(),
      'payload': {'id': build.key.id()},
      'retry_options': {'task_age_limit': model.BUILD_TIMEOUT.total_seconds()},
  }

  return tq.enqueue_async('backend-default', [task_def])


class FinalizeInvocation(webapp2.RequestHandler):  # pragma: no cover
  """Calls ResultDB to finalize the build's invocation."""

  @decorators.require_taskqueue('backend-default')
  def post(self, build_id):  # pylint: disable=unused-argument
    build_id = json.loads(self.request.body)['id']
    _finalize_invocation(build_id)


def _is_interrupted(build):
  # Treat canceled builds and infra failures as interrupted builds.
  return build.status in (common_pb2.CANCELED, common_pb2.INFRA_FAILURE)


def _finalize_invocation(build_id):
  bundle = model.BuildBundle.get(build_id, infra=True)
  rdb = bundle.infra.parse().resultdb
  if not rdb.hostname or not rdb.invocation:
    # If there's no hostname or no invocation, it means resultdb integration
    # is not enabled for this build.
    return

  try:
    _ = _call_finalize_rpc(
        rdb.hostname,
        recorder_pb2.FinalizeInvocationRequest(
            name=rdb.invocation, interrupted=_is_interrupted(bundle.build)
        ),
        {'update-token': bundle.build.resultdb_update_token},
    )
  except client.RpcError as rpce:
    if rpce.status_code in (codes.StatusCode.FAILED_PRECONDITION,
                            codes.StatusCode.PERMISSION_DENIED):
      logging.error('RpcError when finalizing %s: %s', rdb.invocation, rpce)
    else:
      raise  # Retry other errors.


def _call_finalize_rpc(host, req, metadata):  # pragma: no cover
  recorder = client.Client(
      host,
      recorder_prpc_pb2.RecorderServiceDescription,
  )
  return recorder.FinalizeInvocation(
      req, credentials=client.service_account_credentials(), metadata=metadata
  )

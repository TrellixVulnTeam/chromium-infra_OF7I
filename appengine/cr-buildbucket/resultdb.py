# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""This module integrates buildbucket with resultdb."""

import json
import logging

from google.appengine.api import app_identity
from google.appengine.ext import ndb
import webapp2

from components import decorators
from components import net
from components.prpc import client
from components.prpc import codes
from go.chromium.org.luci.buildbucket.proto import common_pb2
from go.chromium.org.luci.resultdb.proto.v1 import recorder_pb2
from go.chromium.org.luci.resultdb.proto.v1 import recorder_prpc_pb2
from go.chromium.org.luci.resultdb.proto.v1 import invocation_pb2

import config
import model
import tq


def _split_by_project(builds_and_configs):
  """Splits the list of pairs into multiple where each belongs to one project.
  """
  batches = {}
  for build, cfg in builds_and_configs:
    project = build.proto.builder.project
    batches.setdefault(project, [])
    batches[project].append((build, cfg))
  return batches.values()


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

  bb_host = app_identity.get_default_version_hostname()
  # We need to do one batch request per project, since the rpc to create
  # invocations uses per-project credentials.
  batches = _split_by_project(builds_and_configs)
  batch_reqs_and_creds = []
  for batch in batches:
    project = batch[0][0].proto.builder.project
    req = recorder_pb2.BatchCreateInvocationsRequest(
        # build-<first build id>+<number of other builds in the batch>
        request_id='build-%d+%d' % (batch[0][0].proto.id, len(batch) - 1)
    )

    for build, cfg in batch:
      history_options = invocation_pb2.HistoryOptions()
      history_options.use_invocation_timestamp = (
          cfg.resultdb.history_options.use_invocation_timestamp
      )
      req.requests.add(
          invocation_id='build-%d' % build.proto.id,
          invocation=invocation_pb2.Invocation(
              realm=build.realm,
              bigquery_exports=cfg.resultdb.bq_exports,
              producer_resource='//%s/builds/%s' % (bb_host, build.key.id()),
              history_options=history_options,
          ),
      )

    # Accumulate one (request, credentials) pair per batch.
    batch_reqs_and_creds.append((
        req,
        client.project_credentials(project),
    ))

  rec_client = _recorder_client(resultdb_host)
  # Do rpcs in parallel.
  resps = yield [
      rec_client.BatchCreateInvocationsAsync(req, credentials=creds)
      for req, creds in batch_reqs_and_creds
  ]

  for batch, res in zip(batches, resps):
    assert len(res.invocations) == len(res.update_tokens) == len(batch)
    # Populate the builds' name and token from the rpc response.
    for inv, tok, (build, _) in zip(res.invocations, res.update_tokens, batch):
      build.proto.infra.resultdb.invocation = inv.name
      build.resultdb_update_token = tok


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


def _finalize_invocation(build_id):
  bundle = model.BuildBundle.get(build_id, infra=True)
  rdb = bundle.infra.parse().resultdb
  if not rdb.hostname or not rdb.invocation:
    # If there's no hostname or no invocation, it means resultdb integration
    # is not enabled for this build.
    return

  try:
    _recorder_client(rdb.hostname).FinalizeInvocation(
        recorder_pb2.FinalizeInvocationRequest(name=rdb.invocation),
        credentials=client.service_account_credentials(),
        metadata={'update-token': bundle.build.resultdb_update_token},
    )
  except client.RpcError as rpce:
    if rpce.status_code in (codes.StatusCode.FAILED_PRECONDITION,
                            codes.StatusCode.PERMISSION_DENIED):
      logging.error('RpcError when finalizing %s: %s', rdb.invocation, rpce)
    else:
      raise  # Retry other errors.


def _recorder_client(hostname):  # pragma: no cover
  return client.Client(hostname, recorder_prpc_pb2.RecorderServiceDescription)

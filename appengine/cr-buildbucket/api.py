# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import functools
import logging

from google.appengine.ext import ndb
from google.protobuf import field_mask_pb2
from google.protobuf import symbol_database

from components import auth
from components import protoutil
from components import prpc
from components import utils

# Some of these imports are required to populate proto symbol db.
from go.chromium.org.luci.buildbucket.proto import build_pb2  # pylint: disable=unused-import
from go.chromium.org.luci.buildbucket.proto import builds_service_pb2 as rpc_pb2  # pylint: disable=unused-import
from go.chromium.org.luci.buildbucket.proto import builds_service_prpc_pb2
from go.chromium.org.luci.buildbucket.proto import common_pb2
from go.chromium.org.luci.buildbucket.proto import step_pb2  # pylint: disable=unused-import

import bbutil
import buildtags
import config
import creation
import default_field_masks
import errors
import events
import experiments
import model
import search
import service
import tokens
import user
import validation

# Header for passing token to authenticate build messages, e.g. UpdateBuild RPC.
# Lowercase because metadata is stored in lowercase.
BUILD_TOKEN_HEADER = 'x-build-token'


class StatusError(errors.Error):

  def __init__(self, code, *details_and_args):
    if details_and_args:
      details = details_and_args[0] % details_and_args[1:]
    else:
      details = code[1]

    self.code = code
    super(StatusError, self).__init__(details)


unimplemented = lambda *args: StatusError(prpc.StatusCode.UNIMPLEMENTED, *args)
not_found = lambda *args: StatusError(prpc.StatusCode.NOT_FOUND, *args)
invalid_argument = (
    lambda *args: StatusError(prpc.StatusCode.INVALID_ARGUMENT, *args)
)


def current_identity_cannot(action_format, *args):
  """Raises a StatusError with a PERMISSION_DENIED code."""
  action = action_format % args
  msg = '%s cannot %s' % (auth.get_current_identity().to_bytes(), action)
  return StatusError(prpc.StatusCode.PERMISSION_DENIED, '%s', msg)


METHODS_BY_NAME = {
    m.name: m for m in (
        builds_service_prpc_pb2.BuildsServiceDescription['service_descriptor']
        .method
    )
}


def rpc_impl_async(rpc_name):
  """Returns a decorator for an async Builds RPC implementation.

  Handles auth.AuthorizationError and StatusError.

  Adds fourth method argument to the method, a protoutil.Mask.
  If request has "fields" field, treats it as a FieldMask, parses it to a
  protoutil.Mask and passes that.
  After the method returns a response, the response is trimmed according to the
  mask. Requires request message to have "fields" field of type FieldMask.
  The default field masks are defined in default_field_masks.MASKS.
  """

  method_desc = METHODS_BY_NAME[rpc_name]
  res_class = symbol_database.Default().GetSymbol(method_desc.output_type[1:])
  default_mask = default_field_masks.MASKS.get(res_class)

  def decorator(fn_async):

    @functools.wraps(fn_async)
    @ndb.tasklet
    def decorated(req, res, ctx):
      try:
        mask = default_mask
        # Require that all RPC requests have "fields" field mask.
        if req.HasField('fields'):
          try:
            mask = protoutil.Mask.from_field_mask(
                req.fields, res_class.DESCRIPTOR
            )
          except ValueError as ex:
            raise invalid_argument('invalid fields: %s', ex)

        try:
          yield fn_async(req, res, ctx, mask)
          if mask:  # pragma: no branch
            mask.trim(res)
          raise ndb.Return(res)
        except auth.AuthorizationError:
          raise not_found()
        except validation.Error as ex:  # pragma: no cover
          raise invalid_argument('%s', ex.message)

      except errors.Error as ex:
        ctx.set_code(ex.code)
        ctx.set_details(ex.message)
        raise ndb.Return(None)

    return decorated

  return decorator


def bucket_id_string(builder_id):
  return config.format_bucket_id(builder_id.project, builder_id.bucket)


def builds_to_protos_async(builds, build_mask=None):
  """Converts model.Build instances to build_pb2.Build messages.

  Like model.builds_to_protos_async, but accepts a build mask and mutates
  model.Build entities in addition to build_pb2.Build.
  """
  # Trim model.Build.proto before deep-copying into destination.
  if build_mask:  # pragma: no branch
    for b, _ in builds:
      build_mask.trim(b.proto)

  includes = lambda path: build_mask and build_mask.includes(path)

  return model.builds_to_protos_async(
      builds,
      load_tags=includes('tags'),
      load_output_properties=includes('output.properties'),
      load_input_properties=includes('input.properties'),
      load_steps=includes('steps'),
      load_infra=includes('infra'),
  )


def build_to_proto_async(build, dest, build_mask=None):
  """Converts a model.Build instance to a build_pb2.Build message.

  Like builds_to_protos_async, but singular.
  """
  return builds_to_protos_async([(build, dest)], build_mask)


def build_predicate_to_search_query(predicate):
  """Converts a rpc_pb2.BuildPredicate to search.Query.

  Assumes predicate is valid.
  """
  q = search.Query(
      tags=[buildtags.unparse(p.key, p.value) for p in predicate.tags],
      created_by=predicate.created_by or None,
      include_experimental=predicate.include_experimental,
      status=predicate.status,
  )

  # Filter by builder.
  if predicate.HasField('builder'):
    if predicate.builder.bucket:
      q.bucket_ids = [bucket_id_string(predicate.builder)]
      q.builder = predicate.builder.builder
    else:
      q.project = predicate.builder.project

  # Filter by gerrit changes.
  buildsets = [
      buildtags.gerrit_change_buildset(c) for c in predicate.gerrit_changes
  ]
  q.tags.extend(buildtags.unparse(buildtags.BUILDSET_KEY, b) for b in buildsets)

  # Filter by creation time.
  if predicate.create_time.HasField('start_time'):
    q.create_time_low = predicate.create_time.start_time.ToDatetime()
  if predicate.create_time.HasField('end_time'):
    q.create_time_high = predicate.create_time.end_time.ToDatetime()

  # Filter by build range.
  if predicate.HasField('build'):
    # 0 means no boundary.
    # Convert BuildRange to search.Query.{build_low, build_high}.
    # Note that, unlike build_low/build_high, BuildRange encapsulates the fact
    # that build ids are decreasing. We need to reverse the order.
    if predicate.build.start_build_id:  # pragma: no branch
      # Add 1 because start_build_id is inclusive and build_high is exclusive.
      q.build_high = predicate.build.start_build_id + 1
    if predicate.build.end_build_id:  # pragma: no branch
      # Subtract 1 because end_build_id is exclusive and build_low is inclusive.
      q.build_low = predicate.build.end_build_id - 1

  # Filter by canary.
  if predicate.canary != common_pb2.UNSET:
    q.canary = predicate.canary == common_pb2.YES

  return q


@ndb.tasklet
def prepare_schedule_build_request_async(req):
  """Populates empty fields in the req with properties from the template build.

  When req doesn't have template_build_id property, return req as is.
  """
  if not req.template_build_id:
    raise ndb.Return(req)

  build = yield service.get_async(req.template_build_id)
  if not build:
    raise not_found('build %d is not found', req.template_build_id)

  bp = build_pb2.Build()
  yield model.builds_to_protos_async(
      [(build, bp)],
      load_tags=True,
      load_input_properties=True,
      load_output_properties=False,
      load_steps=False,
      load_infra=False,
  )

  # Remove empty msgs so empty objects ("{}") won't appear during serialization.
  non_empty_or_none = lambda m: m if len(m.ListFields()) != 0 else None

  # First initialize the new request based on the build.
  new_req = rpc_pb2.ScheduleBuildRequest(
      builder=bp.builder,
      experiments={exp: True for exp in bp.input.experiments},
      properties=non_empty_or_none(bp.input.properties),
      gitiles_commit=non_empty_or_none(bp.input.gitiles_commit),
      gerrit_changes=bp.input.gerrit_changes,
      tags=bp.tags,
      dimensions=bp.infra.buildbucket.requested_dimensions,
      priority=bp.infra.swarming.priority,
      # Don't copy notify and fields because they are not build configuration.
      critical=bp.critical,
      exe=non_empty_or_none(bp.exe),
      # Don't copy swarming or we are likely to create a dead-born build
      # due to completed parent.
  )

  new_req.experiments[experiments.CANARY] = bp.canary
  new_req.experiments[experiments.NON_PROD] = bp.input.experimental

  # Then apply the overrides specified in req.
  # Clear composite fields if they are specified in req.
  for f, _ in req.ListFields():
    if f.name == 'experiments':  # MergeFrom applies this correctly
      continue
    new_req.ClearField(f.name)
  new_req.MergeFrom(req)

  raise ndb.Return(new_req)


@rpc_impl_async('GetBuild')
@ndb.tasklet
def get_build_async(_req, _res, _ctx, _mask):
  """Retrieves a build by id or number."""
  raise unimplemented()


@rpc_impl_async('SearchBuilds')
@ndb.tasklet
def search_builds_async(_req, _res, _ctx, _mask):  # pragma: no cover
  """Searches for builds."""
  raise unimplemented()


@rpc_impl_async('UpdateBuild')
@ndb.tasklet
def update_build_async(_req, _res, _ctx, _mask):  # pragma: no cover
  """Update build as in given request."""
  raise unimplemented()


@rpc_impl_async('ScheduleBuild')
@ndb.tasklet
def schedule_build_async(req, res, _ctx, mask):
  """Schedules one build."""
  validation.validate_schedule_build_request(req)
  req = yield prepare_schedule_build_request_async(req)

  bucket_id = config.format_bucket_id(req.builder.project, req.builder.bucket)
  if not (yield user.has_perm_async(user.PERM_BUILDS_ADD, bucket_id)):
    raise current_identity_cannot('schedule builds to bucket %s', bucket_id)

  build_req = creation.BuildRequest(schedule_build_request=req)
  build = yield creation.add_async(build_req)
  yield build_to_proto_async(build, res, mask)


# A tuple of a request and response.
_ReqRes = collections.namedtuple('_ReqRes', 'request response')
# A tuple of a request, response and field mask.
# Used internally by schedule_build_multi.
_ScheduleItem = collections.namedtuple('_ScheduleItem', 'request response mask')


def schedule_build_multi(batch):
  """Schedules multiple builds.

  Args:
    batch: list of _ReqRes where
      request is rpc_pb2.ScheduleBuildRequest and
      response is rpc_pb2.BatchResponse.Response.
      Response objects will be mutated.
  """
  # Validate requests.
  valid_entries = []
  for rr in batch:
    try:
      validation.validate_schedule_build_request(rr.request)
    except validation.Error as ex:
      rr.response.error.code = prpc.StatusCode.INVALID_ARGUMENT.value
      rr.response.error.message = ex.message
      continue

    # Parse the field mask.
    # Normally it is done by rpc_impl_async.
    mask = None
    if rr.request.HasField('fields'):
      try:
        mask = protoutil.Mask.from_field_mask(
            rr.request.fields, build_pb2.Build.DESCRIPTOR
        )
      except ValueError as ex:
        rr.response.error.code = prpc.StatusCode.INVALID_ARGUMENT.value
        rr.response.error.message = 'invalid fields: %s' % ex.message
        continue

    valid_entries.append(
        (prepare_schedule_build_request_async(rr.request), rr.response, mask)
    )

  # Check permissions.
  def get_bucket_id(req):
    return config.format_bucket_id(req.builder.project, req.builder.bucket)

  valid_items = []
  for req_fut, res, mask in valid_entries:
    try:
      valid_items.append(_ScheduleItem(req_fut.get_result(), res, mask))
    except StatusError as ex:
      res.error.code = ex.code.value
      res.error.message = ex.message

  bucket_ids = {get_bucket_id(x.request) for x in valid_items}
  can_add = user.filter_buckets_by_perm(user.PERM_BUILDS_ADD, bucket_ids)
  identity_str = auth.get_current_identity().to_bytes()
  to_schedule = []
  for x in valid_items:
    bid = get_bucket_id(x.request)
    if bid in can_add:
      to_schedule.append(x)
    else:
      x.response.error.code = prpc.StatusCode.PERMISSION_DENIED.value
      x.response.error.message = (
          '%s cannot schedule builds in bucket %s' % (identity_str, bid)
      )

  # Schedule builds.
  if not to_schedule:  # pragma: no cover
    return
  build_requests = [
      creation.BuildRequest(schedule_build_request=x.request)
      for x in to_schedule
  ]
  results = creation.add_many_async(build_requests).get_result()
  futs = []
  for x, (build, ex) in zip(to_schedule, results):
    res = x.response
    err = res.error
    if isinstance(ex, errors.Error):
      err.code = ex.code.value
      err.message = ex.message
    elif isinstance(ex, auth.AuthorizationError):
      err.code = prpc.StatusCode.PERMISSION_DENIED.value
      err.message = ex.message
    elif ex:
      err.code = prpc.StatusCode.INTERNAL.value
      err.message = ex.message
    else:
      futs.append(build_to_proto_async(build, res.schedule_build, x.mask))
  for f in futs:
    f.get_result()


@rpc_impl_async('CancelBuild')
@ndb.tasklet
def cancel_build_async(_req, _res, _ctx, _mask):  # pragma: no cover
  raise unimplemented()


# Maps an rpc_pb2.BatchRequest.Request field name to an async function
#   (req, ctx) => ndb.Future of res.
BATCH_REQUEST_TYPE_TO_RPC_IMPL = {
    'get_build': get_build_async,
    'search_builds': search_builds_async,
    'cancel_build': cancel_build_async,
}
assert set(BATCH_REQUEST_TYPE_TO_RPC_IMPL) | {'schedule_build'} == set(
    rpc_pb2.BatchRequest.Request.DESCRIPTOR.fields_by_name
)


class BuildsApi(object):
  """Implements buildbucket.v2.Builds proto service."""

  # "mask" parameter in RPC implementations is added by rpc_impl_async.
  # pylint: disable=no-value-for-parameter

  DESCRIPTION = builds_service_prpc_pb2.BuildsServiceDescription

  def _res_if_ok(self, res, ctx):
    return res if ctx.code == prpc.StatusCode.OK else None

  def GetBuild(self, req, ctx):
    res = build_pb2.Build()
    get_build_async(req, res, ctx).get_result()
    return self._res_if_ok(res, ctx)

  def SearchBuilds(self, req, ctx):  # pragma: no cover
    res = rpc_pb2.SearchBuildsResponse()
    search_builds_async(req, res, ctx).get_result()
    return self._res_if_ok(res, ctx)

  def UpdateBuild(self, req, ctx):  # pragma: no cover
    res = build_pb2.Build()
    update_build_async(req, res, ctx).get_result()
    return self._res_if_ok(res, ctx)

  def ScheduleBuild(self, req, ctx):
    res = build_pb2.Build()
    schedule_build_async(req, res, ctx).get_result()
    return self._res_if_ok(res, ctx)

  def CancelBuild(self, req, ctx):  # pragma: no cover
    res = build_pb2.Build()
    cancel_build_async(req, res, ctx).get_result()
    return self._res_if_ok(res, ctx)

  def Batch(self, req, ctx):
    res = rpc_pb2.BatchResponse()
    batch = [_ReqRes(req, res.responses.add()) for req in req.requests]

    # First, execute ScheduleBuild requests.
    schedule_requests = []
    in_parallel = []
    seen_types = set()
    for rr in batch:
      request_type = rr.request.WhichOneof('request')
      if not request_type:
        rr.response.error.code = prpc.StatusCode.INVALID_ARGUMENT.value
        rr.response.error.message = 'request is not specified'
      elif request_type == 'schedule_build':
        schedule_requests.append(rr)
      else:
        in_parallel.append(rr)
      seen_types.add(request_type)
    if len(seen_types) > 1:  # pragma: no cover
      logging.info('Batch: detect multiple types of requests - %s', seen_types)
    if schedule_requests:
      schedule_build_multi([
          _ReqRes(rr.request.schedule_build, rr.response)
          for rr in schedule_requests
      ])

    # Then, execute the rest in parallel.

    @ndb.tasklet
    def serve_subrequest_async(rr):
      request_type = rr.request.WhichOneof('request')
      assert request_type != 'schedule_build'
      rpc_impl = BATCH_REQUEST_TYPE_TO_RPC_IMPL[request_type]
      sub_ctx = ctx.clone()
      yield rpc_impl(
          getattr(rr.request, request_type),
          getattr(rr.response, request_type),
          sub_ctx,
      )
      if sub_ctx.code != prpc.StatusCode.OK:  # pragma: no cover
        rr.response.ClearField(request_type)
        rr.response.error.code = sub_ctx.code.value
        rr.response.error.message = sub_ctx.details

    for f in map(serve_subrequest_async, in_parallel):
      f.check_success()

    assert all(r.WhichOneof('response') for r in res.responses), res.responses
    return self._res_if_ok(res, ctx)

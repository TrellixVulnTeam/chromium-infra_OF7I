# Copyright 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""User-related functions, including access control list implementation.

See Acl message in proto/project_config.proto.
"""

import collections
import logging
import os
import threading

from google.appengine.api import app_identity
from google.appengine.ext import ndb

from components import auth
from components import utils

from protorpc import messages
from go.chromium.org.luci.buildbucket.proto import project_config_pb2
import config
import errors

# Group whitelisting users to update builds. They are expected to be robots.
UPDATE_BUILD_ALLOWED_USERS = 'buildbucket-update-build-users'


################################################################################
## Permissions-based API (implemented in terms of Buildbucket roles for now).

ALL_PERMISSIONS = set()


def _permission(name):
  perm = auth.Permission(name)
  ALL_PERMISSIONS.add(perm)
  return perm


# Builds permissions.

# See all information about a build.
PERM_BUILDS_GET = _permission('buildbucket.builds.get')
# List and search builds in a bucket.
PERM_BUILDS_LIST = _permission('buildbucket.builds.list')
# Schedule new builds in the bucket.
PERM_BUILDS_ADD = _permission('buildbucket.builds.add')
# Cancel a build in the bucket.
PERM_BUILDS_CANCEL = _permission('buildbucket.builds.cancel')
# Lease and control a build via v1 API, deprecated.
PERM_BUILDS_LEASE = _permission('buildbucket.builds.lease')
# Unlease and reset state of an existing build via v1 API, deprecated.
PERM_BUILDS_RESET = _permission('buildbucket.builds.reset')

# Builders permissions.

# List and search builders (but not builds).
PERM_BUILDERS_LIST = _permission('buildbucket.builders.list')
# Set the next build number.
PERM_BUILDERS_SET_NUM = _permission('buildbucket.builders.setBuildNumber')

# Bucket permissions.

# See existence of a bucket, used only by v1 APIs, deprecated.
PERM_BUCKETS_GET = _permission('buildbucket.buckets.get')
# Delete all scheduled builds from a bucket.
PERM_BUCKETS_DELETE_BUILDS = _permission('buildbucket.buckets.deleteBuilds')
# Pause/resume leasing builds in a bucket via v1 API, deprecated.
PERM_BUCKETS_PAUSE = _permission('buildbucket.buckets.pause')

# Forbid adding more permission from other modules or tests after this point.
ALL_PERMISSIONS = frozenset(ALL_PERMISSIONS)

# Maps a Permission to a minimum required project_config_pb2.Acl.Role.
PERM_TO_MIN_ROLE = {
    # Reader.
    PERM_BUILDS_GET: project_config_pb2.Acl.READER,
    PERM_BUILDS_LIST: project_config_pb2.Acl.READER,
    PERM_BUILDERS_LIST: project_config_pb2.Acl.READER,
    PERM_BUCKETS_GET: project_config_pb2.Acl.READER,

    # Scheduler.
    PERM_BUILDS_ADD: project_config_pb2.Acl.SCHEDULER,
    PERM_BUILDS_CANCEL: project_config_pb2.Acl.SCHEDULER,

    # Writer.
    PERM_BUILDS_LEASE: project_config_pb2.Acl.WRITER,
    PERM_BUILDS_RESET: project_config_pb2.Acl.WRITER,
    PERM_BUILDERS_SET_NUM: project_config_pb2.Acl.WRITER,
    PERM_BUCKETS_DELETE_BUILDS: project_config_pb2.Acl.WRITER,
    PERM_BUCKETS_PAUSE: project_config_pb2.Acl.WRITER,
}
assert sorted(PERM_TO_MIN_ROLE.keys()) == sorted(ALL_PERMISSIONS)


@ndb.tasklet
def has_perm_async(perm, bucket_id):
  """Returns True if the caller has the given permission in the bucket.

  Args:
    perm: an instance of auth.Permission.
    bucket_id: a bucket ID string, i.e. "<project>/<bucket>".
  """
  assert isinstance(perm, auth.Permission), perm
  assert perm in ALL_PERMISSIONS, perm
  config.validate_bucket_id(bucket_id)
  role = yield get_role_async_deprecated(bucket_id)
  raise ndb.Return(role is not None and role >= PERM_TO_MIN_ROLE[perm])


def has_perm(perm, bucket_id):
  """Returns True if the caller has the given permission in the bucket.

  Args:
    perm: an instance of auth.Permission.
    bucket_id: a bucket ID string, i.e. "<project>/<bucket>".
  """
  return has_perm_async(perm, bucket_id).get_result()


def filter_buckets_by_perm(perm, bucket_ids):
  """Filters given buckets keeping only ones the caller has the permission in.

  Note that this function is not async!

  Args:
    perm: an instance of auth.Permission.
    bucket_ids: an iterable with bucket IDs.

  Returns:
    A set of bucket IDs.
  """
  pairs = utils.async_apply(
      bucket_ids if isinstance(bucket_ids, set) else set(bucket_ids),
      lambda bid: has_perm_async(perm, bid),
      unordered=True,
  )
  return {bid for bid, has_perm in pairs if has_perm}


def buckets_by_perm_async(perm):
  """Returns buckets that the caller has the given permission in.

  Results are memcached for 10 minutes per (identity, perm) pair.

  Args:
    perm: an instance of auth.Permission.

  Returns:
    A set of bucket IDs.
  """
  assert isinstance(perm, auth.Permission), perm
  assert perm in ALL_PERMISSIONS, perm

  identity = auth.get_current_identity()
  identity_str = identity.to_bytes()

  @ndb.tasklet
  def impl():
    ctx = ndb.get_context()
    cache_key = 'buckets_by_perm/%s/%s' % (identity_str, perm)
    matching_buckets = yield ctx.memcache_get(cache_key)
    if matching_buckets is not None:
      raise ndb.Return(matching_buckets)

    logging.info('Computing a set of buckets %r has %r in', identity_str, perm)
    all_buckets = yield config.get_all_bucket_ids_async()
    per_bucket = yield [has_perm_async(perm, bid) for bid in all_buckets]
    matching_buckets = {bid for bid, has in zip(all_buckets, per_bucket) if has}

    # Cache for 10 min
    yield ctx.memcache_set(cache_key, matching_buckets, 10 * 60)
    raise ndb.Return(matching_buckets)

  return _get_or_create_cached_future(
      identity, 'buckets_by_perm/%s' % perm, impl
  )


################################################################################
## Role definitions (DEPRECATED).


class Action(messages.Enum):
  # Schedule a build.
  ADD_BUILD = 1
  # Get information about a build.
  VIEW_BUILD = 2
  # Lease a build for execution. Normally done by build systems.
  LEASE_BUILD = 3
  # Cancel an existing build. Does not require a lease key.
  CANCEL_BUILD = 4
  # Unlease and reset state of an existing build. Normally done by admins.
  RESET_BUILD = 5
  # Search for builds or get a list of scheduled builds.
  SEARCH_BUILDS = 6
  # Delete all scheduled builds from a bucket.
  DELETE_SCHEDULED_BUILDS = 9
  # Know about bucket existence and read its info.
  ACCESS_BUCKET = 10
  # Pause builds for a given bucket.
  PAUSE_BUCKET = 11
  # Set the number for the next build in a builder.
  SET_NEXT_NUMBER = 12


# Maps an Action to a description.
ACTION_DESCRIPTIONS = {
    Action.ADD_BUILD:
        'Schedule a build.',
    Action.VIEW_BUILD:
        'Get information about a build.',
    Action.LEASE_BUILD:
        'Lease a build for execution.',
    Action.CANCEL_BUILD:
        'Cancel an existing build. Does not require a lease key.',
    Action.RESET_BUILD:
        'Unlease and reset state of an existing build.',
    Action.SEARCH_BUILDS:
        'Search for builds or get a list of scheduled builds.',
    Action.DELETE_SCHEDULED_BUILDS:
        'Delete all scheduled builds from a bucket.',
    Action.ACCESS_BUCKET:
        'Know about a bucket\'s existence and read its info.',
    Action.PAUSE_BUCKET:
        'Pause builds for a given bucket.',
    Action.SET_NEXT_NUMBER:
        'Set the number for the next build in a builder.',
}

# Maps an Action to a minimum project_config_pb2.Acl.Role required for the
# action.
ACTION_TO_MIN_ROLE = {
    # Reader.
    Action.ACCESS_BUCKET: project_config_pb2.Acl.READER,
    Action.VIEW_BUILD: project_config_pb2.Acl.READER,
    Action.SEARCH_BUILDS: project_config_pb2.Acl.READER,
    # Scheduler.
    Action.ADD_BUILD: project_config_pb2.Acl.SCHEDULER,
    Action.CANCEL_BUILD: project_config_pb2.Acl.SCHEDULER,
    # Writer.
    Action.LEASE_BUILD: project_config_pb2.Acl.WRITER,
    Action.RESET_BUILD: project_config_pb2.Acl.WRITER,
    Action.DELETE_SCHEDULED_BUILDS: project_config_pb2.Acl.WRITER,
    Action.PAUSE_BUCKET: project_config_pb2.Acl.WRITER,
    Action.SET_NEXT_NUMBER: project_config_pb2.Acl.WRITER,
}

# Maps a project_config_pb2.Acl.Role to a set of permitted Actions.
ROLE_TO_ACTIONS = {
    r:
    tuple(sorted({a for a, mr in ACTION_TO_MIN_ROLE.iteritems() if r >= mr}))
    for r in project_config_pb2.Acl.Role.values()
}

################################################################################
## Granular actions. API uses these.


@ndb.tasklet
def can_update_build_async():  # pragma: no cover
  """Returns if the current identity is whitelisted to update builds."""
  # TODO(crbug.com/1091604): Implementing using has_perm_async.
  raise ndb.Return(auth.is_group_member(UPDATE_BUILD_ALLOWED_USERS))


################################################################################
## Implementation.


def get_role_async_deprecated(bucket_id):
  """Returns the most permissive role of the current user in |bucket_id|.

  The most permissive role is the role that allows most actions, e.g. WRITER
  is more permissive than READER.

  Returns None if there's no such bucket or the current identity has no roles in
  it at all.
  """
  config.validate_bucket_id(bucket_id)

  identity = auth.get_current_identity()
  identity_str = identity.to_bytes()

  @ndb.tasklet
  def impl():
    ctx = ndb.get_context()
    cache_key = 'role/%s/%s' % (identity_str, bucket_id)
    cache = yield ctx.memcache_get(cache_key)
    if cache is not None:
      raise ndb.Return(cache[0])

    _, bucket_cfg = yield config.get_bucket_async(bucket_id)
    if not bucket_cfg:
      raise ndb.Return(None)
    if auth.is_admin(identity):
      raise ndb.Return(project_config_pb2.Acl.WRITER)

    # A LUCI service calling us in the context of some project is allowed to
    # do anything it wants in that project. We trust all LUCI services to do
    # authorization on their own for this case. A cross-project request must be
    # explicitly authorized in Buildbucket ACLs though (so we proceed to the
    # bucket_cfg check below).
    if identity.is_project:
      project_id, _ = config.parse_bucket_id(bucket_id)
      if project_id == identity.name:
        raise ndb.Return(project_config_pb2.Acl.WRITER)

    # Roles are just numbers. The higher the number, the more permissions
    # the identity has. We exploit this here to get the single maximally
    # permissive role for the current identity.
    role = None
    for rule in bucket_cfg.acls:
      if rule.role <= role:
        continue
      if (rule.identity == identity_str or
          (rule.group and auth.is_group_member(rule.group, identity))):
        role = rule.role
    yield ctx.memcache_set(cache_key, (role,), time=60)
    raise ndb.Return(role)

  return _get_or_create_cached_future(identity, 'role/%s' % bucket_id, impl)


@ndb.tasklet
def permitted_actions_async(bucket_id):
  """Returns a tuple of actions (as Action enums) permitted to the caller."""
  # TODO(crbug.com/1091604): Implementing in terms of has_perm_async.
  role = yield get_role_async_deprecated(bucket_id)
  if role is None:
    raise ndb.Return(())
  raise ndb.Return(ROLE_TO_ACTIONS[role])


@utils.cache
def self_identity():  # pragma: no cover
  """Returns identity of the buildbucket app."""
  return auth.Identity('user', app_identity.get_service_account_name())


def delegate_async(target_service_host, identity=None, tag=''):
  """Mints a delegation token for the current identity."""
  tag = tag or ''
  identity = identity or auth.get_current_identity()

  # TODO(vadimsh): 'identity' here can be 'project:<...>' and we happily create
  # a delegation token for it, which is weird. Buildbucket should call Swarming
  # using 'project:<...>' identity directly, not through a delegation token.

  def impl():
    return auth.delegate_async(
        audience=[self_identity()],
        services=['https://%s' % target_service_host],
        impersonate=identity,
        tags=[tag] if tag else [],
    )

  return _get_or_create_cached_future(
      identity, 'delegation_token:%s:%s' % (target_service_host, tag), impl
  )


def current_identity_cannot(action_format, *args):  # pragma: no cover
  """Returns AuthorizationError."""
  action = action_format % args
  msg = 'User %s cannot %s' % (auth.get_current_identity().to_bytes(), action)
  logging.warning(msg)
  return auth.AuthorizationError(msg)


def parse_identity(identity):
  """Parses an identity string if it is a string."""
  if isinstance(identity, basestring):
    if not identity:  # pragma: no cover
      return None
    if ':' not in identity:  # pragma: no branch
      identity = 'user:%s' % identity
    try:
      identity = auth.Identity.from_bytes(identity)
    except ValueError as ex:
      raise errors.InvalidInputError('Invalid identity: %s' % ex)
  return identity


_thread_local = threading.local()


def _get_or_create_cached_future(identity, key, create_future):
  """Returns a future cached in the current GAE request context.

  Uses the pair (identity, key) as the caching key.

  Using this function may cause RuntimeError with a deadlock if the returned
  future is not waited for before leaving an ndb context, but that's a bug
  in the first place.
  """
  assert isinstance(identity, auth.Identity), identity
  full_key = (identity, key)

  # Docs:
  # https://cloud.google.com/appengine/docs/standard/python/how-requests-are-handled#request-ids
  req_id = os.environ['REQUEST_LOG_ID']
  cache = getattr(_thread_local, 'request_cache', {})
  if cache.get('request_id') != req_id:
    cache = {
        'request_id': req_id,
        'futures': {},
    }
    _thread_local.request_cache = cache

  fut_entry = cache['futures'].get(full_key)
  if fut_entry is None:
    fut_entry = {
        'future': create_future(),
        'ndb_context': ndb.get_context(),
    }
    cache['futures'][full_key] = fut_entry
  assert (
      fut_entry['future'].done() or
      ndb.get_context() is fut_entry['ndb_context']
  )
  return fut_entry['future']


def clear_request_cache():
  _thread_local.request_cache = {}

# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Access to bucket configurations.

Stores bucket list in datastore, synchronizes it with bucket configs in
project repositories: `projects/<project_id>:<buildbucket-app-id>.cfg`.
"""

import collections
import copy
import hashlib
import logging
import re

from google.appengine.api import app_identity
from google.appengine.api import memcache
from google.appengine.ext import ndb

from components import auth
from components import config
from components import datastore_utils
from components import gitiles
from components import utils
from components.config import validation

from go.chromium.org.luci.buildbucket.proto import project_config_pb2
from go.chromium.org.luci.buildbucket.proto import service_config_pb2
import errors

CURRENT_BUCKET_SCHEMA_VERSION = 13
ACL_SET_NAME_RE = re.compile('^[a-z0-9_]+$')

# The memcache key for get_all_bucket_ids_async cache.
_MEMCACHE_ALL_BUCKET_IDS_KEY = 'all_bucket_ids_v1'
# Expiration time for get_all_bucket_ids_async cache.
_MEMCACHE_ALL_BUCKET_IDS_EXP = 90


@utils.cache
def cfg_path():
  """Returns relative buildbucket config file path."""
  try:
    appid = app_identity.get_application_id()
  except AttributeError:  # pragma: no cover | does not get run on some bots
    # Raised in testbed environment because cfg_path is called
    # during decoration.
    appid = 'testbed-test'
  return '%s.cfg' % appid


@utils.cache
def self_config_set():
  """Returns buildbucket's service config set."""
  try:
    return config.self_config_set()
  except AttributeError:  # pragma: no cover | does not get run on some bots
    # Raised in testbed environment because cfg_path is called
    # during decoration.
    return 'services/testbed-test'


def validate_identity(identity, ctx):
  if ':' in identity:
    kind, name = identity.split(':', 2)
  else:
    kind = 'user'
    name = identity
  try:
    auth.Identity(kind, name)
  except ValueError as ex:
    ctx.error('%s', ex)


def validate_access_list(acl_list, ctx):
  """Validates a list of Acl messages."""
  for i, acl in enumerate(acl_list):
    with ctx.prefix('acl #%d: ', i + 1):
      if acl.group and acl.identity:
        ctx.error('either group or identity must be set, not both')
      elif acl.group:
        if not auth.is_valid_group_name(acl.group):
          ctx.error('invalid group: %s', acl.group)
      elif acl.identity:
        validate_identity(acl.identity, ctx)
      else:
        ctx.error('group or identity must be set')


@validation.project_config_rule(cfg_path(), project_config_pb2.BuildbucketCfg)
def validate_buildbucket_cfg(cfg, ctx):
  import swarmingcfg

  global_cfg = get_settings_async().get_result()
  well_known_experiments = set(
      e.name for e in global_cfg.experiment.experiments
  )

  acl_set_names = set()
  for i, acl_set in enumerate(cfg.acl_sets):
    with ctx.prefix('ACL set #%d (%s): ', i + 1, acl_set.name):
      if not acl_set.name:
        ctx.error('name is unspecified')
      elif not ACL_SET_NAME_RE.match(acl_set.name):
        ctx.error(
            'invalid name "%s" does not match regex %r', acl_set.name,
            ACL_SET_NAME_RE.pattern
        )
      elif acl_set.name in acl_set_names:
        ctx.error('duplicate name "%s"', acl_set.name)
      acl_set_names.add(acl_set.name)

      validate_access_list(acl_set.acls, ctx)

  bucket_names = set()

  for i, bucket in enumerate(cfg.buckets):
    with ctx.prefix('Bucket %s: ', bucket.name or ('#%d' % (i + 1))):
      try:
        errors.validate_bucket_name(bucket.name, project_id=ctx.project_id)
      except errors.InvalidInputError as ex:
        ctx.error('invalid name: %s', ex.message)
      else:
        if bucket.name in bucket_names:
          ctx.error('duplicate bucket name')
        else:
          bucket_names.add(bucket.name)
          if i > 0 and bucket.name < cfg.buckets[i - 1].name:
            ctx.warning('out of order')

      validate_access_list(bucket.acls, ctx)
      for name in bucket.acl_sets:
        if name not in acl_set_names:
          ctx.error(
              'undefined ACL set "%s". '
              'It must be defined in the same file', name
          )

      if bucket.HasField('swarming'):  # pragma: no cover
        with ctx.prefix('swarming: '):
          swarmingcfg.validate_project_cfg(
              bucket.swarming, well_known_experiments, ctx
          )


@validation.rule(
    self_config_set(), 'settings.cfg', service_config_pb2.SettingsCfg
)
def validate_settings_cfg(cfg, ctx):  # pragma: no cover
  import swarmingcfg

  if cfg.HasField('swarming'):
    with ctx.prefix('swarming: '):
      swarmingcfg.validate_service_cfg(cfg.swarming, ctx)

  if not cfg.logdog.hostname:
    ctx.error('logdog: hostname is required')

  if not cfg.resultdb.hostname:
    ctx.error('resultdb: hostname is required')


class Project(ndb.Model):
  """Parent entity for Bucket.

  Does not exist in the datastore.

  Entity key:
    Root entity. ID is project id.
  """


def is_legacy_bucket_id(bucket_id):
  return '/' not in bucket_id


def format_bucket_id(project_id, bucket_name):  # pragma: no cover
  """Returns a bucket id string."""
  return '%s/%s' % (project_id, bucket_name)


def parse_bucket_id(bucket_id):
  """Returns a (project_id, bucket_name) tuple."""
  parts = bucket_id.split('/', 1)
  assert len(parts) == 2
  return tuple(parts)


def validate_project_id(project_id):
  """Raises errors.InvalidInputError if project_id is invalid."""
  if not validation.is_valid_project_id(project_id):
    raise errors.InvalidInputError('invalid project_id %r' % project_id)


validate_bucket_name = errors.validate_bucket_name


def validate_bucket_id(bucket_id):
  """Raises errors.InvalidInputError if bucket_id is invalid."""
  assert not is_legacy_bucket_id(bucket_id)

  try:
    project_id, bucket_name = parse_bucket_id(bucket_id)
    validate_project_id(project_id)
    validate_bucket_name(bucket_name)
  except errors.InvalidInputError as ex:
    raise errors.InvalidInputError('invalid bucket_id %r: %s' % (bucket_id, ex))

  parts = bucket_name.split('.', 2)
  if len(parts) == 3 and parts[0] == 'luci' and parts[1] == project_id:
    expected_bucket_id = '%s/%s' % (project_id, parts[2])
    raise errors.InvalidInputError(
        'invalid bucket_id string %r. Did you mean %r?' %
        (bucket_id, expected_bucket_id)
    )


class Bucket(ndb.Model):
  """Stores bucket configurations.

  Bucket entities are updated in cron_update_buckets() from project configs.

  Entity key:
    Parent is Project. Id is a "short" bucket name.
    See also bucket_name attribute and short_bucket_name().
  """

  @classmethod
  def _get_kind(cls):
    return 'BucketV2'

  # Bucket name not prefixed by project id.
  # For example "try" or "master.x".
  #
  # If a bucket in a config file has "luci.<project_id>." prefix, the
  # prefix is stripped, e.g. "try", not "luci.chromium.try".
  bucket_name = ndb.ComputedProperty(lambda self: self.key.id())
  # Version of entity schema. If not current, cron_update_buckets will update
  # the entity forcefully.
  entity_schema_version = ndb.IntegerProperty()
  # Bucket revision matches its config revision.
  revision = ndb.StringProperty(required=True)
  # Binary equivalent of config_content.
  config = datastore_utils.ProtobufProperty(project_config_pb2.Bucket)

  def _pre_put_hook(self):
    assert self.config.name == self.key.id()

  @property
  def project_id(self):
    return self.key.parent().id()

  @staticmethod
  def make_key(project_id, bucket_name):
    return ndb.Key(Project, project_id, Bucket, bucket_name)

  @staticmethod
  def key_to_bucket_id(key):
    return format_bucket_id(key.parent().id(), key.id())

  @property
  def bucket_id(self):
    return format_bucket_id(self.project_id, self.bucket_name)


class Builder(ndb.Model):
  """Stores builder configuration.

  Updated in cron_update_buckets() along with buckets.

  Entity key:
    Parent is Bucket. Id is a builder name.
  """

  @classmethod
  def _get_kind(cls):
    # "Builder" conflicts with model.Builder.
    return 'Bucket.Builder'

  @classmethod
  def _use_memcache(cls, _):
    return False

  # Binary config content.
  config = datastore_utils.ProtobufProperty(project_config_pb2.Builder)
  # Hash used for fast deduplication of configs. Set automatically on put.
  config_hash = ndb.StringProperty(required=True)

  @staticmethod
  def compute_hash(cfg):
    """Computes a hash for a builder config."""
    return hashlib.sha256(cfg.SerializeToString(deterministic=True)).hexdigest()

  def _pre_put_hook(self):
    assert self.config.name == self.key.id()
    assert not self.config_hash
    self.config_hash = self.compute_hash(self.config)

  @staticmethod
  def make_key(project_id, bucket_name, builder_name):
    return ndb.Key(
        Project, project_id, Bucket, bucket_name, Builder, builder_name
    )


def short_bucket_name(bucket_name):
  """Returns bucket name without "luci.<project_id>." prefix."""
  parts = bucket_name.split('.', 2)
  if len(parts) == 3 and parts[0] == 'luci':
    return parts[2]
  return bucket_name


def is_swarming_config(cfg):
  """Returns True if this is a Swarming bucket config."""
  return cfg and cfg.HasField('swarming')


@ndb.non_transactional
@ndb.tasklet
def get_all_bucket_ids_async():
  """Returns a sorted list of all defined bucket IDs."""
  ctx = ndb.get_context()
  ids = yield ctx.memcache_get(_MEMCACHE_ALL_BUCKET_IDS_KEY)
  if ids is None:
    keys = yield Bucket.query().fetch_async(keys_only=True)
    ids = sorted(Bucket.key_to_bucket_id(key) for key in keys)
    yield ctx.memcache_set(
        _MEMCACHE_ALL_BUCKET_IDS_KEY, ids, _MEMCACHE_ALL_BUCKET_IDS_EXP
    )
  raise ndb.Return(ids)


@ndb.non_transactional
@ndb.tasklet
def get_buckets_async(bucket_ids=None, include_builders=False):
  """Returns configured buckets.

  If bucket_ids is None, returns all buckets.
  Otherwise returns only specified buckets.
  If a bucket does not exist, returns a None map value.
  By default, builder configs are omitted.

  Returns:
    {bucket_id: project_config_pb2.Bucket} dict.
  """
  cfgs = {}
  if bucket_ids is not None:
    bucket_ids = list(bucket_ids)
    keys = [Bucket.make_key(*parse_bucket_id(bid)) for bid in bucket_ids]
    cfgs = {bid: None for bid in bucket_ids}
    buckets = yield ndb.get_multi_async(keys)
  else:
    buckets = yield Bucket.query().fetch_async()
  cfgs.update({b.bucket_id: b.config for b in buckets if b})

  builders = {}  # bucket_id -> [builders]
  if include_builders:
    futures = {}
    for bucket_id in cfgs:
      bucket_key = Bucket.make_key(*parse_bucket_id(bucket_id))
      futures[bucket_id] = Builder.query(ancestor=bucket_key).fetch_async()
    for bucket_id, f in futures.iteritems():
      builders[bucket_id] = f.get_result()

  for bucket_id, cfg in cfgs.iteritems():
    if is_swarming_config(cfg):
      del cfg.swarming.builders[:]
      cfg.swarming.builders.extend(
          sorted([b.config for b in builders.get(bucket_id, [])],
                 key=lambda b: b.name)
      )

  raise ndb.Return(cfgs)


@utils.memcache_async(
    'resolve_bucket_name_async', key_args=['bucket_name'], time=300
)  # memcache for 5m
@ndb.non_transactional
@ndb.tasklet
def resolve_bucket_name_async(bucket_name):
  """Returns bucket id for the bucket name.

  Does not check access.

  Raises:
    errors.InvalidInputError if the bucket name is not unique.

  Returns:
    bucket id string or None if such bucket does not exist.
  """
  buckets = yield Bucket.query(Bucket.bucket_name == bucket_name).fetch_async()
  if len(buckets) > 1:
    raise errors.InvalidInputError(
        'bucket name %r is ambiguous, '
        'it has to be prefixed with a project id: "<project_id>/%s"' %
        (bucket_name, bucket_name)
    )
  raise ndb.Return(buckets[0].bucket_id if buckets else None)


@ndb.non_transactional
@ndb.tasklet
def get_bucket_async(bucket_id):
  """Returns a (revision, project_config_pb2.Bucket) tuple."""
  key = Bucket.make_key(*parse_bucket_id(bucket_id))
  bucket = yield key.get_async()
  if bucket is None:
    raise ndb.Return(None, None)
  if is_swarming_config(bucket.config):  # pragma: no cover
    del bucket.config.swarming.builders[:]
  raise ndb.Return(bucket.revision, bucket.config)


@ndb.non_transactional
def get_bucket(bucket_id):
  """Returns a (revision, project_config_pb2.Bucket) tuple."""
  return get_bucket_async(bucket_id).get_result()


def _normalize_acls(acls):
  """Normalizes a RepeatedCompositeContainer of Acl messages."""
  for a in acls:
    if a.identity and ':' not in a.identity:
      a.identity = 'user:%s' % a.identity

  sort_key = lambda a: (a.role, a.group, a.identity)
  acls.sort(key=sort_key)

  for i in xrange(len(acls) - 1, 0, -1):
    if sort_key(acls[i]) == sort_key(acls[i - 1]):
      del acls[i]


def put_bucket(project_id, revision, bucket_cfg):
  # New Bucket format uses short bucket names, e.g. "try" instead of
  # "luci.chromium.try".
  # Use short name in both entity key and config contents.
  short_bucket_cfg = copy.deepcopy(bucket_cfg)
  short_bucket_cfg.name = short_bucket_name(short_bucket_cfg.name)
  # Trim builders. They're stored in separate Builder entities.
  if is_swarming_config(short_bucket_cfg):
    del short_bucket_cfg.swarming.builders[:]
  Bucket(
      key=Bucket.make_key(project_id, short_bucket_cfg.name),
      entity_schema_version=CURRENT_BUCKET_SCHEMA_VERSION,
      revision=revision,
      config=short_bucket_cfg,
  ).put()


def put_builders(project_id, bucket_name, *builder_cfgs):
  builders = [
      Builder(key=Builder.make_key(project_id, bucket_name, b.name), config=b)
      for b in builder_cfgs
  ]
  ndb.put_multi(builders)


def cron_update_buckets():
  """Synchronizes bucket entities with configs fetched from luci-config.

  When storing in the datastore, inlines the referenced ACL sets and clears
  the acl_sets message field.
  """
  config_map = config.get_project_configs(
      cfg_path(), project_config_pb2.BuildbucketCfg
  )

  # Initial set of ndb.Bucket keys present in the datastore. Will be updated
  # alongside datastore mutations below to get the final set of keys. It will be
  # use to refresh get_all_bucket_ids_async cache. Note that we can't just redo
  # the query at the end, since it is eventually consistent and may miss
  # recently added entities.
  buckets_in_datastore = set(Bucket.query().fetch(keys_only=True))

  buckets_to_delete = collections.defaultdict(set)  # project_id -> ndb keys
  for key in buckets_in_datastore:
    buckets_to_delete[key.parent().id()].add(key)

  for project_id, (revision, project_cfg, _) in config_map.iteritems():
    if project_cfg is None:
      logging.error('config of project %s is broken', project_id)
      # Do not delete all buckets of a broken project.
      buckets_to_delete.pop(project_id, None)
      continue

    # revision is None in file-system mode. Use SHA1 of the config as revision.
    revision = revision or 'sha1:%s' % hashlib.sha1(
        project_cfg.SerializeToString(deterministic=True)
    ).hexdigest()
    acl_sets_by_name = {a.name: a for a in project_cfg.acl_sets}

    for bucket_cfg in project_cfg.buckets:
      bucket_key = Bucket.make_key(
          project_id, short_bucket_name(bucket_cfg.name)
      )
      buckets_to_delete[project_id].discard(bucket_key)
      bucket = bucket_key.get()
      if (bucket and
          bucket.entity_schema_version == CURRENT_BUCKET_SCHEMA_VERSION and
          bucket.revision == revision):
        continue

      # Inline ACL sets.
      for name in bucket_cfg.acl_sets:
        acl_set = acl_sets_by_name.get(name)
        if not acl_set:
          logging.error(
              'referenced acl_set not found.\n'
              'Bucket: %s\n'
              'ACL set name: %r\n'
              'Config revision: %r', bucket_key, name, revision
          )
          continue
        bucket_cfg.acls.extend(acl_set.acls)
      del bucket_cfg.acl_sets[:]

      _normalize_acls(bucket_cfg.acls)

      builders = {b.key: b for b in Builder.query(ancestor=bucket_key).fetch()}
      builders_to_delete = set(builders)
      builders_to_put = []
      if bucket_cfg.HasField('swarming'):
        for b in bucket_cfg.swarming.builders:
          builder_key = Builder.make_key(project_id, bucket_key.id(), b.name)
          builders_to_delete.discard(builder_key)
          builder = builders.get(builder_key)
          if not any(d.startswith('pool:') for d in b.dimensions):
            b.dimensions.append(
                'pool:luci.%s.%s' %
                (project_id, short_bucket_name(bucket_cfg.name))
            )
          if builder and builder.config_hash == Builder.compute_hash(b):
            continue
          builders_to_put.append(b)

      # TODO(crbug/917873): Remove builders from the bucket config before put.
      # Requires updating every location that needs builder info to fetch
      # builder entities instead of assuming bucket entities will contain them.

      # pylint: disable=no-value-for-parameter
      @ndb.transactional()
      def update_entities():
        bucket = bucket_key.get()
        if (bucket and
            bucket.entity_schema_version == CURRENT_BUCKET_SCHEMA_VERSION and
            bucket.revision == revision):  # pragma: no coverage
          return

        put_bucket(project_id, revision, bucket_cfg)
        put_builders(project_id, bucket_key.id(), *builders_to_put)
        ndb.delete_multi(list(builders_to_delete))
        logging.info('Updated bucket %s to revision %s', bucket_key, revision)

      update_entities()
      buckets_in_datastore.add(bucket_key)

  # Delete non-existent buckets (and all associated builders).
  for buckets in buckets_to_delete.itervalues():
    to_delete = list(buckets)
    for b in buckets:
      to_delete.extend(Builder.query(ancestor=b).fetch(keys_only=True))
    if to_delete:
      logging.warning('Deleting entities: %s', ' '.join(map(str, to_delete)))
      ndb.transaction(lambda: ndb.delete_multi(to_delete))
    buckets_in_datastore -= set(buckets)

  # Eagerly update get_all_bucket_ids_async cache. No big deal if this fails,
  # the cache will expire naturally at some point. This is just an optimization
  # to speed up bucket list updates.
  memcache.set(
      _MEMCACHE_ALL_BUCKET_IDS_KEY,
      sorted(Bucket.key_to_bucket_id(key) for key in buckets_in_datastore),
      _MEMCACHE_ALL_BUCKET_IDS_EXP
  )


def get_buildbucket_cfg_url(project_id):
  """Returns URL of a buildbucket config file in a project, or None."""
  config_url = config.get_config_set_location('projects/%s' % project_id)
  if config_url is None:  # pragma: no cover
    return None
  try:
    loc = gitiles.Location.parse(config_url)
  except ValueError:  # pragma: no cover
    logging.exception(
        'Not a valid Gitiles URL %r of project %s', config_url, project_id
    )
    return None
  return str(loc.join(cfg_path()))


@ndb.tasklet
def get_settings_async():  # pragma: no cover
  _, global_settings = yield config.get_self_config_async(
      'settings.cfg', service_config_pb2.SettingsCfg, store_last_good=True
  )
  raise ndb.Return(global_settings or service_config_pb2.SettingsCfg())


def builder_id_string(builder_id_message):  # pragma: no cover
  """Returns a canonical string representation of a BuilderID."""
  bid = builder_id_message
  return '%s/%s/%s' % (bid.project, bid.bucket, bid.builder)


def builder_matches(builder_id_msg, predicate):
  """Returns True iff builder_id_msg matches the predicate.

  Args:
    * builder_id_msg (builder_pb2.BuilderID)
    * predicate (service_config_pb2.BuilderPredicate)
  """
  builder_str = builder_id_string(builder_id_msg)

  def _matches(regex_list):
    for pattern in regex_list:
      try:
        if re.match('^%s$' % pattern, builder_str):
          return True
      except re.error:  # pragma: no cover
        logging.exception('Regex %r failed on %r', pattern, builder_str)
    return False

  if _matches(predicate.regex_exclude):
    return False
  return not predicate.regex or _matches(predicate.regex)

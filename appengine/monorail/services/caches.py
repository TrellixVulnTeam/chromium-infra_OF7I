# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Classes to manage cached values.

Monorail makes full use of the RAM of GAE frontends to reduce latency
and load on the database.

Even though these caches do invalidation, there are rare race conditions
that can cause a somewhat stale object to be retrieved from memcache and
then put into a RAM cache and used by a given GAE instance for some time.
So, we only use these caches for operations that can tolerate somewhat
stale data.  For example, displaying issues in a list or displaying brief
info about related issues.  We never use the cache to load objects as
part of a read-modify-save sequence because that could cause stored data
to revert to a previous state.
"""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import logging
import redis

from protorpc import protobuf

from google.appengine.api import memcache

import settings
from framework import framework_constants
from framework import redis_utils
from proto import tracker_pb2


DEFAULT_MAX_SIZE = 10000


class RamCache(object):
  """An in-RAM cache with distributed invalidation."""

  def __init__(self, cache_manager, kind, max_size=None):
    self.cache_manager = cache_manager
    self.kind = kind
    self.cache = {}
    self.max_size = max_size or DEFAULT_MAX_SIZE
    cache_manager.RegisterCache(self, kind)

  def CacheItem(self, key, item):
    """Store item at key in this cache, discarding a random item if needed."""
    if len(self.cache) >= self.max_size:
      self.cache.popitem()

    self.cache[key] = item

  def CacheAll(self, new_item_dict):
    """Cache all items in the given dict, dropping old items if needed."""
    if len(new_item_dict) >= self.max_size:
      logging.warn('Dumping the entire cache! %s', self.kind)
      self.cache = {}
    else:
      while len(self.cache) + len(new_item_dict) > self.max_size:
        self.cache.popitem()

    self.cache.update(new_item_dict)

  def GetItem(self, key):
    """Return the cached item if present, otherwise None."""
    return self.cache.get(key)

  def HasItem(self, key):
    """Return True if there is a value cached at the given key."""
    return key in self.cache

  def GetAll(self, keys):
    """Look up the given keys.

    Args:
      keys: a list of cache keys to look up.

    Returns:
      A pair: (hits_dict, misses_list) where hits_dict is a dictionary of
      all the given keys and the values that were found in the cache, and
      misses_list is a list of given keys that were not in the cache.
    """
    hits, misses = {}, []
    for key in keys:
      try:
        hits[key] = self.cache[key]
      except KeyError:
        misses.append(key)

    return hits, misses

  def LocalInvalidate(self, key):
    """Drop the given key from this cache, without distributed notification."""
    if key in self.cache:
      logging.info('Locally invalidating %r in kind=%r', key, self.kind)
    self.cache.pop(key, None)

  def Invalidate(self, cnxn, key):
    """Drop key locally, and append it to the Invalidate DB table."""
    self.InvalidateKeys(cnxn, [key])

  def InvalidateKeys(self, cnxn, keys):
    """Drop keys locally, and append them to the Invalidate DB table."""
    for key in keys:
      self.LocalInvalidate(key)
    if self.cache_manager:
      self.cache_manager.StoreInvalidateRows(cnxn, self.kind, keys)

  def LocalInvalidateAll(self):
    """Invalidate all keys locally: just start over with an empty dict."""
    logging.info('Locally invalidating all in kind=%r', self.kind)
    self.cache = {}

  def InvalidateAll(self, cnxn):
    """Invalidate all keys in this cache."""
    self.LocalInvalidateAll()
    if self.cache_manager:
      self.cache_manager.StoreInvalidateAll(cnxn, self.kind)


class ShardedRamCache(RamCache):
  """Specialized version of RamCache that stores values in parts.

  Instead of the cache keys being simple integers, they are pairs, e.g.,
  (project_id, shard_id).  Invalidation will invalidate all shards for
  a given main key, e.g, invalidating project_id 16 will drop keys
  (16, 0), (16, 1), (16, 2), ... (16, 9).
  """

  def __init__(self, cache_manager, kind, max_size=None, num_shards=10):
    super(ShardedRamCache, self).__init__(
        cache_manager, kind, max_size=max_size)
    self.num_shards = num_shards

  def LocalInvalidate(self, key):
    """Use the specified value to drop entries from the local cache."""
    logging.info('About to invalidate shared RAM keys %r',
                 [(key, shard_id) for shard_id in range(self.num_shards)
                  if (key, shard_id) in self.cache])
    for shard_id in range(self.num_shards):
      self.cache.pop((key, shard_id), None)


class ValueCentricRamCache(RamCache):
  """Specialized version of RamCache that stores values in InvalidateTable.

  This is useful for caches that have non integer keys.
  """

  def LocalInvalidate(self, value):
    """Use the specified value to drop entries from the local cache."""
    keys_to_drop = []
    # Loop through and collect all keys with the specified value.
    for k, v in self.cache.items():
      if v == value:
        keys_to_drop.append(k)
    for k in keys_to_drop:
      self.cache.pop(k, None)

  def InvalidateKeys(self, cnxn, keys):
    """Drop keys locally, and append their values to the Invalidate DB table."""
    # Find values to invalidate.
    values = [self.cache[key] for key in keys if self.cache.has_key(key)]
    if len(values) == len(keys):
      for value in values:
        self.LocalInvalidate(value)
      if self.cache_manager:
        self.cache_manager.StoreInvalidateRows(cnxn, self.kind, values)
    else:
      # If a value is not found in the cache then invalidate the whole cache.
      # This is done to ensure that we are not in an inconsistent state or in a
      # race condition.
      self.InvalidateAll(cnxn)


class AbstractTwoLevelCache(object):
  """A class to manage both RAM and secondary-caching layer to retrieve objects.

  Subclasses must implement the FetchItems() method to get objects from
  the database when both caches miss.
  """

  # When loading a huge number of issues from the database, do it in chunks
  # so as to avoid timeouts.
  _FETCH_BATCH_SIZE = 10000

  def __init__(
      self,
      cache_manager,
      kind,
      prefix,
      pb_class,
      max_size=None,
      use_redis=False,
      redis_client=None):

    self.cache = self._MakeCache(cache_manager, kind, max_size=max_size)
    self.prefix = prefix
    self.pb_class = pb_class

    if use_redis:
      self.redis_client = redis_client or redis_utils.CreateRedisClient()
      self.use_redis = redis_utils.VerifyRedisConnection(
          self.redis_client, msg=kind)
    else:
      self.redis_client = None
      self.use_redis = False

  def _MakeCache(self, cache_manager, kind, max_size=None):
    """Make the RAM cache and register it with the cache_manager."""
    return RamCache(cache_manager, kind, max_size=max_size)

  def CacheItem(self, key, value):
    """Add the given key-value pair to RAM and L2 cache."""
    self.cache.CacheItem(key, value)
    self._WriteToCache({key: value})

  def HasItem(self, key):
    """Return True if the given key is in the RAM cache."""
    return self.cache.HasItem(key)

  def GetAnyOnHandItem(self, keys, start=None, end=None):
    """Try to find one of the specified items in RAM."""
    if start is None:
      start = 0
    if end is None:
      end = len(keys)
    for i in range(start, end):
      key = keys[i]
      if self.cache.HasItem(key):
        return self.cache.GetItem(key)

    # Note: We could check L2 here too, but the round-trips to L2
    # are kind of slow. And, getting too many hits from L2 actually
    # fills our RAM cache too quickly and could lead to thrashing.

    return None

  def GetAll(self, cnxn, keys, use_cache=True, **kwargs):
    """Get values for the given keys from RAM, the L2 cache, or the DB.

    Args:
      cnxn: connection to the database.
      keys: list of integer keys to look up.
      use_cache: set to False to always hit the database.
      **kwargs: any additional keywords are passed to FetchItems().

    Returns:
      A pair: hits, misses.  Where hits is {key: value} and misses is
        a list of any keys that were not found anywhere.
    """
    if use_cache:
      result_dict, missed_keys = self.cache.GetAll(keys)
    else:
      result_dict, missed_keys = {}, list(keys)

    if missed_keys:
      if use_cache:
        cache_hits, missed_keys = self._ReadFromCache(missed_keys)
        result_dict.update(cache_hits)
        self.cache.CacheAll(cache_hits)

    while missed_keys:
      missed_batch = missed_keys[:self._FETCH_BATCH_SIZE]
      missed_keys = missed_keys[self._FETCH_BATCH_SIZE:]
      retrieved_dict = self.FetchItems(cnxn, missed_batch, **kwargs)
      result_dict.update(retrieved_dict)
      if use_cache:
        self.cache.CacheAll(retrieved_dict)
        self._WriteToCache(retrieved_dict)

    still_missing_keys = [key for key in keys if key not in result_dict]
    return result_dict, still_missing_keys

  def LocalInvalidateAll(self):
    self.cache.LocalInvalidateAll()

  def LocalInvalidate(self, key):
    self.cache.LocalInvalidate(key)

  def InvalidateKeys(self, cnxn, keys):
    """Drop the given keys from both RAM and L2 cache."""
    self.cache.InvalidateKeys(cnxn, keys)
    self._DeleteFromCache(keys)

  def InvalidateAllKeys(self, cnxn, keys):
    """Drop the given keys from L2 cache and invalidate all keys in RAM.

    Useful for avoiding inserting many rows into the Invalidate table when
    invalidating a large group of keys all at once. Only use when necessary.
    """
    self.cache.InvalidateAll(cnxn)
    self._DeleteFromCache(keys)

  def GetAllAlreadyInRam(self, keys):
    """Look only in RAM to return {key: values}, missed_keys."""
    result_dict, missed_keys = self.cache.GetAll(keys)
    return result_dict, missed_keys

  def InvalidateAllRamEntries(self, cnxn):
    """Drop all RAM cache entries. It will refill as needed from L2 cache."""
    self.cache.InvalidateAll(cnxn)

  def FetchItems(self, cnxn, keys, **kwargs):
    """On RAM and L2 cache miss, hit the database."""
    raise NotImplementedError()

  def _ReadFromCache(self, keys):
    # type: (Sequence[int]) -> Mapping[str, Any], Sequence[int]
    """Reads a list of keys from secondary caching service.

    Redis will be used if Redis is enabled and connection is valid;
    otherwise, memcache will be used.

    Args:
      keys: List of integer keys to look up in L2 cache.

    Returns:
      A pair: hits, misses.  Where hits is {key: value} and misses is
        a list of any keys that were not found anywhere.
    """
    if self.use_redis:
      return self._ReadFromRedis(keys)
    else:
      return self._ReadFromMemcache(keys)

  def _WriteToCache(self, retrieved_dict):
    # type: (Mapping[int, Any]) -> None
    """Writes a set of key-value pairs to secondary caching service.

    Redis will be used if Redis is enabled and connection is valid;
    otherwise, memcache will be used.

    Args:
      retrieved_dict: Dictionary contains pairs of key-values to write to cache.
    """
    if self.use_redis:
      return self._WriteToRedis(retrieved_dict)
    else:
      return self._WriteToMemcache(retrieved_dict)

  def _DeleteFromCache(self, keys):
    # type: (Sequence[int]) -> None
    """Selects which cache to delete from.

    Redis will be used if Redis is enabled and connection is valid;
    otherwise, memcache will be used.

    Args:
      keys: List of integer keys to delete from cache.
    """
    if self.use_redis:
      return self._DeleteFromRedis(keys)
    else:
      return self._DeleteFromMemcache(keys)

  def _ReadFromMemcache(self, keys):
    # type: (Sequence[int]) -> Mapping[str, Any], Sequence[int]
    """Read the given keys from memcache, return {key: value}, missing_keys."""
    cache_hits = {}
    cached_dict = memcache.get_multi(
        [self._KeyToStr(key) for key in keys],
        key_prefix=self.prefix,
        namespace=settings.memcache_namespace)

    for key_str, serialized_value in cached_dict.items():
      value = self._StrToValue(serialized_value)
      key = self._StrToKey(key_str)
      cache_hits[key] = value
      self.cache.CacheItem(key, value)

    still_missing_keys = [key for key in keys if key not in cache_hits]
    return cache_hits, still_missing_keys

  def _WriteToMemcache(self, retrieved_dict):
    # type: (Mapping[int, int]) -> None
    """Write entries for each key-value pair to memcache.  Encode PBs."""
    strs_to_cache = {
        self._KeyToStr(key): self._ValueToStr(value)
        for key, value in retrieved_dict.items()}

    try:
      memcache.add_multi(
          strs_to_cache,
          key_prefix=self.prefix,
          time=framework_constants.CACHE_EXPIRATION,
          namespace=settings.memcache_namespace)
    except ValueError as identifier:
      # If memcache does not accept the values, ensure that no stale
      # values are left, then bail out.
      logging.error('Got memcache error: %r', identifier)
      self._DeleteFromMemcache(list(strs_to_cache.keys()))
      return

  def _DeleteFromMemcache(self, keys):
    # type: (Sequence[str]) -> None
    """Delete key-values from memcache. """
    memcache.delete_multi(
        [self._KeyToStr(key) for key in keys],
        seconds=5,
        key_prefix=self.prefix,
        namespace=settings.memcache_namespace)

  def _WriteToRedis(self, retrieved_dict):
    # type: (Mapping[int, Any]) -> None
    """Write entries for each key-value pair to Redis.  Encode PBs.

    Args:
      retrieved_dict: Dictionary of key-value pairs to write to Redis.
    """
    try:
      for key, value in retrieved_dict.items():
        redis_key = redis_utils.FormatRedisKey(key, prefix=self.prefix)
        redis_value = self._ValueToStr(value)

        self.redis_client.setex(
            redis_key, framework_constants.CACHE_EXPIRATION, redis_value)
    except redis.RedisError as identifier:
      logging.error(
          'Redis error occurred during write operation: %s', identifier)
      self._DeleteFromRedis(list(retrieved_dict.keys()))
      return
    logging.info(
        'cached batch of %d values in redis %s', len(retrieved_dict),
        self.prefix)

  def _ReadFromRedis(self, keys):
    # type: (Sequence[int]) -> Mapping[str, Any], Sequence[int]
    """Read the given keys from Redis, return {key: value}, missing keys.

    Args:
      keys: List of integer keys to read from Redis.

    Returns:
      A pair: hits, misses.  Where hits is {key: value} and misses is
        a list of any keys that were not found anywhere.
    """
    cache_hits = {}
    missing_keys = []
    try:
      values_list = self.redis_client.mget(
          [redis_utils.FormatRedisKey(key, prefix=self.prefix) for key in keys])
    except redis.RedisError as identifier:
      logging.error(
          'Redis error occurred during read operation: %s', identifier)
      values_list = [None] * len(keys)

    for key, serialized_value in zip(keys, values_list):
      if serialized_value:
        value = self._StrToValue(serialized_value)
        cache_hits[key] = value
        self.cache.CacheItem(key, value)
      else:
        missing_keys.append(key)
    logging.info(
        'decoded %d values from redis %s, missing %d', len(cache_hits),
        self.prefix, len(missing_keys))
    return cache_hits, missing_keys

  def _DeleteFromRedis(self, keys):
    # type: (Sequence[int]) -> None
    """Delete key-values from redis.

    Args:
      keys: List of integer keys to delete.
    """
    try:
      self.redis_client.delete(
          *[
              redis_utils.FormatRedisKey(key, prefix=self.prefix)
              for key in keys
          ])
    except redis.RedisError as identifier:
      logging.error(
          'Redis error occurred during delete operation %s', identifier)

  def _KeyToStr(self, key):
    # type: (int) -> str
    """Convert our int IDs to strings for use as memcache keys."""
    return str(key)

  def _StrToKey(self, key_str):
    # type: (str) -> int
    """Convert memcache keys back to the ints that we use as IDs."""
    return int(key_str)

  def _ValueToStr(self, value):
    # type: (Any) -> str
    """Serialize an application object so that it can be stored in L2 cache."""
    if self.use_redis:
      return redis_utils.SerializeValue(value, pb_class=self.pb_class)
    else:
      if not self.pb_class:
        return value
      elif self.pb_class == int:
        return str(value)
      else:
        return protobuf.encode_message(value)

  def _StrToValue(self, serialized_value):
    # type: (str) -> Any
    """Deserialize L2 cache string into an application object."""
    if self.use_redis:
      return redis_utils.DeserializeValue(
          serialized_value, pb_class=self.pb_class)
    else:
      if not self.pb_class:
        return serialized_value
      elif self.pb_class == int:
        return int(serialized_value)
      else:
        return protobuf.decode_message(self.pb_class, serialized_value)

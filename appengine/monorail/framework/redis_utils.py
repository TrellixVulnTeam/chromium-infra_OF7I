# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""A utility module for interfacing with Redis conveniently. """
import json
import logging

import redis

import settings
from protorpc import protobuf

connection_pool = None

def CreateRedisClient():
  # type: () -> redis.Redis
  """Creates a Redis object which implements Redis protocol and connection.

  Returns:
    redis.Redis object initialized with a connection pool.
    None on failure.
  """
  global connection_pool
  if not connection_pool:
    connection_pool = redis.BlockingConnectionPool(
        host=settings.redis_host,
        port=settings.redis_port,
        max_connections=1,
        timeout=2)
  return redis.Redis(connection_pool=connection_pool)


def FormatRedisKey(key, prefix=None):
  # type: (int, str) -> str
  """Converts key to string and prepends the prefix.

  Args:
    key: Integer key.
    prefix: String to prepend to the key.

  Returns:
    Formatted key with the format: "namespace:prefix:key".
  """
  formatted_key = ''
  if prefix:
    if prefix[-1] != ':':
      prefix += ':'
    formatted_key += prefix
  return formatted_key + str(key)


def VerifyRedisConnection(redis_client, msg=None):
  # type: (redis.Redis or FakeRedis, str) -> Bool
  """Checks the connection to Redis to ensure a connection can be established.

  Args:
    redis_client: client to connect and ping redis server. This can be a redis
      or fakeRedis object.
    msg: string for used logging information.

  Returns:
    True when connection to server is valid.
    False when an error occurs or redis_client is None.
  """
  if not redis_client:
    logging.info('Redis client is set to None on connect in %s', msg)
    return False
  try:
    redis_client.ping()
    logging.info('Redis client successfully connected to Redis in %s', msg)
    return True
  except redis.RedisError as identifier:
    logging.error(
        'Redis error occurred while connecting to server in %s: %s', msg,
        identifier)
    return False


def SerializeValue(value, pb_class=None):
  #type: (Any) -> strs
  """Serialize object as for storage in Redis. """
  if pb_class and pb_class is not int:
    return protobuf.encode_message(value)
  else:
    return json.dumps(value)


def DeserializeValue(value, pb_class=None):
  #type: (str) -> Any
  """Deserialize a string to create a python object. """
  if pb_class and pb_class is not int:
    return protobuf.decode_message(pb_class, value)
  else:
    return json.loads(value)

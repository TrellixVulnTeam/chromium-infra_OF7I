# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""A utility module for interfacing with Redis conveniently. """
import json
import logging
import threading

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
        # When Redis is not available, calls hang indefinitely without these.
        socket_connect_timeout=2,
        socket_timeout=2,
    )
  return redis.Redis(connection_pool=connection_pool)


def AsyncVerifyRedisConnection():
  # type: () -> None
  """Verifies the redis connection in a separate thread.

  Note that although an exception in the thread won't kill the main thread,
  it is not risk free.

  AppEngine joins with any running threads before finishing the request.
  If this thread were to hang indefinitely, then it would cause the request
  to hit DeadlineExceeded, thus still causing a user facing failure.

  We mitigate this risk by setting socket timeouts on our connection pool.

  # TODO(crbug/monorail/8221): Remove this code during this milestone.
  """

  def _AsyncVerifyRedisConnection():
    logging.info('AsyncVerifyRedisConnection thread started.')
    redis_client = CreateRedisClient()
    VerifyRedisConnection(redis_client)

  logging.info('Starting thread for AsyncVerifyRedisConnection.')
  threading.Thread(target=_AsyncVerifyRedisConnection).start()


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
  # type: (redis.Redis, Optional[str]) -> bool
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
    # TODO(crbug/monorail/8224): We can downgrade this to warning once we are
    # done with the switchover from memcache. Before that, log it to ensure we
    # see it.
    logging.exception(
        'Redis error occurred while connecting to server in %s: %s', msg,
        identifier)
    return False


def SerializeValue(value, pb_class=None):
  # type: (Any, Optional[type|classobj]) -> str
  """Serialize object as for storage in Redis. """
  if pb_class and pb_class is not int:
    return protobuf.encode_message(value)
  else:
    return json.dumps(value)


def DeserializeValue(value, pb_class=None):
  # type: (str, Optional[type|classobj]) -> Any
  """Deserialize a string to create a python object. """
  if pb_class and pb_class is not int:
    return protobuf.decode_message(pb_class, value)
  else:
    return json.loads(value)

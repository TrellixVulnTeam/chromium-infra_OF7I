# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Tests for the Redis utility module."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import fakeredis
import unittest

from framework import redis_utils
from proto import features_pb2


class RedisHelperTest(unittest.TestCase):

  def testFormatRedisKey(self):
    redis_key = redis_utils.FormatRedisKey(111)
    self.assertEqual('111', redis_key)
    redis_key = redis_utils.FormatRedisKey(222, prefix='foo:')
    self.assertEqual('foo:222', redis_key)
    redis_key = redis_utils.FormatRedisKey(333, prefix='bar')
    self.assertEqual('bar:333', redis_key)

  def testCreateRedisClient(self):
    self.assertIsNone(redis_utils.connection_pool)
    redis_client_1 = redis_utils.CreateRedisClient()
    self.assertIsNotNone(redis_client_1)
    self.assertIsNotNone(redis_utils.connection_pool)
    redis_client_2 = redis_utils.CreateRedisClient()
    self.assertIsNotNone(redis_client_2)
    self.assertIsNot(redis_client_1, redis_client_2)

  def testConnectionVerification(self):
    server = fakeredis.FakeServer()
    client = None
    self.assertFalse(redis_utils.VerifyRedisConnection(client))
    server.connected = True
    client = fakeredis.FakeRedis(server=server)
    self.assertTrue(redis_utils.VerifyRedisConnection(client))
    server.connected = False
    self.assertFalse(redis_utils.VerifyRedisConnection(client))

  def testSerializeDeserializeInt(self):
    serialized_int = redis_utils.SerializeValue(123)
    self.assertEqual('123', serialized_int)
    self.assertEquals(123, redis_utils.DeserializeValue(serialized_int))

  def testSerializeDeserializeStr(self):
    serialized = redis_utils.SerializeValue('123')
    self.assertEqual('"123"', serialized)
    self.assertEquals('123', redis_utils.DeserializeValue(serialized))

  def testSerializeDeserializePB(self):
    features = features_pb2.Hotlist.HotlistItem(
        issue_id=7949, rank=0, adder_id=333, date_added=1525)
    serialized = redis_utils.SerializeValue(
        features, pb_class=features_pb2.Hotlist.HotlistItem)
    self.assertIsInstance(serialized, str)
    deserialized = redis_utils.DeserializeValue(
        serialized, pb_class=features_pb2.Hotlist.HotlistItem)
    self.assertIsInstance(deserialized, features_pb2.Hotlist.HotlistItem)
    self.assertEquals(deserialized, features)

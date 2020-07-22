# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for the cache classes."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import fakeredis
import unittest

from google.appengine.api import memcache
from google.appengine.ext import testbed

import settings
from services import caches
from testing import fake


class RamCacheTest(unittest.TestCase):

  def setUp(self):
    self.cnxn = 'fake connection'
    self.cache_manager = fake.CacheManager()
    self.ram_cache = caches.RamCache(self.cache_manager, 'issue', max_size=3)

  def testInit(self):
    self.assertEqual('issue', self.ram_cache.kind)
    self.assertEqual(3, self.ram_cache.max_size)
    self.assertEqual(
        [self.ram_cache],
        self.cache_manager.cache_registry['issue'])

  def testCacheItem(self):
    self.ram_cache.CacheItem(123, 'foo')
    self.assertEqual('foo', self.ram_cache.cache[123])

  def testCacheItem_DropsOldItems(self):
    self.ram_cache.CacheItem(123, 'foo')
    self.ram_cache.CacheItem(234, 'foo')
    self.ram_cache.CacheItem(345, 'foo')
    self.ram_cache.CacheItem(456, 'foo')
    # The cache does not get bigger than its limit.
    self.assertEqual(3, len(self.ram_cache.cache))
    # An old value is dropped, not the newly added one.
    self.assertIn(456, self.ram_cache.cache)

  def testCacheAll(self):
    self.ram_cache.CacheAll({123: 'foo'})
    self.assertEqual('foo', self.ram_cache.cache[123])

  def testCacheAll_DropsOldItems(self):
    self.ram_cache.CacheAll({1: 'a', 2: 'b', 3: 'c'})
    self.ram_cache.CacheAll({4: 'x', 5: 'y'})
    # The cache does not get bigger than its limit.
    self.assertEqual(3, len(self.ram_cache.cache))
    # An old value is dropped, not the newly added one.
    self.assertIn(4, self.ram_cache.cache)
    self.assertIn(5, self.ram_cache.cache)
    self.assertEqual('y', self.ram_cache.cache[5])

  def testHasItem(self):
    self.ram_cache.CacheItem(123, 'foo')
    self.assertTrue(self.ram_cache.HasItem(123))
    self.assertFalse(self.ram_cache.HasItem(999))

  def testGetItem(self):
    self.ram_cache.CacheItem(123, 'foo')
    self.assertEqual('foo', self.ram_cache.GetItem(123))
    self.assertEqual(None, self.ram_cache.GetItem(456))

  def testGetAll(self):
    self.ram_cache.CacheItem(123, 'foo')
    self.ram_cache.CacheItem(124, 'bar')
    hits, misses = self.ram_cache.GetAll([123, 124, 999])
    self.assertEqual({123: 'foo', 124: 'bar'}, hits)
    self.assertEqual([999], misses)

  def testLocalInvalidate(self):
    self.ram_cache.CacheAll({123: 'a', 124: 'b', 125: 'c'})
    self.ram_cache.LocalInvalidate(124)
    self.assertEqual(2, len(self.ram_cache.cache))
    self.assertNotIn(124, self.ram_cache.cache)

    self.ram_cache.LocalInvalidate(999)
    self.assertEqual(2, len(self.ram_cache.cache))

  def testInvalidate(self):
    self.ram_cache.CacheAll({123: 'a', 124: 'b', 125: 'c'})
    self.ram_cache.Invalidate(self.cnxn, 124)
    self.assertEqual(2, len(self.ram_cache.cache))
    self.assertNotIn(124, self.ram_cache.cache)
    self.assertEqual(self.cache_manager.last_call,
                     ('StoreInvalidateRows', self.cnxn, 'issue', [124]))

  def testInvalidateKeys(self):
    self.ram_cache.CacheAll({123: 'a', 124: 'b', 125: 'c'})
    self.ram_cache.InvalidateKeys(self.cnxn, [124])
    self.assertEqual(2, len(self.ram_cache.cache))
    self.assertNotIn(124, self.ram_cache.cache)
    self.assertEqual(self.cache_manager.last_call,
                     ('StoreInvalidateRows', self.cnxn, 'issue', [124]))

  def testLocalInvalidateAll(self):
    self.ram_cache.CacheAll({123: 'a', 124: 'b', 125: 'c'})
    self.ram_cache.LocalInvalidateAll()
    self.assertEqual(0, len(self.ram_cache.cache))

  def testInvalidateAll(self):
    self.ram_cache.CacheAll({123: 'a', 124: 'b', 125: 'c'})
    self.ram_cache.InvalidateAll(self.cnxn)
    self.assertEqual(0, len(self.ram_cache.cache))
    self.assertEqual(self.cache_manager.last_call,
                     ('StoreInvalidateAll', self.cnxn, 'issue'))


class ShardedRamCacheTest(unittest.TestCase):

  def setUp(self):
    self.cnxn = 'fake connection'
    self.cache_manager = fake.CacheManager()
    self.sharded_ram_cache = caches.ShardedRamCache(
        self.cache_manager, 'issue', max_size=3, num_shards=3)

  def testLocalInvalidate(self):
    self.sharded_ram_cache.CacheAll({
        (123, 0): 'a',
        (123, 1): 'aa',
        (123, 2): 'aaa',
        (124, 0): 'b',
        (124, 1): 'bb',
        (124, 2): 'bbb',
        })
    self.sharded_ram_cache.LocalInvalidate(124)
    self.assertEqual(3, len(self.sharded_ram_cache.cache))
    self.assertNotIn((124, 0), self.sharded_ram_cache.cache)
    self.assertNotIn((124, 1), self.sharded_ram_cache.cache)
    self.assertNotIn((124, 2), self.sharded_ram_cache.cache)

    self.sharded_ram_cache.LocalInvalidate(999)
    self.assertEqual(3, len(self.sharded_ram_cache.cache))


class TestableTwoLevelCache(caches.AbstractTwoLevelCache):

  def __init__(
      self,
      cache_manager,
      kind,
      max_size=None,
      use_redis=False,
      redis_client=None):
    super(TestableTwoLevelCache, self).__init__(
        cache_manager,
        kind,
        'testable:',
        None,
        max_size=max_size,
        use_redis=use_redis,
        redis_client=redis_client)

  # pylint: disable=unused-argument
  def FetchItems(self, cnxn, keys, **kwargs):
    """On RAM and memcache miss, hit the database."""
    return {key: key for key in keys if key < 900}


class AbstractTwoLevelCacheTest_Memcache(unittest.TestCase):

  def setUp(self):
    self.testbed = testbed.Testbed()
    self.testbed.activate()
    self.testbed.init_memcache_stub()

    self.cnxn = 'fake connection'
    self.cache_manager = fake.CacheManager()
    self.testable_2lc = TestableTwoLevelCache(self.cache_manager, 'issue')

  def tearDown(self):
    self.testbed.deactivate()

  def testCacheItem(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.assertEqual(12300, self.testable_2lc.cache.cache[123])

  def testHasItem(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.assertTrue(self.testable_2lc.HasItem(123))
    self.assertFalse(self.testable_2lc.HasItem(444))
    self.assertFalse(self.testable_2lc.HasItem(999))

  def testWriteToMemcache_Normal(self):
    retrieved_dict = {123: 12300, 124: 12400}
    self.testable_2lc._WriteToMemcache(retrieved_dict)
    actual_123, _ = self.testable_2lc._ReadFromMemcache([123])
    self.assertEqual(12300, actual_123[123])
    actual_124, _ = self.testable_2lc._ReadFromMemcache([124])
    self.assertEqual(12400, actual_124[124])

  def testWriteToMemcache_String(self):
    retrieved_dict = {123: 'foo', 124: 'bar'}
    self.testable_2lc._WriteToMemcache(retrieved_dict)
    actual_123, _ = self.testable_2lc._ReadFromMemcache([123])
    self.assertEqual('foo', actual_123[123])
    actual_124, _ = self.testable_2lc._ReadFromMemcache([124])
    self.assertEqual('bar', actual_124[124])

  def testWriteToMemcache_ProtobufInt(self):
    self.testable_2lc.pb_class = int
    retrieved_dict = {123: 12300, 124: 12400}
    self.testable_2lc._WriteToMemcache(retrieved_dict)
    actual_123, _ = self.testable_2lc._ReadFromMemcache([123])
    self.assertEqual(12300, actual_123[123])
    actual_124, _ = self.testable_2lc._ReadFromMemcache([124])
    self.assertEqual(12400, actual_124[124])

  def testWriteToMemcache_List(self):
    retrieved_dict = {123: [1, 2, 3], 124: [1, 2, 4]}
    self.testable_2lc._WriteToMemcache(retrieved_dict)
    actual_123, _ = self.testable_2lc._ReadFromMemcache([123])
    self.assertEqual([1, 2, 3], actual_123[123])
    actual_124, _ = self.testable_2lc._ReadFromMemcache([124])
    self.assertEqual([1, 2, 4], actual_124[124])

  def testWriteToMemcache_Dict(self):
    retrieved_dict = {123: {'ham': 2, 'spam': 3}, 124: {'eggs': 2, 'bean': 4}}
    self.testable_2lc._WriteToMemcache(retrieved_dict)
    actual_123, _ = self.testable_2lc._ReadFromMemcache([123])
    self.assertEqual({'ham': 2, 'spam': 3}, actual_123[123])
    actual_124, _ = self.testable_2lc._ReadFromMemcache([124])
    self.assertEqual({'eggs': 2, 'bean': 4}, actual_124[124])

  def testWriteToMemcache_HugeValue(self):
    """If memcache refuses to store a huge value, we don't store any."""
    self.testable_2lc._WriteToMemcache({124: 124999})  # Gets deleted.
    huge_str = 'huge' * 260000
    retrieved_dict = {123: huge_str, 124: 12400}
    self.testable_2lc._WriteToMemcache(retrieved_dict)
    actual_123 = memcache.get('testable:123')
    self.assertEqual(None, actual_123)
    actual_124 = memcache.get('testable:124')
    self.assertEqual(None, actual_124)

  def testGetAll_FetchGetsIt(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    # Clear the RAM cache so that we find items in memcache.
    self.testable_2lc.cache.LocalInvalidateAll()
    self.testable_2lc.CacheItem(125, 12500)
    hits, misses = self.testable_2lc.GetAll(self.cnxn, [123, 124, 333, 444])
    self.assertEqual({123: 12300, 124: 12400, 333: 333, 444: 444}, hits)
    self.assertEqual([], misses)
    # The RAM cache now has items found in memcache and DB.
    self.assertItemsEqual(
        [123, 124, 125, 333, 444], list(self.testable_2lc.cache.cache.keys()))

  def testGetAll_FetchGetsItFromDB(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    hits, misses = self.testable_2lc.GetAll(self.cnxn, [123, 124, 333, 444])
    self.assertEqual({123: 12300, 124: 12400, 333: 333, 444: 444}, hits)
    self.assertEqual([], misses)

  def testGetAll_FetchDoesNotFindIt(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    hits, misses = self.testable_2lc.GetAll(self.cnxn, [123, 124, 999])
    self.assertEqual({123: 12300, 124: 12400}, hits)
    self.assertEqual([999], misses)

  def testInvalidateKeys(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    self.testable_2lc.CacheItem(125, 12500)
    self.testable_2lc.InvalidateKeys(self.cnxn, [124])
    self.assertEqual(2, len(self.testable_2lc.cache.cache))
    self.assertNotIn(124, self.testable_2lc.cache.cache)
    self.assertEqual(
        self.cache_manager.last_call,
        ('StoreInvalidateRows', self.cnxn, 'issue', [124]))

  def testGetAllAlreadyInRam(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    hits, misses = self.testable_2lc.GetAllAlreadyInRam(
        [123, 124, 333, 444, 999])
    self.assertEqual({123: 12300, 124: 12400}, hits)
    self.assertEqual([333, 444, 999], misses)

  def testInvalidateAllRamEntries(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    self.testable_2lc.InvalidateAllRamEntries(self.cnxn)
    self.assertFalse(self.testable_2lc.HasItem(123))
    self.assertFalse(self.testable_2lc.HasItem(124))


class AbstractTwoLevelCacheTest_Redis(unittest.TestCase):

  def setUp(self):
    self.cnxn = 'fake connection'
    self.cache_manager = fake.CacheManager()

    self.server = fakeredis.FakeServer()
    self.fake_redis_client = fakeredis.FakeRedis(server=self.server)
    self.testable_2lc = TestableTwoLevelCache(
        self.cache_manager,
        'issue',
        use_redis=True,
        redis_client=self.fake_redis_client)

  def tearDown(self):
    self.fake_redis_client.flushall()

  def testCacheItem(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.assertEqual(12300, self.testable_2lc.cache.cache[123])

  def testHasItem(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.assertTrue(self.testable_2lc.HasItem(123))
    self.assertFalse(self.testable_2lc.HasItem(444))
    self.assertFalse(self.testable_2lc.HasItem(999))

  def testWriteToRedis_Normal(self):
    retrieved_dict = {123: 12300, 124: 12400}
    self.testable_2lc._WriteToRedis(retrieved_dict)
    actual_123, _ = self.testable_2lc._ReadFromRedis([123])
    self.assertEqual(12300, actual_123[123])
    actual_124, _ = self.testable_2lc._ReadFromRedis([124])
    self.assertEqual(12400, actual_124[124])

  def testWriteToRedis_str(self):
    retrieved_dict = {111: 'foo', 222: 'bar'}
    self.testable_2lc._WriteToRedis(retrieved_dict)
    actual_111, _ = self.testable_2lc._ReadFromRedis([111])
    self.assertEqual('foo', actual_111[111])
    actual_222, _ = self.testable_2lc._ReadFromRedis([222])
    self.assertEqual('bar', actual_222[222])

  def testWriteToRedis_ProtobufInt(self):
    self.testable_2lc.pb_class = int
    retrieved_dict = {123: 12300, 124: 12400}
    self.testable_2lc._WriteToRedis(retrieved_dict)
    actual_123, _ = self.testable_2lc._ReadFromRedis([123])
    self.assertEqual(12300, actual_123[123])
    actual_124, _ = self.testable_2lc._ReadFromRedis([124])
    self.assertEqual(12400, actual_124[124])

  def testWriteToRedis_List(self):
    retrieved_dict = {123: [1, 2, 3], 124: [1, 2, 4]}
    self.testable_2lc._WriteToRedis(retrieved_dict)
    actual_123, _ = self.testable_2lc._ReadFromRedis([123])
    self.assertEqual([1, 2, 3], actual_123[123])
    actual_124, _ = self.testable_2lc._ReadFromRedis([124])
    self.assertEqual([1, 2, 4], actual_124[124])

  def testWriteToRedis_Dict(self):
    retrieved_dict = {123: {'ham': 2, 'spam': 3}, 124: {'eggs': 2, 'bean': 4}}
    self.testable_2lc._WriteToRedis(retrieved_dict)
    actual_123, _ = self.testable_2lc._ReadFromRedis([123])
    self.assertEqual({'ham': 2, 'spam': 3}, actual_123[123])
    actual_124, _ = self.testable_2lc._ReadFromRedis([124])
    self.assertEqual({'eggs': 2, 'bean': 4}, actual_124[124])

  def testGetAll_FetchGetsIt(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    # Clear the RAM cache so that we find items in redis.
    self.testable_2lc.cache.LocalInvalidateAll()
    self.testable_2lc.CacheItem(125, 12500)
    hits, misses = self.testable_2lc.GetAll(self.cnxn, [123, 124, 333, 444])
    self.assertEqual({123: 12300, 124: 12400, 333: 333, 444: 444}, hits)
    self.assertEqual([], misses)
    # The RAM cache now has items found in redis and DB.
    self.assertItemsEqual(
        [123, 124, 125, 333, 444], list(self.testable_2lc.cache.cache.keys()))

  def testGetAll_FetchGetsItFromDB(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    hits, misses = self.testable_2lc.GetAll(self.cnxn, [123, 124, 333, 444])
    self.assertEqual({123: 12300, 124: 12400, 333: 333, 444: 444}, hits)
    self.assertEqual([], misses)

  def testGetAll_FetchDoesNotFindIt(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    hits, misses = self.testable_2lc.GetAll(self.cnxn, [123, 124, 999])
    self.assertEqual({123: 12300, 124: 12400}, hits)
    self.assertEqual([999], misses)

  def testInvalidateKeys(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    self.testable_2lc.CacheItem(125, 12500)
    self.testable_2lc.InvalidateKeys(self.cnxn, [124])
    self.assertEqual(2, len(self.testable_2lc.cache.cache))
    self.assertNotIn(124, self.testable_2lc.cache.cache)
    self.assertEqual(self.cache_manager.last_call,
                     ('StoreInvalidateRows', self.cnxn, 'issue', [124]))

  def testGetAllAlreadyInRam(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    hits, misses = self.testable_2lc.GetAllAlreadyInRam(
        [123, 124, 333, 444, 999])
    self.assertEqual({123: 12300, 124: 12400}, hits)
    self.assertEqual([333, 444, 999], misses)

  def testInvalidateAllRamEntries(self):
    self.testable_2lc.CacheItem(123, 12300)
    self.testable_2lc.CacheItem(124, 12400)
    self.testable_2lc.InvalidateAllRamEntries(self.cnxn)
    self.assertFalse(self.testable_2lc.HasItem(123))
    self.assertFalse(self.testable_2lc.HasItem(124))

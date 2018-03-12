# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock
import random

from common.findit_http_client import FinditHttpClient
from common.waterfall import buildbucket_client
from model.build_ahead_try_job import BuildAheadTryJob
from model.wf_try_bot_cache import WfTryBotCache
from services import build_ahead
from services import git
from services import swarmbot_util
from waterfall.test import wf_testcase
from waterfall import waterfall_config

CN = 'builder_cc0b584fcab5ab502af9c154891c705115ea1fefd4d176cabf5d04ae0cd4e18c'


class BuildAheadTest(wf_testcase.WaterfallTestCase):

  @mock.patch.object(buildbucket_client, 'TriggerTryJobs')
  def testBuildAhead(self, mock_trigger):
    _ = build_ahead.TriggerBuildAhead('master2', 'builder5', 'some_bot')
    mock_trigger.assert_called_once_with([
        buildbucket_client.TryJob(
            master_name='luci.chromium.findit',
            builder_name='findit_variable',
            properties={
                'recipe': 'findit/chromium/compile',
                'good_revision': 'HEAD~1',
                'target_mastername': 'master2',
                'mastername': 'tryserver2',
                'suspected_revisions': [],
                'target_buildername': 'builder5',
                'bad_revision': 'HEAD'
            },
            tags=[],
            additional_build_parameters=None,
            cache_name=CN,
            dimensions=[
                'os:Mac-10.9', 'cpu:x86-64', 'pool:luci.chromium.findit',
                'id:some_bot'
            ],
            pubsub_callback=None)
    ])
    mock_trigger.reset_mock()

    _ = build_ahead.TriggerBuildAhead('master2', 'builder5', None)
    mock_trigger.assert_called_once_with([
        buildbucket_client.TryJob(
            master_name='luci.chromium.findit',
            builder_name='findit_variable',
            properties={
                'recipe': 'findit/chromium/compile',
                'good_revision': 'HEAD~1',
                'target_mastername': 'master2',
                'mastername': 'tryserver2',
                'suspected_revisions': [],
                'target_buildername': 'builder5',
                'bad_revision': 'HEAD'
            },
            tags=[],
            additional_build_parameters=None,
            cache_name=CN,
            dimensions=[
                'os:Mac-10.9', 'cpu:x86-64', 'pool:luci.chromium.findit'
            ],
            pubsub_callback=None)
    ])

  @mock.patch.object(FinditHttpClient, 'Get')
  def testTreeIsOpen(self, mock_get):
    responses = [
        (500, None),
        (400, None),
        (200, None),
        (200, '[]'),
        (200, '[{}]'),
        (200, '[{"general_state":"closed"}]'),
        (200, '[{"general_state":"open"}]'),
    ]
    mock_get.side_effect = responses

    for _ in range(len(responses) - 1):
      self.assertFalse(build_ahead._TreeIsOpen())

    self.assertTrue(build_ahead._TreeIsOpen())

  @mock.patch.object(buildbucket_client, 'GetTryJobs')
  def testUpdateRunningJobs(self, mock_get_tryjobs):
    build_ahead._UpdateRunningBuilds()
    self.assertFalse(mock_get_tryjobs.called)
    BuildAheadTryJob.Create('80000001', 'unix', 'cache_1').put()
    BuildAheadTryJob.Create('80000002', 'win', 'cache_2').put()
    BuildAheadTryJob.Create('80000003', 'mac', 'cache_3').put()
    mock_get_tryjobs.return_value = [
        (None,
         buildbucket_client.BuildbucketBuild({
             'id': '80000001',
             'status': 'STARTED'
         })),
        (None,
         buildbucket_client.BuildbucketBuild({
             'id': '80000002',
             'status': 'STARTED'
         })),
        (None,
         buildbucket_client.BuildbucketBuild({
             'id': '80000003',
             'status': 'STARTED'
         })),
    ]
    self.assertEqual(3, len(build_ahead._UpdateRunningBuilds()))

    mock_get_tryjobs.return_value = [
        (None,
         buildbucket_client.BuildbucketBuild({
             'id': '80000001',
             'status': 'COMPLETED'
         })),
        (None,
         buildbucket_client.BuildbucketBuild({
             'id': '80000002',
             'status': 'STARTED'
         })),
        (None,
         buildbucket_client.BuildbucketBuild({
             'id': '80000003',
             'status': 'STARTED'
         })),
    ]
    self.assertEqual(2, len(build_ahead._UpdateRunningBuilds()))

    mock_get_tryjobs.return_value = [
        (None,
         buildbucket_client.BuildbucketBuild({
             'id': '80000002',
             'status': 'COMPLETED'
         })),
        (None,
         buildbucket_client.BuildbucketBuild({
             'id': '80000003',
             'status': 'COMPLETED'
         })),
    ]
    self.assertEqual(0, len(build_ahead._UpdateRunningBuilds()))

    BuildAheadTryJob.Create('80000004', 'mac', 'cache_4').put()
    mock_get_tryjobs.return_value = [(buildbucket_client.BuildbucketError({
        'reason': 'BUILD_NOT_FOUND',
        'message': 'BUILD_NOT_FOUND'
    }), None)]

    self.assertEqual(0, len(build_ahead._UpdateRunningBuilds()))
    self.assertTrue(BuildAheadTryJob.Get('80000004').running)

  @mock.patch.object(git, 'CountRecentCommits')
  def testLowRepoActivity(self, mock_count_commits):
    mock_count_commits.side_effect = [i for i in range(10)]
    for i in range(4):
      self.assertTrue(build_ahead._LowRepoActivity())
    for i in range(6):
      self.assertFalse(build_ahead._LowRepoActivity())

  def testPlatformToDimensions(self):
    self.assertEqual(['os:Mac'], build_ahead._PlatformToDimensions('mac'))
    self.assertEqual(['os:Windows'], build_ahead._PlatformToDimensions('win'))
    self.assertEqual(['os:Linux'], build_ahead._PlatformToDimensions('unix'))
    self.assertEqual(['os:Linux'], build_ahead._PlatformToDimensions('android'))

  @mock.patch.object(swarmbot_util, 'OnlyAvailable')
  @mock.patch.object(swarmbot_util, 'GetBotsByDimension')
  def testAvailableBotsByPlatform(self, mock_get_bots, mock_available):
    _ = build_ahead._AvailableBotsByPlatform('mac')
    self.assertIn('pool:luci.chromium.findit', mock_get_bots.call_args[0][0])
    self.assertIn('os:Mac', mock_get_bots.call_args[0][0])
    mock_available.assert_called_once()

  @mock.patch.object(build_ahead, '_AvailableBotsByPlatform')
  @mock.patch.object(BuildAheadTryJob, 'RunningJobs')
  @mock.patch.object(build_ahead, '_LowRepoActivity')
  def testPlatformsToBuildHighActivity(self, mock_lo_activity, mock_jobs,
                                       mock_bots):
    mock_lo_activity.return_value = False
    mock_jobs.return_value = []
    self.assertEqual(4, len(build_ahead._PlatformsToBuild()))

    mock_jobs.side_effect = lambda platform: [
        BuildAheadTryJob.Create('1234', platform, 'cache_x')]
    self.assertEqual(0, len(build_ahead._PlatformsToBuild()))

    mock_jobs.side_effect = [
        [BuildAheadTryJob.Create('1234', 'android', 'cache_x')],
        [],
        [],
        [],
    ]
    self.assertEqual(3, len(build_ahead._PlatformsToBuild()))

    mock_jobs.side_effect = [
        [],
        [BuildAheadTryJob.Create('1235', 'mac', 'cache_y')],
        [BuildAheadTryJob.Create('1236', 'unix', 'cache_z')],
        [BuildAheadTryJob.Create('1237', 'win', 'cache_a')],
    ]
    self.assertEqual(1, len(build_ahead._PlatformsToBuild()))
    self.assertFalse(mock_bots.called)

  @mock.patch.object(build_ahead, '_AvailableBotsByPlatform')
  @mock.patch.object(BuildAheadTryJob, 'RunningJobs')
  @mock.patch.object(build_ahead, '_LowRepoActivity')
  def testPlatformsToBuildLowActivity(self, mock_lo_activity, mock_jobs,
                                      mock_bots):
    mock_lo_activity.return_value = True
    mock_jobs.side_effect = [
        [],
        [BuildAheadTryJob.Create('1235', 'mac', 'cache_y')],
        [BuildAheadTryJob.Create('1236', 'unix', 'cache_z')],
        [BuildAheadTryJob.Create('1237', 'win', 'cache_a')],
    ]
    mock_bots.side_effect = [
        [{
            'id': 'bot1'
        }],
        [],
        [{
            'id': 'bot2'
        }],
        [{
            'id': 'bot3'
        }, {
            'id': 'bot4'
        }],
    ]
    self.assertEqual(['android', 'win'], build_ahead._PlatformsToBuild())

  @mock.patch.object(waterfall_config, 'GetSupportedCompileBuilders')
  @mock.patch.object(random, 'uniform')
  def testPickRandomBuilder(self, mock_random, mock_builders):
    linux_bot = {'master': 'dummy', 'builder': 'Linux Builder'}
    chrome_bot = {'master': 'dummy', 'builder': 'Chrome Builder'}
    android_bot = {'master': 'dummy', 'builder': 'Android Builder'}
    mock_builders.return_value = [linux_bot, chrome_bot, android_bot]

    linux_cache = WfTryBotCache.Get(
        swarmbot_util.GetCacheName(linux_bot['master'], linux_bot['builder']))
    chrome_cache = WfTryBotCache.Get(
        swarmbot_util.GetCacheName(chrome_bot['master'], chrome_bot['builder']))
    android_cache = WfTryBotCache.Get(
        swarmbot_util.GetCacheName(android_bot['master'],
                                   android_bot['builder']))

    linux_cache.AddFullBuild('bot1', 100, None)
    linux_cache.AddFullBuild('bot2', 110, None)
    linux_cache.AddFullBuild('bot3', 120, None)

    chrome_cache.AddFullBuild('bot1', 110, None)
    chrome_cache.AddFullBuild('bot2', 120, None)
    chrome_cache.AddFullBuild('bot3', 130, None)

    android_cache.AddFullBuild('bot1', 140, None)

    linux_cache.put()
    chrome_cache.put()
    android_cache.put()

    # Expected weights are 20 for linux, 10 for chrome, 0 for android.
    mock_random.side_effect = [i + 0.1 for i in range(30)]
    for i in range(20):
      self.assertEqual(
          linux_cache,
          build_ahead._PickRandomBuilder(
              build_ahead._GetSupportedCompileCaches('unix'))['cache_stats'])
    for i in range(10):
      self.assertEqual(
          chrome_cache,
          build_ahead._PickRandomBuilder(
              build_ahead._GetSupportedCompileCaches('unix'))['cache_stats'])

  @mock.patch.object(build_ahead, 'TriggerBuildAhead')
  def testTriggerAndSave(self, mock_trigger):
    mock_trigger.return_value = Exception("Dummy exception"), None
    with self.assertRaisesRegexp(Exception, ".*Dummy.*"):
      build_ahead._TriggerAndSave('master', 'builder', 'cache_name', 'platform',
                                  'bot')

    build_id = '81234567890'
    mock_trigger.return_value = (None,
                                 buildbucket_client.BuildbucketBuild({
                                     'id': build_id
                                 }))
    ba = build_ahead._TriggerAndSave('master', 'builder', 'cache_name',
                                     'platform', 'bot')
    self.assertEqual(build_id, ba.BuildId)
    self.assertEqual('platform', ba.platform)
    self.assertEqual('cache_name', ba.cache_name)

    ba = build_ahead._TriggerAndSave('master', 'builder', 'cache_name',
                                     'platform', None)
    self.assertEqual(build_id, ba.BuildId)
    self.assertEqual('platform', ba.platform)
    self.assertEqual('cache_name', ba.cache_name)

  @mock.patch.object(git, 'GetCLInfo')
  def testOldEnough(self, mock_cl):
    bot_id = 'old_enough_bot'
    head_cp = 20000
    mock_cl.return_value = {
        'HEAD': {
            'revision': 'HEAD',
            'repo_name': 'chromium',
            'commit_position': 20000,
            'url': 'https://dummyurl.com/a.git/+/HEAD',
            'author': 'dummy@dummy.org',
        }
    }
    oldest_cp = head_cp - build_ahead.STALE_CACHE_AGE
    results = []
    for i in range(oldest_cp - 10, oldest_cp + 10):
      tbc = WfTryBotCache.Get('Test %d' % i)
      tbc.full_build_commit_positions[bot_id] = i
      results.append(build_ahead._OldEnough(tbc, bot_id))
    self.assertEqual(10 * [True] + 10 * [False], results)

  @mock.patch.object(build_ahead, '_GetSupportedCompileCaches')
  @mock.patch.object(build_ahead, '_OldEnough')
  @mock.patch.object(build_ahead, '_TriggerAndSave')
  @mock.patch.object(swarmbot_util, 'GetBotsByDimension')
  @mock.patch.object(build_ahead, '_PickRandomBuilder')
  def testStartBuildAhead(self, mock_pick, mock_bots, mock_trigger, mock_old,
                          _):
    build_ahead.PLATFORM_DIMENSION_MAP['dummy_platform'] = ['os:dummy']
    dummy_cache = WfTryBotCache.Get('Dummy')
    mock_pick.return_value = {
        'cache_stats': dummy_cache,
        'master': 'chromium',
        'builder': 'dummy',
        'cache_name': 'cache_dumy123123',
    }
    mock_bots.return_value = [
        {
            'bot_id': 'bot_1'
        },
        {
            'bot_id': 'bot_2'
        },
        {
            'bot_id': 'bot_3'
        },
    ]

    # > 2: select second newest.
    dummy_cache.full_build_commit_positions = {
        'bot_1': 1010,
        'bot_2': 1020,
        'bot_3': 1030,
    }
    _ = build_ahead._StartBuildAhead('dummy_platform')
    mock_trigger.assert_called_once_with(
        'chromium', 'dummy', 'cache_dumy123123', 'dummy_platform', 'bot_2')
    mock_trigger.reset_mock()

    # = 2: select second newest.
    dummy_cache.full_build_commit_positions = {
        'bot_1': 1010,
        'bot_2': 1020,
    }
    _ = build_ahead._StartBuildAhead('dummy_platform')
    mock_trigger.assert_called_once_with(
        'chromium', 'dummy', 'cache_dumy123123', 'dummy_platform', 'bot_1')
    mock_trigger.reset_mock()

    # No caches: let swarming decide by passing None.
    dummy_cache.full_build_commit_positions = {
        'bot_8': 1010,
        'bot_9': 1020,
    }
    _ = build_ahead._StartBuildAhead('dummy_platform')
    mock_trigger.assert_called_once_with(
        'chromium', 'dummy', 'cache_dumy123123', 'dummy_platform', None)
    mock_trigger.reset_mock()

    # Exactly 1 bot: rebuild if old.
    dummy_cache.full_build_commit_positions = {
        'bot_1': 10,
    }
    mock_old.return_value = True
    _ = build_ahead._StartBuildAhead('dummy_platform')
    mock_trigger.assert_called_once_with(
        'chromium', 'dummy', 'cache_dumy123123', 'dummy_platform', 'bot_1')
    mock_trigger.reset_mock()

    # Exactly 1 bot: do not rebuild if not old enough.
    dummy_cache.full_build_commit_positions = {
        'bot_1': 1010,
    }
    mock_old.return_value = False
    _ = build_ahead._StartBuildAhead('dummy_platform')
    self.assertFalse(mock_trigger.called)

  @mock.patch.object(build_ahead, '_UpdateRunningBuilds')
  @mock.patch.object(build_ahead, '_TreeIsOpen')
  @mock.patch.object(build_ahead, '_PlatformsToBuild')
  @mock.patch.object(build_ahead, '_StartBuildAhead')
  def testBuildCaches(self, mock_start, mock_platforms, mock_tree, _):
    mock_tree.return_value = True
    mock_platforms.side_effect = [
        ['unix', 'win', 'mac'],
        ['win', 'mac'],
        ['win'],
        [],
    ]
    build_ahead.BuildCaches()
    self.assertEqual(3, len(mock_start.call_args_list))
    mock_start.reset_mock()

    build_ahead.BuildCaches()
    self.assertEqual(2, len(mock_start.call_args_list))
    mock_start.reset_mock()

    build_ahead.BuildCaches()
    self.assertEqual(1, len(mock_start.call_args_list))
    mock_start.reset_mock()

    mock_tree.return_value = False
    build_ahead.BuildCaches()
    self.assertFalse(mock_start.called)

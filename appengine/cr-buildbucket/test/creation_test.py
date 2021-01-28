# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import contextlib
import datetime

from components import auth
from components import utils
from google.appengine.ext import ndb
from google.protobuf import duration_pb2
from google.protobuf import struct_pb2
from testing_utils import testing
import mock

from go.chromium.org.luci.buildbucket.proto import build_pb2
from go.chromium.org.luci.buildbucket.proto import builder_pb2
from go.chromium.org.luci.buildbucket.proto import common_pb2
from go.chromium.org.luci.buildbucket.proto import builds_service_pb2 as rpc_pb2
from go.chromium.org.luci.buildbucket.proto import service_config_pb2
from test import test_util
import bbutil
import config
import creation
import errors
import model
import search
import user

future = test_util.future


class CreationTest(testing.AppengineTestCase):
  test_build = None

  def setUp(self):
    super(CreationTest, self).setUp()

    self.current_identity = auth.Identity('service', 'unittest')
    self.patch(
        'components.auth.get_current_identity',
        side_effect=lambda: self.current_identity
    )
    self.now = datetime.datetime(2015, 1, 1)
    self.patch('components.utils.utcnow', side_effect=lambda: self.now)

    self.patch(
        'tokens.generate_build_token',
        side_effect=lambda build_id: 'token-for-%d' % build_id
    )

    perms = test_util.mock_permissions(self)
    perms['chromium/try'] = [user.PERM_BUILDS_ADD]

    self.chromium_try = test_util.parse_bucket_cfg(
        '''
        name: "luci.chromium.try"
        swarming {
          builders {
            name: "linux"
            build_numbers: YES
            swarming_host: "chromium-swarm.appspot.com"
            recipe {
              name: "recipe"
              cipd_package: "infra/recipe_bundle"
              cipd_version: "refs/heads/master"
            }
          }
          builders {
            name: "linux_legacy"
            build_numbers: YES
            swarming_host: "chromium-swarm.appspot.com"
            recipe {
              name: "recipe"
              cipd_package: "infra/recipe_bundle"
              cipd_version: "refs/heads/master"
            }
          }
          builders {
            name: "linux_wait"
            wait_for_capacity: YES
            swarming_host: "chromium-swarm.appspot.com"
            recipe {
              name: "recipe"
              cipd_package: "infra/recipe_bundle"
              cipd_version: "refs/heads/master"
            }
          }
          builders {
            name: "linux_modern"
            build_numbers: YES
            swarming_host: "chromium-swarm.appspot.com"
            exe {
              cipd_package: "infra/recipe_bundle"
              cipd_version: "refs/heads/master"
              cmd: "luciexe"
              cmd: "-custom"
              cmd: "-flags"
            }
            properties: "{\\"recipe\\":\\"something\\"}"
          }
          builders {
            name: "mac"
            swarming_host: "chromium-swarm.appspot.com"
            recipe {
              name: "recipe"
              cipd_package: "infra/recipe_bundle"
              cipd_version: "refs/heads/master"
            }
          }
          builders {
            name: "mac_exp"
            swarming_host: "chromium-swarm.appspot.com"
            recipe {
              name: "recipe"
              cipd_package: "infra/recipe_bundle"
              cipd_version: "refs/heads/master"
            }
            experiments {
              key: "chromium.exp_foo"
              value: 10
            }
          }
          builders {
            name: "win"
            swarming_host: "chromium-swarm.appspot.com"
            recipe {
              name: "recipe"
              cipd_package: "infra/recipe_bundle"
              cipd_version: "refs/heads/master"
            }
          }
        }
        '''
    )
    config.put_bucket('chromium', 'try', self.chromium_try)
    config.put_builders('chromium', 'try', *self.chromium_try.swarming.builders)
    self.create_sync_task = self.patch(
        'swarming.create_sync_task',
        autospec=True,
        return_value={'is_payload': True},
    )
    self.patch('swarming.cancel_task_async', return_value=future(None))

    self.patch(
        'google.appengine.api.app_identity.get_default_version_hostname',
        autospec=True,
        return_value='buildbucket.example.com'
    )

    self.patch('tq.enqueue_async', autospec=True, return_value=future(None))
    self.settings = service_config_pb2.SettingsCfg(
        swarming=dict(global_caches=[dict(path='git')]),
        logdog=dict(hostname='logs.example.com'),
    )
    self.settings.swarming.bbagent_package.builders.regex_exclude.append(
        'chromium/try/linux_legacy',
    )
    self.patch(
        'config.get_settings_async',
        autospec=True,
        return_value=future(self.settings)
    )

    self.patch('creation._should_update_builder', side_effect=lambda p: p > 0.5)
    self.patch(
        'creation._should_enable_experiment', side_effect=lambda p: p > 50
    )

    self.patch('search.TagIndex.random_shard_index', return_value=0)

  @contextlib.contextmanager
  def mutate_builder_cfg(self):
    mutable = self.chromium_try.swarming.builders[0]
    yield mutable
    config.put_bucket('chromium', 'try', self.chromium_try)
    config.put_builders('chromium', 'try', mutable)

  def build_request(self, schedule_build_request_fields=None, **kwargs):
    schedule_build_request_fields = schedule_build_request_fields or {}
    sbr = rpc_pb2.ScheduleBuildRequest(**schedule_build_request_fields)
    sbr.builder.project = sbr.builder.project or 'chromium'
    sbr.builder.bucket = sbr.builder.bucket or 'try'
    sbr.builder.builder = sbr.builder.builder or 'linux'
    return creation.BuildRequest(schedule_build_request=sbr, **kwargs)

  def add(self, *args, **kwargs):
    br = self.build_request(*args, **kwargs)
    return creation.add_async(br).get_result()

  def test_add(self):
    builder_id = builder_pb2.BuilderID(
        project='chromium',
        bucket='try',
        builder='linux',
    )
    build = self.add(dict(builder=builder_id))
    self.assertIsNotNone(build.key)
    self.assertIsNotNone(build.key.id())
    self.assertIsNotNone(build.update_token)

    build = build.key.get()
    self.assertEqual(build.proto.id, build.key.id())
    self.assertEqual(build.proto.builder, builder_id)
    self.assertEqual(
        build.proto.created_by,
        auth.get_current_identity().to_bytes()
    )

    self.assertEqual(build.proto.exe.cmd, ['luciexe'])

    self.assertEqual(build.proto.builder.project, 'chromium')
    self.assertEqual(build.proto.builder.bucket, 'try')
    self.assertEqual(build.proto.builder.builder, 'linux')
    self.assertEqual(build.created_by, auth.get_current_identity())

    infra = model.BuildInfra.key_for(build.key).get().parse()
    self.assertEqual(infra.logdog.hostname, 'logs.example.com')
    self.assertIn(
        build_pb2.BuildInfra.Swarming.CacheEntry(
            path='git', name='git', wait_for_warm_cache=dict()
        ),
        infra.swarming.caches,
    )
    self.assertEqual(build.proto.wait_for_capacity, False)

  def test_add_legacy(self):
    builder_id = builder_pb2.BuilderID(
        project='chromium',
        bucket='try',
        builder='linux_legacy',
    )
    build = self.add(dict(builder=builder_id))
    self.assertEqual(build.proto.exe.cmd, ['recipes'])

  def test_add_wait(self):
    builder_id = builder_pb2.BuilderID(
        project='chromium',
        bucket='try',
        builder='linux_wait',
    )
    build = self.add(dict(builder=builder_id))
    self.assertEqual(build.proto.wait_for_capacity, True)

  def test_add_custom_exe(self):
    builder_id = builder_pb2.BuilderID(
        project='chromium',
        bucket='try',
        builder='linux_modern',
    )
    build = self.add(dict(builder=builder_id))
    self.assertEqual(build.proto.exe.cmd, ['luciexe', '-custom', '-flags'])

    in_props = model.BuildInputProperties.key_for(build.key).get()
    actual = in_props.parse()
    self.assertEqual(actual, bbutil.dict_to_struct({"recipe": "something"}))

  def test_non_existing_builder(self):
    builder_id = builder_pb2.BuilderID(
        project='chromium',
        bucket='try',
        builder='non-existing',
    )
    req = self.build_request(dict(builder=builder_id))
    (_, ex), = creation.add_many_async([req]).get_result()
    self.assertIsInstance(ex, errors.BuilderNotFoundError)

  def test_non_existing_builder_legacy(self):
    config.put_bucket(
        'legacy', 'try', test_util.parse_bucket_cfg('name: "luci.legacy.try"')
    )
    builder_id = builder_pb2.BuilderID(
        project='legacy',
        bucket='try',
        builder='non-existing',
    )
    build = self.add(dict(builder=builder_id))
    self.assertIsNotNone(build)

  def test_critical(self):
    build = self.add(dict(critical=common_pb2.YES))
    self.assertEqual(build.proto.critical, common_pb2.YES)

  def test_critical_default(self):
    build = self.add()
    self.assertEqual(build.proto.critical, common_pb2.UNSET)

  def _test_canary(self, req, is_canary):
    build = self.add(req)
    if is_canary:
      self.assertTrue(build.proto.canary)
      self.assertIn(config.EXPERIMENT_CANARY, build.proto.input.experiments)
      self.assertIn('+%s' % config.EXPERIMENT_CANARY, build.experiments)
    else:
      self.assertFalse(build.proto.canary)
      self.assertNotIn(config.EXPERIMENT_CANARY, build.proto.input.experiments)
      self.assertIn('-%s' % config.EXPERIMENT_CANARY, build.experiments)

  def test_canary_in_request_deprecated(self):
    self._test_canary(dict(canary=common_pb2.NO), False)
    self._test_canary(dict(canary=common_pb2.YES), True)

  def test_canary_in_request(self):
    self._test_canary(
        dict(experiments={config.EXPERIMENT_CANARY: False}), False
    )
    self._test_canary(dict(experiments={config.EXPERIMENT_CANARY: True}), True)

  def test_canary_in_request_conflict(self):
    req = {
        'canary': common_pb2.YES,
        'experiments': {config.EXPERIMENT_CANARY: False},
    }
    self._test_canary(req, False)

  def test_canary_in_builder(self):
    with self.mutate_builder_cfg() as cfg:
      cfg.experiments[config.EXPERIMENT_CANARY] = 10
    self._test_canary({}, False)
    with self.mutate_builder_cfg() as cfg:
      cfg.experiments[config.EXPERIMENT_CANARY] = 100
    self._test_canary({}, True)

  def test_properties(self):
    props = {'foo': 'bar', 'qux': 1}
    prop_struct = bbutil.dict_to_struct(props)
    build = self.add(dict(properties=prop_struct))
    in_props = model.BuildInputProperties.key_for(build.key).get()
    actual = in_props.parse()

    expected = bbutil.dict_to_struct(props)
    expected['recipe'] = 'recipe'
    self.assertEqual(actual, expected)
    infra = model.BuildInfra.key_for(build.key).get().parse()
    self.assertEqual(infra.buildbucket.requested_properties, prop_struct)

  def _test_experimental(self, req, is_experimental):
    build = self.add(req)
    infra = model.BuildInfra.key_for(build.key).get().parse()
    if is_experimental:
      self.assertTrue(build.proto.input.experimental)
      self.assertIn(config.EXPERIMENT_NON_PROD, build.proto.input.experiments)
      self.assertIn('+%s' % config.EXPERIMENT_NON_PROD, build.experiments)
      self.assertEqual(infra.swarming.priority, 60)
    else:
      self.assertFalse(build.proto.input.experimental)
      self.assertNotIn(
          config.EXPERIMENT_NON_PROD, build.proto.input.experiments
      )
      self.assertIn('-%s' % config.EXPERIMENT_NON_PROD, build.experiments)
      self.assertEqual(infra.swarming.priority, 30)

  def test_experimental_in_request_deprecated(self):
    self._test_experimental(dict(experimental=common_pb2.NO), False)
    self._test_experimental(dict(experimental=common_pb2.YES), True)

  def test_experimental_in_request(self):
    self._test_experimental(
        dict(experiments={config.EXPERIMENT_NON_PROD: False}), False
    )
    self._test_experimental(
        dict(experiments={config.EXPERIMENT_NON_PROD: True}), True
    )

  def test_experimental_in_request_conflict(self):
    req = {
        'experimental': common_pb2.YES,
        'experiments': {config.EXPERIMENT_NON_PROD: False},
    }
    self._test_experimental(req, False)

  def test_experimental_in_builder(self):
    with self.mutate_builder_cfg() as cfg:
      cfg.experiments[config.EXPERIMENT_NON_PROD] = 10
    self._test_experimental({}, False)
    with self.mutate_builder_cfg() as cfg:
      cfg.experiments[config.EXPERIMENT_NON_PROD] = 100
    self._test_experimental({}, True)

  def test_builder_config_experiments(self):
    builder_id = builder_pb2.BuilderID(
        project='chromium',
        bucket='try',
        builder='mac_exp',
    )
    build = self.add(dict(builder=builder_id))
    self.assertFalse(build.proto.input.experiments)
    self.assertEqual(build.experiments, [
        '-chromium.exp_foo',
    ])

  def test_schedule_build_request_experiments(self):
    builder_id = builder_pb2.BuilderID(
        project='chromium',
        bucket='try',
        builder='mac_exp',
    )
    build = self.add({
        'builder': builder_id,
        'experiments': {
            'chromium.exp_foo': True,  # override the one in builder config
            'chromium.exp_bar': False,
        }
    })
    self.assertEqual(build.proto.input.experiments, ['chromium.exp_foo'])
    self.assertEqual(
        build.experiments, [
            '+chromium.exp_foo',
            '-chromium.exp_bar',
        ]
    )

  def test_configured_caches(self):
    with self.mutate_builder_cfg() as cfg:
      cfg.caches.add(
          path='required',
          name='1',
      )
      cfg.caches.add(
          path='optional',
          name='1',
          wait_for_warm_cache_secs=60,
      )

    infra = model.BuildInfra.key_for(self.add().key).get().parse()
    caches = infra.swarming.caches
    self.assertIn(
        build_pb2.BuildInfra.Swarming.CacheEntry(
            path='required',
            name='1',
            wait_for_warm_cache=dict(),
        ),
        caches,
    )
    self.assertIn(
        build_pb2.BuildInfra.Swarming.CacheEntry(
            path='optional',
            name='1',
            wait_for_warm_cache=dict(seconds=60),
        ),
        caches,
    )

  def test_configured_cache_overrides_global_one(self):
    with self.mutate_builder_cfg() as cfg:
      cfg.caches.add(
          path='git',
          name='git2',
      )
    infra = model.BuildInfra.key_for(self.add().key).get().parse()
    caches = infra.swarming.caches
    git_caches = [c for c in caches if c.path == 'git']
    self.assertEqual(
        git_caches,
        [
            build_pb2.BuildInfra.Swarming.CacheEntry(
                path='git',
                name='git2',
                wait_for_warm_cache=dict(),
            )
        ],
    )

  def test_builder_cache(self):
    infra = model.BuildInfra.key_for(self.add().key).get().parse()
    caches = infra.swarming.caches

    self.assertIn(
        build_pb2.BuildInfra.Swarming.CacheEntry(
            path='builder',
            name=(
                'builder_ccadafffd20293e0378d1f94d214c63a0f8342d1161454ef0acf'
                'a0405178106b_v2'
            ),
            wait_for_warm_cache=dict(seconds=240),
        ),
        caches,
    )

  def test_builder_cache_overridden(self):
    with self.mutate_builder_cfg() as cfg:
      cfg.caches.add(
          path='builder',
          name='builder',
      )

    infra = model.BuildInfra.key_for(self.add().key).get().parse()
    caches = infra.swarming.caches
    self.assertIn(
        build_pb2.BuildInfra.Swarming.CacheEntry(
            path='builder',
            name='builder',
            wait_for_warm_cache=dict(),
        ),
        caches,
    )

  def test_configured_timeouts(self):
    with self.mutate_builder_cfg() as cfg:
      cfg.expiration_secs = 60
      cfg.execution_timeout_secs = 120

    build = self.add()
    self.assertEqual(build.proto.scheduling_timeout.seconds, 60)
    self.assertEqual(build.proto.execution_timeout.seconds, 120)

  def test_requested_timeouts(self):
    """Ensures that timeouts set in request override defaults."""
    with self.mutate_builder_cfg() as cfg:
      cfg.expiration_secs = 60
      cfg.execution_timeout_secs = 120

    build = self.add(
        dict(
            scheduling_timeout=duration_pb2.Duration(seconds=300),
            execution_timeout=duration_pb2.Duration(seconds=360)
        )
    )
    self.assertEqual(build.proto.scheduling_timeout.seconds, 300)
    self.assertEqual(build.proto.execution_timeout.seconds, 360)

  def test_builder_critical_yes(self):
    with self.mutate_builder_cfg() as cfg:
      cfg.critical = common_pb2.YES

    build = self.add()
    self.assertEqual(build.proto.critical, common_pb2.YES)

  def test_builder_critical_no(self):
    with self.mutate_builder_cfg() as cfg:
      cfg.critical = common_pb2.NO

    build = self.add()
    self.assertEqual(build.proto.critical, common_pb2.NO)

  def test_builder_critical_get_overriden(self):
    with self.mutate_builder_cfg() as cfg:
      cfg.critical = common_pb2.NO

    build = self.add(dict(critical=common_pb2.YES))
    self.assertEqual(build.proto.critical, common_pb2.YES)

  def test_dimensions(self):
    dims = [
        common_pb2.RequestedDimension(key='d', value='1'),
        common_pb2.RequestedDimension(
            key='d', value='1', expiration=dict(seconds=60)
        ),
    ]
    build = self.add(dict(dimensions=dims))

    infra = model.BuildInfra.key_for(build.key).get().parse()
    self.assertEqual(list(infra.buildbucket.requested_dimensions), dims)
    self.assertEqual(list(infra.swarming.task_dimensions), dims)

  def test_dimensions_in_builder(self):
    with self.mutate_builder_cfg() as cfg:
      cfg.dimensions[:] = [
          '60:a:0',
          '0:a:1',
          'b:0',
          'c:1',
          'c:2',
          '60:c:3',
          'tombstone:',
      ]

    dims = [
        common_pb2.RequestedDimension(
            key='b',
            value='1',
            expiration=dict(seconds=60),
        ),
        common_pb2.RequestedDimension(key='c', value='4'),
        common_pb2.RequestedDimension(key='c', value='5'),
        common_pb2.RequestedDimension(
            key='c',
            value='6',
            expiration=dict(seconds=60),
        ),
        common_pb2.RequestedDimension(key='d', value='1'),
    ]
    build = self.add(dict(dimensions=dims))

    infra = model.BuildInfra.key_for(build.key).get().parse()
    self.assertEqual(list(infra.buildbucket.requested_dimensions), dims)
    self.assertEqual(
        list(infra.swarming.task_dimensions), [
            common_pb2.RequestedDimension(
                key='a',
                value='1',
                expiration=dict(seconds=0),
            ),
            common_pb2.RequestedDimension(
                key='a',
                value='0',
                expiration=dict(seconds=60),
            ),
            common_pb2.RequestedDimension(
                key='b',
                value='1',
                expiration=dict(seconds=60),
            ),
            common_pb2.RequestedDimension(key='c', value='4'),
            common_pb2.RequestedDimension(key='c', value='5'),
            common_pb2.RequestedDimension(
                key='c',
                value='6',
                expiration=dict(seconds=60),
            ),
            common_pb2.RequestedDimension(key='d', value='1'),
        ]
    )

  def test_notify(self):
    build = self.add(
        dict(
            notify=dict(
                pubsub_topic='projects/p/topics/t',
                user_data='hello',
            )
        ),
    )
    self.assertEqual(build.pubsub_callback.topic, 'projects/p/topics/t')
    self.assertEqual(build.pubsub_callback.user_data, 'hello')

  def test_gitiles_commit(self):
    gitiles_commit = common_pb2.GitilesCommit(
        host='gitiles.example.com',
        project='chromium/src',
        ref='refs/heads/master',
        id='b7a757f457487cd5cfe2dae83f65c5bc10e288b7',
        position=1,
    )

    build = self.add(dict(gitiles_commit=gitiles_commit))
    bs = (
        'commit/gitiles/gitiles.example.com/chromium/src/+/'
        'b7a757f457487cd5cfe2dae83f65c5bc10e288b7'
    )
    self.assertIn('buildset:' + bs, build.tags)
    self.assertIn('gitiles_ref:refs/heads/master', build.tags)

  def test_gitiles_commit_without_id(self):
    gitiles_commit = common_pb2.GitilesCommit(
        host='gitiles.example.com',
        project='chromium/src',
        ref='refs/heads/master',
    )

    build = self.add(dict(gitiles_commit=gitiles_commit))
    self.assertFalse(any(t.startswith('buildset:commit') for t in build.tags))
    self.assertFalse(any(t.startswith('gititles_ref:') for t in build.tags))

  def test_gerrit_change(self):
    cl = common_pb2.GerritChange(
        host='gerrit.example.com',
        change=1234,
        patchset=5,
    )
    build = self.add(dict(gerrit_changes=[cl]))
    self.assertEqual(build.proto.input.gerrit_changes[:], [cl])
    bs = 'patch/gerrit/gerrit.example.com/1234/5'
    self.assertIn('buildset:' + bs, build.tags)

  def test_priority(self):
    build = self.add(dict(priority=42))
    infra = model.BuildInfra.key_for(build.key).get().parse()
    self.assertEqual(infra.swarming.priority, 42)

  def test_parent_run_id(self):
    build = self.add(
        schedule_build_request_fields=dict(
            swarming=dict(parent_run_id='deadbeef')
        )
    )
    infra = model.BuildInfra.key_for(build.key).get().parse()
    self.assertEqual(infra.swarming.parent_run_id, 'deadbeef')

  def test_update_builders(self):
    recently = self.now - datetime.timedelta(minutes=1)
    while_ago = self.now - datetime.timedelta(minutes=61)
    ndb.put_multi([
        model.Builder(id='chromium:try:linux', last_scheduled=recently),
        model.Builder(id='chromium:try:mac', last_scheduled=while_ago),
    ])

    creation.add_many_async([
        self.build_request(dict(builder=dict(builder='linux'))),
        self.build_request(dict(builder=dict(builder='mac'))),
        self.build_request(dict(builder=dict(builder='win'))),
    ]).get_result()

    builders = model.Builder.query().fetch()
    self.assertEqual(len(builders), 3)
    self.assertEqual(builders[0].key.id(), 'chromium:try:linux')
    self.assertEqual(builders[0].last_scheduled, recently)
    self.assertEqual(builders[1].key.id(), 'chromium:try:mac')
    self.assertEqual(builders[1].last_scheduled, self.now)
    self.assertEqual(builders[2].key.id(), 'chromium:try:win')
    self.assertEqual(builders[2].last_scheduled, self.now)

  def test_request_id(self):
    build = self.add(dict(request_id='1'))
    build2 = self.add(dict(request_id='1'))
    self.assertIsNotNone(build.key)
    self.assertEqual(build, build2)

  def test_leasing(self):
    build = self.add(
        lease_expiration_date=utils.utcnow() + datetime.timedelta(seconds=10),
    )
    self.assertTrue(build.is_leased)
    self.assertGreater(build.lease_expiration_date, utils.utcnow())
    self.assertIsNotNone(build.lease_key)

  def test_builder_tag(self):
    build = self.add(dict(builder=dict(builder='linux')))
    self.assertTrue('builder:linux' in build.tags)

  def test_builder_tag_coincide(self):
    build = self.add(
        dict(
            builder=dict(builder='linux'),
            tags=[dict(key='builder', value='linux')],
        )
    )
    self.assertIn('builder:linux', build.tags)

  def test_buildset_index(self):
    build = self.add(
        dict(
            tags=[
                dict(key='buildset', value='foo'),
                dict(key='buildset', value='bar'),
            ]
        )
    )

    for t in ('buildset:foo', 'buildset:bar'):
      index = search.TagIndex.get_by_id(t)
      self.assertIsNotNone(index)
      self.assertEqual(len(index.entries), 1)
      self.assertEqual(index.entries[0].build_id, build.key.id())
      self.assertEqual(index.entries[0].bucket_id, build.bucket_id)

  def test_buildset_index_with_request_id(self):
    build = self.add(
        dict(
            tags=[dict(key='buildset', value='foo')],
            request_id='0',
        )
    )

    index = search.TagIndex.get_by_id('buildset:foo')
    self.assertIsNotNone(index)
    self.assertEqual(len(index.entries), 1)
    self.assertEqual(index.entries[0].build_id, build.key.id())
    self.assertEqual(index.entries[0].bucket_id, build.bucket_id)

  def test_buildset_index_existing(self):
    search.TagIndex(
        id='buildset:foo',
        entries=[
            search.TagIndexEntry(
                build_id=int(2**63 - 1),
                bucket_id='chromium/try',
            ),
            search.TagIndexEntry(
                build_id=0,
                bucket_id='chromium/try',
            ),
        ]
    ).put()
    build = self.add(dict(tags=[dict(key='buildset', value='foo')]))
    index = search.TagIndex.get_by_id('buildset:foo')
    self.assertIsNotNone(index)
    self.assertEqual(len(index.entries), 3)
    self.assertIn(build.key.id(), [e.build_id for e in index.entries])
    self.assertIn(build.bucket_id, [e.bucket_id for e in index.entries])

  def test_many(self):
    results = creation.add_many_async([
        self.build_request(dict(tags=[dict(key='buildset', value='a')])),
        self.build_request(dict(tags=[dict(key='buildset', value='a')])),
    ]).get_result()
    self.assertEqual(len(results), 2)
    self.assertIsNotNone(results[0][0])
    self.assertIsNone(results[0][1])
    self.assertIsNotNone(results[1][0])
    self.assertIsNone(results[1][1])

    self.assertEqual(
        results, sorted(results, key=lambda (b, _): b.key.id(), reverse=True)
    )
    results.reverse()

    index = search.TagIndex.get_by_id('buildset:a')
    self.assertIsNotNone(index)
    self.assertEqual(len(index.entries), 2)
    self.assertEqual(index.entries[0].build_id, results[1][0].key.id())
    self.assertEqual(index.entries[0].bucket_id, results[1][0].bucket_id)
    self.assertEqual(index.entries[1].build_id, results[0][0].key.id())
    self.assertEqual(index.entries[1].bucket_id, results[0][0].bucket_id)

  def test_many_with_request_id(self):
    req1 = self.build_request(
        dict(
            tags=[dict(key='buildset', value='a')],
            request_id='0',
        ),
    )
    req2 = self.build_request(dict(tags=[dict(key='buildset', value='a')]))
    creation.add_async(req1).get_result()
    creation.add_many_async([req1, req2]).get_result()

    # Build for req1 must be added only once.
    idx = search.TagIndex.get_by_id('buildset:a')
    self.assertEqual(len(idx.entries), 2)
    self.assertEqual(idx.entries[0].bucket_id, 'chromium/try')

  def test_create_sync_task(self):
    expected_ex1 = errors.InvalidInputError()

    def create_sync_task(build, *_args, **_kwargs):
      if 'buildset:a' in build.tags:
        raise expected_ex1

    self.create_sync_task.side_effect = create_sync_task

    ((b1, ex1), (b2, ex2)) = creation.add_many_async([
        self.build_request(dict(tags=[dict(key='buildset', value='a')])),
        self.build_request(dict(tags=[dict(key='buildset', value='b')])),
    ]).get_result()

    self.assertEqual(ex1, expected_ex1)
    self.assertIsNone(b1)
    self.assertIsNone(ex2)
    self.assertIsNotNone(b2)

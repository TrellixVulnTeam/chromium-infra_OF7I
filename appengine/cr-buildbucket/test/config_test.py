# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import copy
import hashlib
import logging

from parameterized import parameterized

from components import utils

utils.fix_protobuf_package()

from google.appengine.api import memcache
from google.appengine.ext import ndb
from google.protobuf import text_format

from components import config as config_component
from components.config import validation_context
from testing_utils import testing
import mock

from go.chromium.org.luci.buildbucket.proto import build_pb2
from go.chromium.org.luci.buildbucket.proto import builder_pb2
from go.chromium.org.luci.buildbucket.proto import project_config_pb2
from go.chromium.org.luci.buildbucket.proto import service_config_pb2
from test import test_util
import config
import errors


def short_bucket_cfg(cfg):
  cfg = copy.deepcopy(cfg)
  cfg.name = config.short_bucket_name(cfg.name)
  return cfg


def without_builders(cfg):
  cfg = copy.deepcopy(cfg)
  if config.is_swarming_config(cfg):
    del cfg.swarming.builders[:]
  return cfg


# NOTE: This has the pool dimension, but other uses of LUCI_CHROMIUM_TRY
# do NOT list pool dimension explicitly; This is because config.py has api
# little-used feature where pool is 'automatically' filled in from the project
# and bucket name as 'luci.$project.$bucket' if it's omitted.
LUCI_CHROMIUM_TRY = test_util.parse_bucket_cfg(
    '''
    name: "luci.chromium.try"
    acls {
      role: READER
      group: "all"
    }
    acls {
      role: SCHEDULER
      identity: "user:johndoe@example.com"
    }
    swarming {
      task_template_canary_percentage { value: 10 }
      builders {
        name: "linux"
        swarming_host: "swarming.example.com"
        task_template_canary_percentage { value: 10 }
        dimensions: "os:Linux"
        dimensions: "pool:luci.chromium.try"
        recipe {
          cipd_package: "infra/recipe_bundle"
          cipd_version: "refs/heads/master"
          name: "x"
        }
      }
    }
'''
)

LUCI_DART_TRY = test_util.parse_bucket_cfg(
    '''
    name: "luci.dart.try"
    swarming {
      builders {
        name: "linux"
        dimensions: "pool:Dart.LUCI"
        recipe {
          cipd_package: "infra/recipe_bundle"
          cipd_version: "refs/heads/master"
          name: "x"
        }
      }
    }
    '''
)

MASTER_TRYSERVER_CHROMIUM_LINUX = test_util.parse_bucket_cfg(
    '''
    name: "master.tryserver.chromium.linux"
    acls {
      role: READER
      group: "all"
    }
    acls {
      role: SCHEDULER
      group: "tryjob-access"
    }
    '''
)

MASTER_TRYSERVER_CHROMIUM_WIN = test_util.parse_bucket_cfg(
    '''
    name: "master.tryserver.chromium.win"
    acls {
      role: READER
      group: "all"
    }
    acls {
      role: SCHEDULER
      group: "tryjob-access"
    }
    '''
)

MASTER_TRYSERVER_CHROMIUM_MAC = test_util.parse_bucket_cfg(
    '''
    name: "master.tryserver.chromium.mac"
    acls {
      role: READER
      group: "all"
    }
    acls {
      role: SCHEDULER
      group: "tryjob-access"
    }
    '''
)

MASTER_TRYSERVER_V8 = test_util.parse_bucket_cfg(
    '''
    name: "master.tryserver.v8"
    acls {
      role: WRITER
      group: "v8-team"
    }
    '''
)


def parse_cfg(text):
  cfg = project_config_pb2.BuildbucketCfg()
  text_format.Merge(text, cfg)
  return cfg


def errmsg(text):
  return validation_context.Message(
      severity=logging.ERROR, text=text.decode('utf-8')
  )


class ConfigTest(testing.AppengineTestCase):

  def test_get_bucket(self):
    config.put_bucket('chromium', 'deadbeef', LUCI_CHROMIUM_TRY)
    rev, cfg = config.get_bucket('chromium/try')
    self.assertEqual(rev, 'deadbeef')
    self.assertEqual(cfg, without_builders(short_bucket_cfg(LUCI_CHROMIUM_TRY)))

    self.assertIsNone(config.get_bucket('chromium/nonexistent')[0])

  def test_get_buckets_async(self):
    config.put_bucket('chromium', 'deadbeef', MASTER_TRYSERVER_CHROMIUM_LINUX)
    config.put_builders(
        'chromium', 'master.tryserver.chromium.linux',
        *MASTER_TRYSERVER_CHROMIUM_LINUX.swarming.builders
    )
    config.put_bucket('chromium', 'deadbeef', LUCI_CHROMIUM_TRY)
    config.put_builders('chromium', 'try', *LUCI_CHROMIUM_TRY.swarming.builders)
    config.put_bucket('dart', 'deadbeef', LUCI_DART_TRY)
    config.put_builders('dart', 'try', *LUCI_DART_TRY.swarming.builders)

    # Without builders.
    actual = config.get_buckets_async().get_result()
    expected = {
        'chromium/master.tryserver.chromium.linux':
            without_builders(MASTER_TRYSERVER_CHROMIUM_LINUX),
        'chromium/try':
            without_builders(short_bucket_cfg(LUCI_CHROMIUM_TRY)),
        'dart/try':
            without_builders(short_bucket_cfg(LUCI_DART_TRY)),
    }
    self.assertEqual(actual, expected)

    # With builders.
    actual = config.get_buckets_async(include_builders=True).get_result()
    expected = {
        'chromium/master.tryserver.chromium.linux':
            MASTER_TRYSERVER_CHROMIUM_LINUX,
        'chromium/try':
            short_bucket_cfg(LUCI_CHROMIUM_TRY),
        'dart/try':
            short_bucket_cfg(LUCI_DART_TRY),
    }
    self.assertEqual(actual, expected)

  def test_get_buckets_async_with_bucket_ids(self):
    config.put_bucket('chromium', 'deadbeef', LUCI_CHROMIUM_TRY)
    config.put_builders('chromium', 'try', *LUCI_CHROMIUM_TRY.swarming.builders)
    config.put_bucket('chromium', 'deadbeef', MASTER_TRYSERVER_CHROMIUM_WIN)
    config.put_builders(
        'chromium', 'master.tryserver.chromium.win',
        *MASTER_TRYSERVER_CHROMIUM_WIN.swarming.builders
    )

    # Without builders.
    actual = config.get_buckets_async(bucket_ids=['chromium/try']).get_result()
    expected = {
        'chromium/try': without_builders(short_bucket_cfg(LUCI_CHROMIUM_TRY))
    }
    self.assertEqual(actual, expected)

    # With builders.
    actual = config.get_buckets_async(
        bucket_ids=['chromium/master.tryserver.chromium.win'],
        include_builders=True
    ).get_result()
    expected = {
        'chromium/master.tryserver.chromium.win': MASTER_TRYSERVER_CHROMIUM_WIN
    }
    self.assertEqual(actual, expected)

  def test_get_buckets_async_with_bucket_ids_not_found(self):
    bid = 'chromium/try'
    actual = config.get_buckets_async([bid]).get_result()
    self.assertEqual(actual, {bid: None})

  def test_get_all_bucket_ids_cold_cache(self):
    config.put_bucket(
        'chromium', 'deadbeef', test_util.parse_bucket_cfg('name: "try"')
    )
    config.put_bucket(
        'chromium', 'deadbeef', test_util.parse_bucket_cfg('name: "ci"')
    )
    config.put_bucket(
        'v8', 'deadbeef', test_util.parse_bucket_cfg('name: "ci"')
    )
    ids = config.get_all_bucket_ids_async().get_result()
    self.assertEqual(ids, ['chromium/ci', 'chromium/try', 'v8/ci'])

  def resolve_bucket(self, bucket_name):
    return config.resolve_bucket_name_async(bucket_name).get_result()

  def test_resolve_bucket_name_async_does_not_exist(self):
    self.assertIsNone(self.resolve_bucket('try'))

  def test_resolve_bucket_name_async_unique(self):
    config.put_bucket('chromium', 'deadbeef', LUCI_CHROMIUM_TRY)
    self.assertEqual(self.resolve_bucket('try'), 'chromium/try')

  def test_resolve_bucket_name_async_ambiguous(self):
    config.put_bucket('chromium', 'deadbeef', LUCI_CHROMIUM_TRY)
    config.put_bucket('dart', 'deadbeef', LUCI_DART_TRY)
    with self.assertRaisesRegexp(errors.InvalidInputError, r'ambiguous'):
      self.resolve_bucket('try')

  def test_resolve_bucket_name_async_cache_key(self):
    config.put_bucket('chromium', 'deadbeef', LUCI_CHROMIUM_TRY)
    config.put_bucket('chromium', 'deadbeef', MASTER_TRYSERVER_CHROMIUM_LINUX)
    self.assertEqual(self.resolve_bucket('try'), 'chromium/try')
    self.assertEqual(
        self.resolve_bucket('master.tryserver.chromium.linux'),
        'chromium/master.tryserver.chromium.linux'
    )

  @mock.patch('components.config.get_project_configs', autospec=True)
  def test_cron_update_buckets(self, get_project_configs):
    chromium_buildbucket_cfg = parse_cfg(
        '''
        acl_sets {
          name: "public"
          acls {
            role: READER
            group: "all"
          }

          # Test deduplication.
          acls {
            group: "all"
          }
        }
        buckets {
          name: "try"
          acl_sets: "public"
          acls {
            role: SCHEDULER
            identity: "johndoe@example.com"
          }
          swarming {
            task_template_canary_percentage { value: 10 }
            builders {
              name: "linux"
              dimensions: "os:Linux"
              recipe {
                name: "x"
                cipd_version: "refs/heads/master"
                cipd_package: "infra/recipe_bundle"
              }
              swarming_host: "swarming.example.com"
              task_template_canary_percentage {
                value: 10
              }
            }
          }
        }
        buckets {
          name: "master.tryserver.chromium.linux"
          acl_sets: "public"
          acl_sets: "undefined_acl_set_will_log_an_error_but_not_failure"
          acls {
            role: SCHEDULER
            group: "tryjob-access"
          }
        }

        buckets {
          name: "master.tryserver.chromium.win"
          acls {
            role: READER
            group: "all"
          }
          acls {
            role: SCHEDULER
            group: "tryjob-access"
          }
        }
        '''
    )

    dart_buildbucket_cfg = parse_cfg(
        '''
      buckets {
        name: "try"
        swarming {
          builders {
            name: "linux"
            dimensions: "pool:Dart.LUCI"
            recipe {
              cipd_package: "infra/recipe_bundle"
              cipd_version: "refs/heads/master"
              name: "x"
            }
          }
        }
      }
      '''
    )

    v8_buildbucket_cfg = parse_cfg(
        '''
      buckets {
        name: "master.tryserver.v8"
        acls {
          role: WRITER
          group: "v8-team"
        }
      }
      '''
    )

    get_project_configs.return_value = {
        'chromium': ('deadbeef', chromium_buildbucket_cfg, None),
        'dart': ('deadbeef', dart_buildbucket_cfg, None),
        'v8': (None, v8_buildbucket_cfg, None),
    }

    config.cron_update_buckets()

    actual = config.Bucket.query().fetch()
    actual = sorted(actual, key=lambda b: b.key)
    expected = [
        config.Bucket(
            parent=ndb.Key(config.Project, 'chromium'),
            id='master.tryserver.chromium.linux',
            entity_schema_version=config.CURRENT_BUCKET_SCHEMA_VERSION,
            revision='deadbeef',
            config=without_builders(MASTER_TRYSERVER_CHROMIUM_LINUX),
        ),
        config.Bucket(
            parent=ndb.Key(config.Project, 'chromium'),
            id='master.tryserver.chromium.win',
            entity_schema_version=config.CURRENT_BUCKET_SCHEMA_VERSION,
            revision='deadbeef',
            config=without_builders(MASTER_TRYSERVER_CHROMIUM_WIN),
        ),
        config.Bucket(
            parent=ndb.Key(config.Project, 'chromium'),
            id='try',
            entity_schema_version=config.CURRENT_BUCKET_SCHEMA_VERSION,
            revision='deadbeef',
            config=without_builders(short_bucket_cfg(LUCI_CHROMIUM_TRY)),
        ),
        config.Bucket(
            parent=ndb.Key(config.Project, 'dart'),
            id='try',
            entity_schema_version=config.CURRENT_BUCKET_SCHEMA_VERSION,
            revision='deadbeef',
            config=without_builders(short_bucket_cfg(LUCI_DART_TRY)),
        ),
        config.Bucket(
            parent=ndb.Key(config.Project, 'v8'),
            id='master.tryserver.v8',
            entity_schema_version=config.CURRENT_BUCKET_SCHEMA_VERSION,
            revision='sha1:cfc761d7a953a72ddea8f3d4c9a28e69777ca22c',
            config=without_builders(MASTER_TRYSERVER_V8),
        ),
    ]
    self.assertEqual(actual, expected)

    actual = config.Builder.query().fetch()
    actual = sorted(actual, key=lambda b: b.key)
    expected = [
        config.Builder(
            parent=ndb.Key(config.Project, 'chromium', config.Bucket, 'try'),
            id='linux',
            config=LUCI_CHROMIUM_TRY.swarming.builders[0],
            config_hash=hashlib.sha256(
                LUCI_CHROMIUM_TRY.swarming.builders[0].SerializeToString(
                    deterministic=True
                )
            ).hexdigest(),
        ),
        config.Builder(
            parent=ndb.Key(config.Project, 'dart', config.Bucket, 'try'),
            id='linux',
            config=LUCI_DART_TRY.swarming.builders[0],
            config_hash=hashlib.sha256(
                LUCI_DART_TRY.swarming.builders[0].SerializeToString(
                    deterministic=True
                )
            ).hexdigest(),
        ),
    ]
    self.assertEqual(actual, expected)

  @mock.patch('components.config.get_project_configs', autospec=True)
  def test_cron_update_buckets_with_existing(self, get_project_configs):
    chromium_buildbucket_cfg = parse_cfg(
        '''
        buckets {
          name: "master.tryserver.chromium.linux"
          acls {
            role: READER
            group: "all"
          }
          acls {
            role: SCHEDULER
            group: "tryjob-access"
          }
        }
        buckets {
          name: "master.tryserver.chromium.mac"
          acls {
            role: READER
            group: "all"
          }
          acls {
            role: SCHEDULER
            group: "tryjob-access"
          }
        }
        buckets {
          name: "try"
          acls {
            role: READER
            group: "all"
          }
          acls {
            role: SCHEDULER
            identity: "johndoe@example.com"
          }
          swarming {
            task_template_canary_percentage { value: 10 }
            builders {
              name: "linux"
              swarming_host: "swarming.example.com"
              task_template_canary_percentage { value: 10 }
              dimensions: "os:Linux"
              recipe {
                cipd_package: "infra/recipe_bundle"
                cipd_version: "refs/heads/master"
                name: "x"
              }
            }
          }
        }
        '''
    )

    dart_buildbucket_cfg = parse_cfg(
        '''
      buckets {
        name: "try"
        swarming {
          builders {
            name: "linux"
            dimensions: "pool:Dart.LUCI"
            recipe {
              cipd_package: "infra/recipe_bundle"
              cipd_version: "refs/heads/master"
              name: "x"
            }
          }
        }
      }
      '''
    )

    v8_buildbucket_cfg = parse_cfg(
        '''
        buckets {
          name: "master.tryserver.v8"
          acls {
            role: WRITER
            group: "v8-team"
          }
        }
        '''
    )
    get_project_configs.return_value = {
        'chromium': ('new!', chromium_buildbucket_cfg, None),
        'dart': ('deadbeef', dart_buildbucket_cfg, None),
        'v8': ('deadbeef', v8_buildbucket_cfg, None),
    }

    # Will be updated.
    config.put_bucket('chromium', 'deadbeef', MASTER_TRYSERVER_CHROMIUM_LINUX)
    # Will not be updated.
    config.put_bucket('v8', 'deadbeef', MASTER_TRYSERVER_V8)
    # Will be deleted.
    config.put_bucket('chromium', 'deadbeef', MASTER_TRYSERVER_CHROMIUM_WIN)
    # Will be deleted.
    config.put_bucket('chromium', 'deadbeef', LUCI_CHROMIUM_TRY)

    # Will be updated.
    config.put_builders(
        'chromium', 'try', project_config_pb2.Builder(name='linux')
    )
    # Will not be updated.
    config.put_builders('dart', 'try', LUCI_DART_TRY.swarming.builders[0])
    # Will be deleted.
    config.put_builders('dart', 'try', project_config_pb2.Builder(name='linux'))
    # Will be deleted.
    config.put_builders(
        'chromium', 'try', LUCI_CHROMIUM_TRY.swarming.builders[0]
    )

    config.cron_update_buckets()

    actual = config.Bucket.query().fetch()
    actual = sorted(actual, key=lambda b: b.key)
    expected = [
        config.Bucket(
            parent=ndb.Key(config.Project, 'chromium'),
            id='master.tryserver.chromium.linux',
            entity_schema_version=config.CURRENT_BUCKET_SCHEMA_VERSION,
            revision='new!',
            config=without_builders(MASTER_TRYSERVER_CHROMIUM_LINUX),
        ),
        config.Bucket(
            parent=ndb.Key(config.Project, 'chromium'),
            id='master.tryserver.chromium.mac',
            entity_schema_version=config.CURRENT_BUCKET_SCHEMA_VERSION,
            revision='new!',
            config=without_builders(MASTER_TRYSERVER_CHROMIUM_MAC),
        ),
        config.Bucket(
            parent=ndb.Key(config.Project, 'chromium'),
            id='try',
            entity_schema_version=config.CURRENT_BUCKET_SCHEMA_VERSION,
            revision='new!',
            config=without_builders(short_bucket_cfg(LUCI_CHROMIUM_TRY)),
        ),
        config.Bucket(
            parent=ndb.Key(config.Project, 'dart'),
            id='try',
            entity_schema_version=config.CURRENT_BUCKET_SCHEMA_VERSION,
            revision='deadbeef',
            config=without_builders(short_bucket_cfg(LUCI_DART_TRY)),
        ),
        config.Bucket(
            parent=ndb.Key(config.Project, 'v8'),
            id='master.tryserver.v8',
            entity_schema_version=config.CURRENT_BUCKET_SCHEMA_VERSION,
            revision='deadbeef',
            config=without_builders(MASTER_TRYSERVER_V8),
        ),
    ]
    self.assertEqual(actual, expected)

    actual = config.Builder.query().fetch()
    actual = sorted(actual, key=lambda b: b.key)
    expected = [
        config.Builder(
            parent=ndb.Key(config.Project, 'chromium', config.Bucket, 'try'),
            id='linux',
            config=LUCI_CHROMIUM_TRY.swarming.builders[0],
            config_hash=hashlib.sha256(
                LUCI_CHROMIUM_TRY.swarming.builders[0].SerializeToString(
                    deterministic=True
                )
            ).hexdigest(),
        ),
        config.Builder(
            parent=ndb.Key(config.Project, 'dart', config.Bucket, 'try'),
            id='linux',
            config=LUCI_DART_TRY.swarming.builders[0],
            config_hash=hashlib.sha256(
                LUCI_DART_TRY.swarming.builders[0].SerializeToString(
                    deterministic=True
                )
            ).hexdigest(),
        ),
    ]
    self.assertEqual(actual, expected)

    # Updated get_all_bucket_ids_async cache.
    actual = memcache.get(config._MEMCACHE_ALL_BUCKET_IDS_KEY)
    expected = [
        'chromium/master.tryserver.chromium.linux',
        'chromium/master.tryserver.chromium.mac',
        'chromium/try',
        'dart/try',
        'v8/master.tryserver.v8',
    ]
    self.assertEqual(actual, expected)

    # Coverage for the warm cache case.
    actual = config.get_all_bucket_ids_async().get_result()
    self.assertEqual(actual, expected)

  @mock.patch('components.config.get_project_configs', autospec=True)
  def test_cron_update_buckets_with_broken_configs(self, get_project_configs):
    config.put_bucket('dart', 'deadbeef', LUCI_DART_TRY)
    config.put_builders('dart', 'try', LUCI_DART_TRY.swarming.builders[0])

    get_project_configs.return_value = {
        'dart': ('new!', None, config_component.ConfigFormatError('broken!')),
    }

    config.cron_update_buckets()

    # We must not delete buckets or builders defined in a project that currently
    # have a broken config.
    _, actual = config.get_bucket('dart/try')
    self.assertEqual(actual, without_builders(short_bucket_cfg(LUCI_DART_TRY)))
    actual = config.Builder.make_key('dart', 'try', 'linux').get()
    self.assertTrue(actual)
    self.assertEqual(actual.config, LUCI_DART_TRY.swarming.builders[0])

  def cfg_validation_test(self, cfg, expected_messages):
    ctx = config_component.validation.Context()
    ctx.config_set = 'projects/chromium'
    config.validate_buildbucket_cfg(cfg, ctx)
    self.assertEqual(expected_messages, ctx.result().messages)

  def test_validate_buildbucket_cfg_success(self):
    self.cfg_validation_test(
        parse_cfg(
            '''
      acl_sets {
        name: "public"
        acls {
          role: READER
          group: "all"
        }
      }
      buckets {
        name: "good.name"
        acls {
          role: WRITER
          group: "writers"
        }
      }
      buckets {
        name: "good.name2"
        acl_sets: "public"
        acls {
          role: READER
          identity: "a@a.com"
        }
        acls {
          role: READER
          identity: "user:b@a.com"
        }
      }
      '''
        ), []
    )

  def test_validate_buildbucket_cfg_fail(self):
    self.cfg_validation_test(
        parse_cfg(
            '''
      acl_sets {}
      acl_sets {
        name: "^"
        acls {}
      }
      acl_sets { name: "a" }
      acl_sets { name: "a" }
      buckets {
        name: "a"
        acl_sets: "does_not_exist"
        acls {
          role: READER
          group: "writers"
          identity: "a@a.com"
        }
        acls {
          role: READER
        }
      }
      buckets {
        name: "a"
        acls {
          role: READER
          identity: "ldap"
        }
        acls {
          role: READER
          group: ";%:"
        }
      }
      buckets {}
      buckets { name: "luci.x" }
      '''
        ), [
            errmsg('ACL set #1 (): name is unspecified'),
            errmsg(
                'ACL set #2 (^): invalid name "^" does not match regex '
                '\'^[a-z0-9_]+$\''
            ),
            errmsg('ACL set #2 (^): acl #1: group or identity must be set'),
            errmsg('ACL set #4 (a): duplicate name "a"'),
            errmsg(
                'Bucket a: acl #1: either group or identity must be set, '
                'not both'
            ),
            errmsg('Bucket a: acl #2: group or identity must be set'),
            errmsg(
                'Bucket a: undefined ACL set "does_not_exist". '
                'It must be defined in the same file'
            ),
            errmsg('Bucket a: duplicate bucket name'),
            errmsg('Bucket a: acl #1: Identity has invalid format: ldap'),
            errmsg('Bucket a: acl #2: invalid group: ;%:'),
            errmsg('Bucket #3: invalid name: Bucket not specified'),
            errmsg(
                'Bucket luci.x: invalid name: Bucket must start with '
                '"luci.chromium." because it starts with "luci." and is defined'
                ' in the chromium project'
            ),
        ]
    )

  def test_validate_buildbucket_cfg_unsorted(self):
    self.cfg_validation_test(
        parse_cfg(
            '''
            buckets { name: "c" }
            buckets { name: "b" }
            buckets { name: "a" }
            '''
        ),
        [
            validation_context.Message(
                severity=logging.WARNING,
                text='Bucket b: out of order',
            ),
            validation_context.Message(
                severity=logging.WARNING,
                text='Bucket a: out of order',
            ),
        ],
    )

  @mock.patch('components.config.get_config_set_location', autospec=True)
  def test_get_buildbucket_cfg_url(self, get_config_set_location):
    get_config_set_location.return_value = (
        'https://chromium.googlesource.com/chromium/src/+/infra/config'
    )

    url = config.get_buildbucket_cfg_url('chromium')
    self.assertEqual(
        url, (
            'https://chromium.googlesource.com/chromium/src/+/'
            'refs/heads/infra/config/testbed-test.cfg'
        )
    )


class ValidateBucketIDTest(testing.AppengineTestCase):

  @parameterized.expand([
      ('chromium/try',),
      ('chrome-internal/try',),
      ('chrome-internal/try.x',),
  ])
  def test_valid(self, bucket_id):
    config.validate_bucket_id(bucket_id)

  @parameterized.expand([
      ('a/b/c',),
      ('a b/c',),
      ('chromium/luci.chromium.try',),
  ])
  def test_invalid(self, bucket_id):
    with self.assertRaises(errors.InvalidInputError):
      config.validate_bucket_id(bucket_id)

  @parameterized.expand([
      ('chromium',),
      ('a:b',),
  ])
  def test_assertions(self, bucket_id):
    # We must never pass legacy bucket id to validate_bucket_id().
    with self.assertRaises(AssertionError):
      config.validate_bucket_id(bucket_id)


class BuilderMatchesTest(testing.AppengineTestCase):

  @parameterized.expand([
      ([], [], True),
      ([], ['chromium/.+'], False),
      ([], ['v8/.+'], True),
      (['chromium/.+'], [], True),
      (['v8/.+'], [], False),
  ])
  def test_builder_matches(self, regex, regex_exclude, expected):
    predicate = service_config_pb2.BuilderPredicate(
        regex=regex, regex_exclude=regex_exclude
    )
    builder_id = builder_pb2.BuilderID(
        project='chromium',
        bucket='try',
        builder='linux-rel',
    )
    actual = config.builder_matches(builder_id, predicate)
    self.assertEqual(expected, actual)

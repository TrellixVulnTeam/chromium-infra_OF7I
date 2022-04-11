# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import datetime
import json

from components import auth
from components import auth_testing
from testing_utils import testing

from legacy import api_common
from legacy import swarmbucket_api
from go.chromium.org.luci.buildbucket.proto import service_config_pb2
from test import test_util
from test.test_util import future, mock_permissions
import config
import experiments
import model
import sequence
import swarming
import user


class SwarmbucketApiTest(testing.EndpointsTestCase):
  api_service_cls = swarmbucket_api.SwarmbucketApi
  maxDiff = None

  def setUp(self):
    super(SwarmbucketApiTest, self).setUp()

    self.patch(
        'components.utils.utcnow',
        autospec=True,
        return_value=datetime.datetime(2015, 11, 30)
    )

    self.patch(
        'google.appengine.api.app_identity.get_default_version_hostname',
        return_value='cr-buildbucket.appspot.com'
    )

    self.perms = mock_permissions(self)

    chromium_cfg = test_util.parse_bucket_cfg(
        '''
          name: "luci.chromium.try"
          swarming {
            hostname: "swarming.example.com"
            builders {
              name: "linux"
              swarming_host: "swarming.example.com"
              category: "Chromium"
              build_numbers: YES
              recipe {
                cipd_package: "infra/recipe_bundle"
                cipd_version: "refs/heads/master"
                name: "presubmit"
                properties: "foo:bar"
                properties_j: "baz:1"
              }
              dimensions: "foo:bar"
              dimensions: "baz:baz"
              auto_builder_dimension: YES

              experiments: {
                key:   "luci.buildbucket.use_bbagent"
                value: 100
              }

              # Override builder cache without timeout to make tests
              # simpler.
              caches {
                path: "builder"
                name: "builder_cache_name"
              }
              resultdb {
                  enable: True
              }
            }
            builders {
              name: "windows"
              category: "Chromium"
              swarming_host: "swarming.example.com"
              recipe {
                cipd_package: "infra/recipe_bundle"
                cipd_version: "refs/heads/master"
                name: "presubmit"
              }

              # Override builder cache without timeout to make tests
              # simpler.
              caches {
                path: "builder"
                name: "builder_cache_name"
              }
            }
          }
    '''
    )
    config.put_bucket('chromium', 'deadbeef', chromium_cfg)
    config.put_builders('chromium', 'try', *chromium_cfg.swarming.builders)
    self.perms['chromium/try'] = [
        user.PERM_BUCKETS_GET,
        user.PERM_BUILDERS_LIST,
    ]

    v8_cfg = test_util.parse_bucket_cfg('''name: "luci.v8.try"''')
    config.put_bucket('v8', 'deadbeef', v8_cfg)
    self.perms['v8/try'] = [
        user.PERM_BUCKETS_GET,
        user.PERM_BUILDERS_LIST,
    ]

    self.settings = service_config_pb2.SettingsCfg(
        swarming=dict(
            milo_hostname='milo.example.com',
            bbagent_package=dict(
                package_name='infra/tools/bbagent',
                version='luci-runner-version',
            ),
            kitchen_package=dict(
                package_name='infra/tools/kitchen',
                version='kitchen-version',
            ),
            user_packages=[
                dict(
                    package_name='infra/tools/git',
                    version='git-version',
                ),
            ],
        ),
        logdog=dict(hostname='logdog.example.com'),
        experiment=dict(
            experiments=[
                dict(name=experiments.BBAGENT_DOWNLOAD_CIPD, default_value=100),
                dict(name=experiments.BBAGENT_GET_BUILD),
                dict(name=experiments.CANARY),
                dict(name=experiments.NON_PROD),
                dict(name=experiments.USE_BBAGENT),
            ],
        )
    )
    self.patch(
        'config.get_settings_async',
        autospec=True,
        return_value=future(self.settings),
    )

  def test_get_task_def(self):
    self.patch(
        'tokens.generate_build_token',
        autospec=True,
        return_value='beeff00d',
    )

    req = {
        'build_request': {
            'bucket':
                'luci.chromium.try',
            'parameters_json':
                json.dumps({
                    api_common.BUILDER_PARAMETER: 'linux',
                }),
        },
    }
    resp = self.call_api('get_task_def', req).json_body
    actual_task_def = json.loads(resp['task_definition'])
    props_def = {
        'env': [{'key': 'BUILDBUCKET_EXPERIMENTAL', 'value': 'FALSE'}],
        'env_prefixes': [{
            'key': 'PATH',
            'value': ['cipd_bin_packages', 'cipd_bin_packages/bin']
        }],
        # Concrete command line is not a concern of this test.
        'command':
            test_util.ununicode(
                actual_task_def['task_slices'][0]['properties']['command']
            ),
        'execution_timeout_secs':
            '10800',
        'grace_period_secs':
            '210',
        'cipd_input': {
            'packages': [
                {
                    'package_name': 'infra/tools/bbagent',
                    'path': '.',
                    'version': 'luci-runner-version',
                },
                {
                    'package_name': 'infra/tools/kitchen',
                    'path': '.',
                    'version': 'kitchen-version',
                },
                {
                    'package_name': 'infra/recipe_bundle',
                    'path': 'kitchen-checkout',
                    'version': 'refs/heads/master',
                },
                {
                    'package_name': 'infra/tools/git',
                    'path': swarming.USER_PACKAGE_DIR,
                    'version': 'git-version',
                },
            ],
        },
        'dimensions': [
            {'key': 'baz', 'value': 'baz'},
            {'key': 'builder', 'value': 'linux'},
            {'key': 'foo', 'value': 'bar'},
        ],
        'caches': [{
            'path': 'cache/builder',
            'name': 'builder_cache_name',
        }],
    }
    expected_task_def = {
        'name':
            'bb-1-chromium/try/linux-1',
        'realm':
            'chromium:try',
        'tags': [
            'buildbucket_bucket:chromium/try',
            'buildbucket_build_id:1',
            'buildbucket_hostname:cr-buildbucket.appspot.com',
            'buildbucket_template_canary:0',
            'builder:linux',
            'luci_project:chromium',
        ],
        'priority':
            '30',
        'task_slices': [{
            'expiration_secs': '21600',
            'properties': props_def,
            'wait_for_capacity': False,
        }],
    }
    self.assertEqual(test_util.ununicode(actual_task_def), expected_task_def)
    self.assertEqual(resp['swarming_host'], 'swarming.example.com')

  def test_get_task_def_bad_request(self):
    req = {
        'build_request': {
            'bucket':
                ')))',
            'parameters_json':
                json.dumps({
                    api_common.BUILDER_PARAMETER: 'linux',
                }),
        },
    }
    self.call_api('get_task_def', req, status=400)

  def test_get_task_def_builder_not_found(self):
    req = {
        'build_request': {
            'bucket':
                'luci.chromium.try',
            'parameters_json':
                json.dumps({
                    api_common.BUILDER_PARAMETER: 'not-existing-builder',
                }),
        },
    }
    self.call_api('get_task_def', req, status=404)

  def test_get_task_def_forbidden(self):
    req = {
        'build_id': '8982540789124571952',
        'build_request': {
            'bucket':
                'secret.bucket',
            'parameters_json':
                json.dumps({
                    api_common.BUILDER_PARAMETER: 'linux',
                }),
        },
    }

    self.call_api('get_task_def', req, status=403)

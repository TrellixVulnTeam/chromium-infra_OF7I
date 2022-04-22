# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import base64
import json
import logging

from google.appengine.ext import ndb

from components import utils
utils.fix_protobuf_package()

from components import net
from components.prpc import client, codes
from testing_utils import testing
import mock

from go.chromium.org.luci.buildbucket.proto import builds_service_pb2 as rpc_pb2
from go.chromium.org.luci.buildbucket.proto import common_pb2
from go.chromium.org.luci.buildbucket.proto import project_config_pb2
from go.chromium.org.luci.buildbucket.proto import service_config_pb2
from go.chromium.org.luci.resultdb.proto.v1 import invocation_pb2
from go.chromium.org.luci.resultdb.proto.v1 import recorder_pb2
from test.test_util import build_bundle, future, future_exception
import main
import model
import resultdb
import tq


def _make_builder_cfg():
  return project_config_pb2.BuilderConfig(
      resultdb=project_config_pb2.BuilderConfig.ResultDB(
          bq_exports=[
              invocation_pb2.BigQueryExport(
                  project='luci-resultdb',
                  dataset='chromium',
                  table='all_test_results',
              )
          ]
      )
  )


def _make_build_and_config(
    build_id, hostname='rdb.dev', invocation=None, builder_cfg=None
):
  bundle = build_bundle(
      id=build_id,
      for_creation=True,
      infra=dict(
          resultdb=dict(
              hostname=hostname or '',
              invocation=invocation or '',
          )
      )
  )
  bundle.put()
  return bundle.build, builder_cfg or _make_builder_cfg()


class ResultDBTest(testing.AppengineTestCase):

  def setUp(self):
    super(ResultDBTest, self).setUp()
    self.patch('components.net.request_async')
    self.settings = service_config_pb2.SettingsCfg(
        resultdb=dict(hostname='rdb.example.com'),
    )
    self.patch(
        'config.get_settings_async',
        autospec=True,
        return_value=future(self.settings),
    )
    self.builds_and_configs = None

    self.patch('resultdb._recorder_client')
    self.rpc_mock = (
        resultdb._recorder_client.return_value.BatchCreateInvocationsAsync
    )

    self.patch(
        'google.appengine.api.app_identity.get_default_version_hostname',
        return_value='buildbucket.example.com',
    )

  @property
  def builds(self):
    # The first item of each build_and_config tuple.
    return zip(*self.builds_and_configs)[0]

  def test_no_hostname(self):
    self.builds_and_configs = [_make_build_and_config(1, hostname=None)]
    self.settings.resultdb.hostname = ''
    resultdb.create_invocations_async(self.builds_and_configs).get_result()
    self.assertFalse(net.request_async.called)

  def test_invocation_created(self):
    builder_cfg = _make_builder_cfg()
    builder_cfg.resultdb.history_options.use_invocation_timestamp = True
    self.builds_and_configs = [
        _make_build_and_config(4, builder_cfg=builder_cfg)
    ]

    self.rpc_mock.return_value = future(
        recorder_pb2.BatchCreateInvocationsResponse(
            invocations=[
                invocation_pb2.Invocation(name='invocations/build-4'),
                invocation_pb2.Invocation(name='invocations/build-ccadafffd20293e0378d1f94d214c63a0f8342d1161454ef0acfa0405178106b-1'),  # pylint: disable=line-too-long
            ],
            update_tokens=['FakeUpdateToken1', 'FakeUpdateToken2'],
        )
    )

    resultdb.create_invocations_async(self.builds_and_configs).get_result()

    self.rpc_mock.assert_called_with(
        recorder_pb2.BatchCreateInvocationsRequest(
            request_id='build-4+0',
            requests=[
                dict(
                    invocation_id='build-4',
                    invocation=dict(
                        bigquery_exports=[
                            dict(
                                project='luci-resultdb',
                                dataset='chromium',
                                table='all_test_results',
                            )
                        ],
                        producer_resource='//buildbucket.example.com/builds/4',
                        realm='chromium:try',
                        history_options=dict(use_invocation_timestamp=True),
                    ),
                ),
                dict(
                    # echo -n 'chromium/try/linux'|openssl sha256
                    invocation_id='build-ccadafffd20293e0378d1f94d214c63a0f8342d1161454ef0acfa0405178106b-1',  # pylint: disable=line-too-long
                    invocation=dict(
                        included_invocations=['invocations/build-4'],
                        producer_resource='//buildbucket.example.com/builds/4',
                        realm='chromium:try',
                        state=invocation_pb2.Invocation.State.FINALIZING,
                    ),
                ),
            ],
        ),
        credentials=mock.ANY,
    )
    self.assertEqual(self.builds[0].resultdb_update_token, 'FakeUpdateToken1')
    self.assertEqual(
        self.builds[0].proto.infra.resultdb.invocation, 'invocations/build-4'
    )

  def test_invocation_created_without_number(self):
    builder_cfg = _make_builder_cfg()
    builder_cfg.resultdb.history_options.use_invocation_timestamp = True
    self.builds_and_configs = [
        _make_build_and_config(20, builder_cfg=builder_cfg)
    ]
    self.builds_and_configs[0][0].proto.number = 0
    self.rpc_mock.return_value = future(
        recorder_pb2.BatchCreateInvocationsResponse(
            invocations=[
                invocation_pb2.Invocation(name='invocations/build-20')
            ],
            update_tokens=['FakeUpdateToken'],
        )
    )

    resultdb.create_invocations_async(self.builds_and_configs).get_result()

    self.rpc_mock.assert_called_with(
        recorder_pb2.BatchCreateInvocationsRequest(
            request_id='build-20+0',
            requests=[
                dict(
                    invocation_id='build-20',
                    invocation=dict(
                        bigquery_exports=[
                            dict(
                                project='luci-resultdb',
                                dataset='chromium',
                                table='all_test_results',
                            )
                        ],
                        producer_resource='//buildbucket.example.com/builds/20',
                        realm='chromium:try',
                        history_options=dict(use_invocation_timestamp=True),
                    ),
                ),
            ],
        ),
        credentials=mock.ANY,
    )
    self.assertEqual(self.builds[0].resultdb_update_token, 'FakeUpdateToken')
    self.assertEqual(
        self.builds[0].proto.infra.resultdb.invocation, 'invocations/build-20'
    )

  def test_invocations_created_protobuf_update_tokens(self):
    self.builds_and_configs = [
        _make_build_and_config(15),
        _make_build_and_config(16)
    ]

    self.rpc_mock.return_value = future(
        recorder_pb2.BatchCreateInvocationsResponse(
            invocations=[
                invocation_pb2.Invocation(name='invocations/build-15'),
                invocation_pb2.Invocation(name='invocations/build-16'),
            ],
            update_tokens=['FakeUpdateToken', 'FakeUpdateToken2'],
        )
    )

    resultdb.create_invocations_async(self.builds_and_configs).get_result()
    self.assertEqual(self.builds[0].resultdb_update_token, 'FakeUpdateToken')
    self.assertEqual(
        self.builds[0].proto.infra.resultdb.invocation, 'invocations/build-15'
    )
    self.assertEqual(self.builds[1].resultdb_update_token, 'FakeUpdateToken2')
    self.assertEqual(
        self.builds[1].proto.infra.resultdb.invocation, 'invocations/build-16'
    )


class ResultDBEnqueueFinalizeTaskTest(testing.AppengineTestCase):

  def setUp(self):
    super(ResultDBEnqueueFinalizeTaskTest, self).setUp()
    self.patch('tq.enqueue_async', autospec=True, return_value=future(None))
    self.build = _make_build_and_config(1)[0]

  @ndb.transactional
  def txn(self):
    resultdb.enqueue_invocation_finalization_async(self.build)

  def test_enqueue_invocation_finalization_not_ended(self):
    self.build.proto.status = common_pb2.STARTED
    with self.assertRaises(AssertionError):
      self.txn()

  def test_enqueue_invocation_finalization(self):
    self.build.proto.status = common_pb2.SUCCESS
    self.txn()
    request = {
        'url': '/internal/task/resultdb/finalize/%d' % self.build.key.id(),
        'retry_options': {
            'task_age_limit': model.BUILD_TIMEOUT.total_seconds(),
        },
        'payload': {'id': self.build.key.id()},
    }
    tq.enqueue_async.assert_called_with('backend-default', [request])


class ResultDBFinalizeInvocationTest(testing.AppengineTestCase):

  def setUp(self):
    super(ResultDBFinalizeInvocationTest, self).setUp()
    self.metadata = {'update-token': 'FakeToken'}
    self.patch('resultdb._recorder_client')
    self.rpc_mock = resultdb._recorder_client.return_value.FinalizeInvocation

  def _create_and_finalize(self, build_id, hostname=None, invocation=None):
    bundle = build_bundle(
        id=build_id,
        infra=dict(
            resultdb=dict(
                hostname=hostname or '',
                invocation=invocation or '',
            )
        )
    )
    bundle.build.resultdb_update_token = 'FakeToken'
    bundle.put()

    return resultdb._finalize_invocation(build_id)

  @staticmethod
  def _req(name):
    return recorder_pb2.FinalizeInvocationRequest(name=name)

  @mock.patch.object(logging, 'error')
  def test_no_resultdb(self, mock_err):
    self._create_and_finalize(1)
    self.assertFalse(mock_err.called)
    self.assertFalse(self.rpc_mock.called)

  @mock.patch.object(logging, 'error')
  def test_no_invocation(self, mock_err):
    self._create_and_finalize(2, 'rdb.com')
    self.assertFalse(mock_err.called)
    self.assertFalse(self.rpc_mock.called)

  @mock.patch.object(logging, 'error')
  def test_no_permission(self, mock_err):
    self.rpc_mock.side_effect = client.RpcError(
        'Permission Denied', codes.StatusCode.PERMISSION_DENIED, {}
    )
    self._create_and_finalize(3, 'rdb.dev', 'invocations/build-3')
    self.rpc_mock.assert_called_with(
        self._req('invocations/build-3'),
        credentials=mock.ANY,
        metadata=self.metadata,
    )
    self.assertTrue(mock_err.called)

  @mock.patch.object(logging, 'error')
  def test_failed_precondition(self, mock_err):
    self.rpc_mock.side_effect = client.RpcError(
        'Failed Precondition', codes.StatusCode.FAILED_PRECONDITION, {}
    )
    self._create_and_finalize(4, 'rdb.dev', 'invocations/build-4')
    self.assertTrue(mock_err.called)

  @mock.patch.object(logging, 'error')
  def test_success(self, mock_err):
    self._create_and_finalize(5, 'rdb.dev', 'invocations/build-5')
    self.rpc_mock.assert_called_with(
        self._req('invocations/build-5'),
        credentials=mock.ANY,
        metadata=self.metadata,
    )
    self.assertFalse(mock_err.called)

  def test_transient_fail(self):
    self.rpc_mock.side_effect = client.RpcError(
        'Unavailable', codes.StatusCode.UNAVAILABLE, {}
    )
    with self.assertRaises(client.RpcError):  # Causes retry.
      self._create_and_finalize(6, 'rdb.dev', 'invocations/build-6')
    self.rpc_mock.assert_called_with(
        self._req('invocations/build-6'),
        credentials=mock.ANY,
        metadata=self.metadata,
    )

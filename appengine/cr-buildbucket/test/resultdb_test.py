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

from go.chromium.org.luci.buildbucket.proto import common_pb2
from go.chromium.org.luci.buildbucket.proto import service_config_pb2
from go.chromium.org.luci.resultdb.proto.rpc.v1 import invocation_pb2
from go.chromium.org.luci.resultdb.proto.rpc.v1 import recorder_pb2
from test.test_util import build_bundle, future, future_exception
import main
import model
import resultdb
import tq


def _make_build(build_id, hostname='rdb.dev', invocation=None):
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
  return bundle.build


def _mock_create_request_async(
    response, metadata_update_tokens=None, pb_update_tokens=None
):

  def inner(*_, **kwargs):
    if metadata_update_tokens:
      kwargs['response_headers']['update-token'] = ','.join(
          metadata_update_tokens
      )
    else:
      response.update_tokens.extend(pb_update_tokens)
    ret = future(response.SerializeToString())
    return ret

  return inner


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
    self.builds = None

  def test_no_hostname(self):
    self.builds = [_make_build(1, hostname=None)]
    self.settings.resultdb.hostname = ''
    resultdb.create_invocations_async(self.builds).get_result()
    self.assertFalse(net.request_async.called)

  def test_cannot_create_invocation(self):
    self.builds = [_make_build(3)]
    net.request_async.side_effect = [
        future_exception(
            net.Error(
                'Internal Error',
                500,
                'Internal Error',
                headers={'X-Prpc-Grpc-Code': '2'}
            )
        )
    ]
    with self.assertRaises(client.RpcError):
      resultdb.create_invocations_async(self.builds).get_result()

  def test_invocation_created(self):
    self.builds = [_make_build(4)]
    response = recorder_pb2.BatchCreateInvocationsResponse(
        invocations=[invocation_pb2.Invocation(name='invocations/build:4')]
    )
    net.request_async.side_effect = _mock_create_request_async(
        response, ['FakeUpdateToken']
    )
    resultdb.create_invocations_async(self.builds).get_result()
    self.assertEqual(self.builds[0].resultdb_update_token, 'FakeUpdateToken')
    self.assertEqual(
        self.builds[0].proto.infra.resultdb.invocation, 'invocations/build:4'
    )

  def test_invocations_created_metadata_update_tokens(self):
    self.builds = [_make_build(5), _make_build(6)]
    response = recorder_pb2.BatchCreateInvocationsResponse(
        invocations=[
            invocation_pb2.Invocation(name='invocations/build:5'),
            invocation_pb2.Invocation(name='invocations/build:6'),
        ]
    )
    net.request_async.side_effect = _mock_create_request_async(
        response,
        metadata_update_tokens=['FakeUpdateToken', 'FakeUpdateToken2']
    )
    resultdb.create_invocations_async(self.builds).get_result()
    self.assertEqual(self.builds[0].resultdb_update_token, 'FakeUpdateToken')
    self.assertEqual(
        self.builds[0].proto.infra.resultdb.invocation, 'invocations/build:5'
    )
    self.assertEqual(self.builds[1].resultdb_update_token, 'FakeUpdateToken2')
    self.assertEqual(
        self.builds[1].proto.infra.resultdb.invocation, 'invocations/build:6'
    )

  def test_invocations_created_protobuf_update_tokens(self):
    self.builds = [_make_build(15), _make_build(16)]
    response = recorder_pb2.BatchCreateInvocationsResponse(
        invocations=[
            invocation_pb2.Invocation(name='invocations/build:15'),
            invocation_pb2.Invocation(name='invocations/build:16'),
        ]
    )
    net.request_async.side_effect = _mock_create_request_async(
        response, pb_update_tokens=['FakeUpdateToken', 'FakeUpdateToken2']
    )
    resultdb.create_invocations_async(self.builds).get_result()
    self.assertEqual(self.builds[0].resultdb_update_token, 'FakeUpdateToken')
    self.assertEqual(
        self.builds[0].proto.infra.resultdb.invocation, 'invocations/build:15'
    )
    self.assertEqual(self.builds[1].resultdb_update_token, 'FakeUpdateToken2')
    self.assertEqual(
        self.builds[1].proto.infra.resultdb.invocation, 'invocations/build:16'
    )


class ResultDBEnqueueFinalizeTaskTest(testing.AppengineTestCase):

  def setUp(self):
    super(ResultDBEnqueueFinalizeTaskTest, self).setUp()
    self.patch('tq.enqueue_async', autospec=True, return_value=future(None))
    self.build = _make_build(1)

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
    self.patch('resultdb._call_finalize_rpc')
    self.metadata = {'update-token': 'FakeToken'}

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
    return recorder_pb2.FinalizeInvocationRequest(name=name, interrupted=False)

  @mock.patch.object(logging, 'error')
  def test_no_resultdb(self, mock_err):
    self._create_and_finalize(1)
    self.assertFalse(mock_err.called)
    self.assertFalse(resultdb._call_finalize_rpc.called)

  @mock.patch.object(logging, 'error')
  def test_no_invocation(self, mock_err):
    self._create_and_finalize(2, 'rdb.com')
    self.assertTrue(mock_err.called)
    self.assertFalse(resultdb._call_finalize_rpc.called)

  @mock.patch.object(logging, 'error')
  def test_no_permission(self, mock_err):
    resultdb._call_finalize_rpc.side_effect = client.RpcError(
        'Permission Denied', codes.StatusCode.PERMISSION_DENIED, {}
    )
    self._create_and_finalize(3, 'rdb.dev', 'invocations/build:3')
    resultdb._call_finalize_rpc.assert_called_with(
        'rdb.dev', self._req('invocations/build:3'), self.metadata
    )
    self.assertTrue(mock_err.called)

  @mock.patch.object(logging, 'error')
  def test_failed_precondition(self, mock_err):
    resultdb._call_finalize_rpc.side_effect = client.RpcError(
        'Failed Precondition', codes.StatusCode.FAILED_PRECONDITION, {}
    )
    self._create_and_finalize(4, 'rdb.dev', 'invocations/build:4')
    self.assertTrue(mock_err.called)

  @mock.patch.object(logging, 'error')
  def test_success(self, mock_err):
    self._create_and_finalize(5, 'rdb.dev', 'invocations/build:5')
    resultdb._call_finalize_rpc.assert_called_with(
        'rdb.dev', self._req('invocations/build:5'), self.metadata
    )
    self.assertFalse(mock_err.called)

  def test_transient_fail(self):
    resultdb._call_finalize_rpc.side_effect = client.RpcError(
        'Unavailable', codes.StatusCode.UNAVAILABLE, {}
    )
    with self.assertRaises(client.RpcError):  # Causes retry.
      self._create_and_finalize(6, 'rdb.dev', 'invocations/build:6')
    resultdb._call_finalize_rpc.assert_called_with(
        'rdb.dev', self._req('invocations/build:6'), self.metadata
    )

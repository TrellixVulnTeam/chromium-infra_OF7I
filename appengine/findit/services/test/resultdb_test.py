# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock
from services import resultdb
from waterfall.test import wf_testcase
from go.chromium.org.luci.resultdb.proto.v1 import (artifact_pb2, predicate_pb2,
                                                    resultdb_pb2)
from components.prpc import client as prpc_client


class ResultDBTest(wf_testcase.WaterfallTestCase):

  @mock.patch.object(prpc_client, 'service_account_credentials')
  @mock.patch('components.prpc.client.Client')
  def test_query_resultdb_with_pagination(self, mock_client, _):
    side_effect = [
        mock.MagicMock(
            next_page_token="123",
            test_results=[mock.MagicMock(), mock.MagicMock()],
        ),
        mock.MagicMock(
            next_page_token="456",
            test_results=[mock.MagicMock(), mock.MagicMock()],
        ),
        mock.MagicMock(
            next_page_token="",
            test_results=[mock.MagicMock()],
        ),
    ]
    mock_client.return_value.QueryTestResults.side_effect = side_effect
    results = resultdb.query_resultdb("my_inv_name")
    self.assertEqual(len(results), 5)

  def test_resultdb_req(self):
    req = resultdb.resultdb_req("inv_name", True)
    self.assertEqual(
        req.predicate.expectancy, predicate_pb2.TestResultPredicate.Expectancy
        .VARIANTS_WITH_UNEXPECTED_RESULTS)
    req = resultdb.resultdb_req("inv_name", False)
    self.assertEqual(req.predicate.expectancy,
                     predicate_pb2.TestResultPredicate.Expectancy.ALL)

  @mock.patch.object(prpc_client, 'service_account_credentials')
  @mock.patch('components.prpc.client.Client')
  def test_list_artifacts(self, mock_client, *_):
    # pylint: disable=line-too-long
    mock_client.return_value.ListArtifacts.return_value = resultdb_pb2.ListArtifactsResponse(
        artifacts=[
            artifact_pb2.Artifact(artifact_id="stack_trace"),
            artifact_pb2.Artifact(artifact_id="stack_trace1"),
        ])
    results = resultdb.list_artifacts("test_result_name")
    self.assertEqual(len(results), 2)

# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock
from services import resultdb
from waterfall.test import wf_testcase
from components.prpc import client as prpc_client


class ResultDBTest(wf_testcase.WaterfallTestCase):

  @mock.patch.object(prpc_client, 'service_account_credentials')
  @mock.patch('components.prpc.client.Client')
  def testQueryResultDBWithPagination(self, mock_client, _):
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
    results = resultdb.query_resultdb("my_inv_id")
    self.assertEqual(len(results), 5)

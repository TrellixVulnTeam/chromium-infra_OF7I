# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""This module is for communicating with ResultDB."""
import logging
import os
import sys

from components.prpc import client as prpc_client
from go.chromium.org.luci.resultdb.proto.v1 import resultdb_pb2
from go.chromium.org.luci.resultdb.proto.v1 import resultdb_prpc_pb2
from google.protobuf.field_mask_pb2 import FieldMask

RESULTDB_HOSTNAME = "results.api.cr.dev"

def query_resultdb(inv_id):
  logging.info("Query test results for invocation %s from resultdb", inv_id)
  client = resultdb_client(RESULTDB_HOSTNAME)
  inv_name = "invocations/" + inv_id
  next_page_token = None
  results = []
  while True:
    req = resultdb_pb2.QueryTestResultsRequest(
        invocations=[inv_name],
        read_mask=FieldMask(paths=["*"]),
        page_size=1000,
    )
    if next_page_token is not None:
      req.page_token = next_page_token
    resp = client.QueryTestResults(
        req,
        credentials=prpc_client.service_account_credentials(),
    )
    next_page_token = resp.next_page_token
    results.extend(resp.test_results)
    if next_page_token is None or next_page_token == "":
      break
  logging.info("Got %d test results", len(results))
  return results


def resultdb_client(hostname):
  return prpc_client.Client(hostname,
                            resultdb_prpc_pb2.ResultDBServiceDescription)

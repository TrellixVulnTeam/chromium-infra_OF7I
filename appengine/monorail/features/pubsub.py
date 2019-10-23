# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Task handlers for publishing issue updates onto a pub/sub topic.

The pub/sub topic name is: `projects/{project-id}/topics/issue-updates`.
"""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import httplib2
import logging
import sys

import settings

from googleapiclient.discovery import build
from apiclient.errors import Error as ApiClientError
from oauth2client.client import GoogleCredentials
from oauth2client.client import Error as Oauth2ClientError

from framework import jsonfeed


class PublishPubsubIssueChangeTask(jsonfeed.InternalTask):
  """JSON servlet that pushes issue update messages onto a pub/sub topic."""

  def HandleRequest(self, mr):
    """Push a message onto a pub/sub queue.

    Args:
      mr: common information parsed from the HTTP request.
    Returns:
      A dictionary. If an error occurred, the 'error' field will be a string
      containing the error message.
    """
    pubsub_client = set_up_pubsub_api()
    if not pubsub_client:
      return {
        'error': 'Pub/Sub API init failure.',
      }

    issue_id = mr.GetPositiveIntParam('issue_id')
    if not issue_id:
      return {
        'error': 'Cannot proceed without a valid issue ID.',
      }

    pubsub_client.projects().topics().publish(
        topic=settings.pubsub_topic_id,
        body={
          'messages': [{
            'attributes': {
              'issue_id': str(issue_id),
            },
          }],
        },
      ).execute()

    return {}


def set_up_pubsub_api():
  """Attempts to build and return a pub/sub API client."""
  try:
    return build('pubsub', 'v1', http=httplib2.Http(),
        credentials=GoogleCredentials.get_application_default())
  except (Oauth2ClientError, ApiClientError):
    logging.error("Error setting up Pub/Sub API: %s" % sys.exc_info()[0])
    return None

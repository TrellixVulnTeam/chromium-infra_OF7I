# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

#!/usr/bin/env python
"""
This script requires `google-auth` 1.15.0 or higher.
To update this for monorail's third_party run the following from
monorail/third_party/google:
bash ./update.sh 1.15.0

This is an example of how a script might make calls to monorail's v3  pRPC API.

Usage example:
```
python v3_test_call.py \
monorail.v3.Issues GetIssue '{"name": "projects/monorail/issues/404"}'
```

The email of your service account should be allow-listed with Monorail.
"""

import argparse
import json
import logging
import os
import sys
import requests

monorail_dir = os.path.dirname(os.path.abspath(__file__ + '/..'))
third_party_path = os.path.join(monorail_dir, 'third_party')
if third_party_path not in sys.path:
  sys.path.insert(0, third_party_path)

# Older versions of https://github.com/googleapis/google-auth-library-python
# do not have the fetch_id_token() method called below.
# v1.15.0 or later should be fine.
from google.oauth2 import id_token
from google.auth.transport import requests as google_requests

# Download and save your service account credentials file in
# api/service-account-key.json.
# id_token.fetch_id_token looks inside GOOGLE_APPLICATION_CREDENTIALS to fetch
# service account credentials.
os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = 'service-account-key.json'

# BASE_URL can point to any monorail-dev api service version.
# However, you MAY get ssl cert errors when BASE_URL is not the
# default below. If this happens you will have to test your version
# by using the existing BASE_URL and migrating all traffic to your api
# version via pantheon.
BASE_URL = 'https://api-dot-monorail-dev.appspot.com/prpc'

# TARGET_AUDIENCE should not change as long as BASE_URL is pointing to
# some monorail-dev version. If BASE_URL is updated to point to
# monorail-{staging|prod}, update TARGET_AUDIENCE accordingly.
TARGET_AUDIENCE = 'https://monorail-dev.appspot.com'

# XSSI_PREFIX found at the beginning of every prpc response.
XSSI_PREFIX = ")]}'\n"

import httplib2
from oauth2client.client import GoogleCredentials


def make_call(service, method, json_body):
  # Fetch ID token
  request = google_requests.Request()
  token = id_token.fetch_id_token(request, TARGET_AUDIENCE)
  # Note: ID tokens for service accounts can also be fetched with with the
  # Cloud IAM API projects.serviceAccounts.generateIdToken
  # generateIdToken only needs the service account email or ID and the
  # target_audience.

  # Call monorail's API.
  headers = {
      'Authorization': 'Bearer %s' % token,
      'Content-Type': 'application/json',
      'Accept': 'application/json',
  }

  url = "%s/%s/%s" % (BASE_URL, service, method)

  body = json.loads(json_body)
  resp = requests.post(url, data=json.dumps(body), headers=headers)
  logging.info(resp)
  logging.info(resp.text)
  logging.info(resp.content)
  logging.info(json.dumps(json.loads(resp.content[len(XSSI_PREFIX):])))

  # Verify and decode ID token to take a look at what's inside.
  # API users should not have to do this. This is just for learning about
  # how ID tokens work.
  request = google_requests.Request()
  id_info = id_token.verify_oauth2_token(token, request)
  logging.info('id_info %s' % id_info)


if __name__ == '__main__':
  parser = argparse.ArgumentParser(description='Process some integers.')
  parser.add_argument('service', help='pRPC service name.')
  parser.add_argument('method', help='pRPC method name.')
  parser.add_argument('json_body', help='pRPC HTTP body in valid JSON.')
  args = parser.parse_args()
  log_level = logging.INFO
  logging.basicConfig(format='%(levelname)s: %(message)s', level=log_level)
  make_call(args.service, args.method, args.json_body)

#!/usr/bin/env vpython3
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This test is only run under python3, but this is needed to keep
# pylint 1.5 happy.
from __future__ import print_function

import sys
import requests


TEST_CASES = [
    # OK.
    ('https://clients3.google.com/generate_204', 204),
    # ('https://extended-validation.badssl.com', 200),

    # Bad certs.
    # ('https://expired.badssl.com', requests.exceptions.SSLError),
    # ('https://wrong.host.badssl.com', requests.exceptions.SSLError),
    # ('https://self-signed.badssl.com', requests.exceptions.SSLError),
    # ('https://untrusted-root.badssl.com', requests.exceptions.SSLError),

    # 'requests' is known to accept revoked certificates.
    # https://github.com/kennethreitz/requests/issues/3770
    # ('https://revoked.badssl.com', requests.exceptions.SSLError),
]


def get_code_or_err(url):
  try:
    print('Trying %s' % url)
    return requests.get(url).status_code
  except requests.exceptions.SSLError as exc:
    return exc


def tests_succeed():
  ok = True
  for url, exp in TEST_CASES:
    res = get_code_or_err(url)
    if isinstance(exp, int):
      if exp != res:
        print('For %s: expecting %d, got %s' % (url, exp, res), file=sys.stderr)
        ok = False
    elif not isinstance(res, exp):
      print('For %s: expecting %s, got %s' % (url, exp, res), file=sys.stderr)
      ok = False
  return ok


sys.exit(0 if tests_succeed() else 1)

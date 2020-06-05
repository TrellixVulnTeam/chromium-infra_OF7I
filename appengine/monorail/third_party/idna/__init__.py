# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Emulate the bare minimum for idna for Monorail.
# In practice, we do not need it, and it's very large.
# See https://pypi.org/project/idna/

from encodings import idna


class IDNAError(Exception):
  # Referred to by requests/models.py
  pass


class core(object):
  class IDNAError(Exception):
    # Referred to by urllib3/contrib/pyopenssl.py
    pass


def encode(host, uts46=False):  # pylint: disable=unused-argument
  # Used by urllib3
  return idna.ToASCII(host)


def decode(host):
  # Used by cryptography/hazmat/backends/openssl/x509.py
  return idna.ToUnicode(host)

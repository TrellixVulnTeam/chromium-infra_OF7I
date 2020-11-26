#!/usr/bin/env python
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import json
import ssl
import os
import sys
import urllib

import certifi

# Make sure up-to-date root certificates are used.
urllib._urlopener = urllib.FancyURLopener(
    context=ssl.create_default_context(cafile=certifi.where()))


def do_latest():
  print json.load(urllib.urlopen(
      'https://registry.npmjs.org/firebase-tools'))['dist-tags']['latest']


def get_download_url(version, platform):
  ext = ''
  if platform.startswith('windows-'):
    # pylint: disable=line-too-long
    url = 'https://github.com/firebase/firebase-tools/releases/download/v%(ver)s/firebase-tools-instant-win.exe' % {
        'ver': version
    }
    ext = '.exe'
  elif platform.startswith('mac-'):
    # pylint: disable=line-too-long
    url = 'https://github.com/firebase/firebase-tools/releases/download/v%(ver)s/firebase-tools-macos' % {
        'ver': version
    }
  elif platform.startswith('linux-'):
    # pylint: disable=line-too-long
    url = 'https://github.com/firebase/firebase-tools/releases/download/v%(ver)s/firebase-tools-linux' % {
        'ver': version
    }
  else:
    raise ValueError('fetch.py is only supported for amd64 hosts.')
  partial_manifest = {
    'url': [url],
    'ext': ext,
  }
  print(json.dumps(partial_manifest))


def main():
  ap = argparse.ArgumentParser()
  sub = ap.add_subparsers()

  latest = sub.add_parser("latest")
  latest.set_defaults(func=lambda _opts: do_latest())

  download = sub.add_parser("get_url")
  download.set_defaults(func=lambda opts: get_download_url(
      os.environ['_3PP_VERSION'], os.environ['_3PP_PLATFORM']))

  opts = ap.parse_args()
  return opts.func(opts)


if __name__ == '__main__':
  sys.exit(main())

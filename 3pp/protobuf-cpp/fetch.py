#!/usr/bin/env python
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import json
import os
import sys
import urllib

# https://developer.github.com/v3/repos/releases/#get-the-latest-release
# Returns a JSON-loadable text response like:
# {
#   ...,
#   "assets": [
#     {
#       ...,
#       "browser_download_url": "...",
#       ...,
#       "name": "protobuf-cpp-3.17.3.tar.gz",
#       ...,
#     },
#     ...
#   ],
#   ...
#   "tag_name": "v3.17.3",
#   ...
# }
#
# Of interest are tag_name, which contains the version, and assets, which
# details platform-specific binaries. Under assets, name indicates the platform
# and browser_download_url indicates where to download a zip file containing the
# prebuilt binary.
LATEST = 'https://api.github.com/repos/protocolbuffers/protobuf/releases/latest'

# https://developer.github.com/v3/repos/releases/#get-a-release-by-tag-name
# Returns a JSON loadable text response like LATEST, but for a specific tag.
TAGGED_RELEASE = (
    'https://api.github.com/repos/protocolbuffers/protobuf/releases/tags/v%s')


def do_latest():
  print json.load(
      urllib.urlopen(LATEST))['tag_name'][1:]  # e.g. v3.8.0 -> 3.8.0


def get_download_url(version):
  name = 'protobuf-cpp-%s.tar.gz' % version

  rsp = json.load(urllib.urlopen(TAGGED_RELEASE % version))
  actual_tag = rsp['tag_name'][1:]
  if version != actual_tag:
    raise ValueError('expected %s, actual is %s' % (version, actual_tag))

  for a in rsp['assets']:
    if a['name'] == name:
      partial_manifest = {
          'url': [a['browser_download_url']],
          'ext': '.tar.gz',
      }
      print(json.dumps(partial_manifest))
      return
  raise ValueError('missing release for protobuf-cpp')


def main():
  ap = argparse.ArgumentParser()
  sub = ap.add_subparsers()

  latest = sub.add_parser("latest")
  latest.set_defaults(func=lambda _opts: do_latest())

  download = sub.add_parser("get_url")
  download.set_defaults(
      func=lambda opts: get_download_url(os.environ['_3PP_VERSION']))

  opts = ap.parse_args()
  return opts.func(opts)


if __name__ == '__main__':
  sys.exit(main())

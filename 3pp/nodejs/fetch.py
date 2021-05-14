#!/usr/bin/env python
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import json
import os
import ssl
import sys
import urllib

from pkg_resources import parse_version
import certifi

BASE_URL = 'https://nodejs.org/dist/'

# Make sure up-to-date root certificates are used.
urllib._urlopener = urllib.FancyURLopener(
    context=ssl.create_default_context(cafile=certifi.where()))


def do_latest():
  data = json.load(urllib.urlopen(BASE_URL + 'index.json'))
  max_version, max_string = parse_version('0'), '0'
  for release in data:
    s = release['version'].lstrip('v')
    v = parse_version(s)
    if v > max_version:
      max_version = v
      max_string = s

  print str(max_string)


def get_download_url(version, platform):
  targ_os, targ_arch = platform.split('-')
  ext = '.zip' if targ_os == 'windows' else '.tar.gz'
  fragment = {
      ('mac', 'amd64'): 'darwin-x64',
      ('mac', 'arm64'): 'darwin-arm64',
      ('linux', 'amd64'): 'linux-x64',
      ('linux', 'armv6l'): 'linux-armv6l',
      ('linux', 'arm64'): 'linux-arm64',
      ('windows', 'amd64'): 'win-x64',
  }[(targ_os, targ_arch)]
  download_url = (
    '%(base)s/v%(version)s/node-v%(version)s-%(fragment)s%(ext)s'
    % {
      'base': BASE_URL,
      'version': version,
      'fragment': fragment,
      'ext': ext
    })
  partial_manifest = {
    'url': [download_url],
    'ext': ext,
  }
  print(json.dumps(partial_manifest))


def main():
  ap = argparse.ArgumentParser()
  sub = ap.add_subparsers()

  latest = sub.add_parser("latest")
  latest.set_defaults(func=lambda _opts: do_latest())

  download = sub.add_parser("get_url")
  download.set_defaults(
    func=lambda opts: get_download_url(
      os.environ['_3PP_VERSION'], os.environ['_3PP_PLATFORM']))

  opts = ap.parse_args()
  return opts.func(opts)


if __name__ == '__main__':
  sys.exit(main())

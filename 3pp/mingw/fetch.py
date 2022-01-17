#!/usr/bin/env python
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from __future__ import print_function

import argparse
import json
import os


def do_latest():
  print('11.2.0-6')


def get_download_url():
  # These were found following the dependency chain in
  # https://packages.msys2.org/package/mingw-w64-x86_64-gcc?repo=mingw64.
  # Similarly, these could be found by running pacman -S mingw-w64-x86_64-gcc
  # and reading the list of packages being installed.
  packages = [
      "binutils-2.37-4-any.pkg.tar.zst",
      "crt-git-9.0.0.6373.5be8fcd83-1-any.pkg.tar.zst",
      "gcc-11.2.0-6-any.pkg.tar.zst",
      "gcc-libs-11.2.0-6-any.pkg.tar.zst",
      "gmp-6.2.1-3-any.pkg.tar.zst",
      "headers-git-9.0.0.6373.5be8fcd83-1-any.pkg.tar.zst",
      "isl-0.24-1-any.pkg.tar.zst",
      "libwinpthread-git-9.0.0.6373.5be8fcd83-1-any.pkg.tar.zst",
      "libiconv-1.16-2-any.pkg.tar.zst",
      "mpc-1.2.1-1-any.pkg.tar.zst",
      "mpfr-4.1.0.p13-1-any.pkg.tar.zst",
      "zlib-1.2.11-9-any.pkg.tar.zst",
      "zstd-1.5.1-1-any.pkg.tar.zst",
      "windows-default-manifest-6.4-3-any.pkg.tar.xz",
      "winpthreads-git-9.0.0.6373.5be8fcd83-1-any.pkg.tar.zst",
  ]
  url_prefix = "https://repo.msys2.org/mingw/mingw64/mingw-w64-x86_64-"
  urls = [url_prefix + p for p in packages]

  partial_manifest = {
      'url': urls,
      'name': packages,
      'ext': '.tar.zst',
  }
  print(json.dumps(partial_manifest))


def main():
  ap = argparse.ArgumentParser()
  sub = ap.add_subparsers()

  latest = sub.add_parser("latest")
  latest.set_defaults(func=lambda _opts: do_latest())

  download = sub.add_parser("get_url")
  download.set_defaults(func=lambda _opts: get_download_url())

  opts = ap.parse_args()
  opts.func(opts)


if __name__ == '__main__':
  main()

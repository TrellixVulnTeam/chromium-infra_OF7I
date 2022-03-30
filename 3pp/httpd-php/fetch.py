#!/usr/bin/env python
# Copyright 2022 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import json
import os

apr_version="1.6.5"
apr_util_version="1.6.1"
expat_version="2.4.7"
httpd_version="2.4.38"
libxml2_version="2.9.12"
openssl_version="1.1.1j"
pcre_version="8.41"
php_version="7.3.31"
zlib_version="1.2.12"

def do_latest():
  print('httpd{0}.php{1}.chromium.3'.format(httpd_version, php_version))


def get_download_url():
  urls = [
      "https://archive.apache.org/dist/apr/"
      "apr-{}.tar.gz".format(apr_version),
      "https://archive.apache.org/dist/apr/"
      "apr-util-{}.tar.gz".format(apr_util_version),
      "https://github.com/libexpat/libexpat/releases/download/R_{1}_{2}_{3}/"
      "expat-{0}.tar.gz".format(expat_version, *expat_version.split(".")),
      "https://archive.apache.org/dist/httpd/"
      "httpd-{}.tar.gz".format(httpd_version),
      "http://xmlsoft.org/download/"
      "libxml2-{}.tar.gz".format(libxml2_version),
      "https://www.openssl.org/source/"
      "openssl-{}.tar.gz".format(openssl_version),
      "https://sourceforge.net/projects/pcre/files/pcre/"
      "{0}/pcre-{0}.tar.gz/download".format(pcre_version),
      "https://secure.php.net/distributions/"
      "php-{}.tar.gz".format(php_version),
      "https://www.zlib.net/zlib-{}.tar.gz".format(zlib_version),
  ]

  packages = [
      "apr-{}.tar.gz".format(apr_version),
      "apr-util-{}.tar.gz".format(apr_util_version),
      "expat-{}.tar.gz".format(expat_version),
      "httpd-{}.tar.gz".format(httpd_version),
      "libxml2-{}.tar.gz".format(libxml2_version),
      "openssl-{}.tar.gz".format(openssl_version),
      "pcre-{}.tar.gz".format(pcre_version),
      "php-{}.tar.gz".format(php_version),
      "zlib-{}.tar.gz".format(zlib_version),
  ]

  partial_manifest = {
      'url': urls,
      'name': packages,
      'ext': '.tar.gz',
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

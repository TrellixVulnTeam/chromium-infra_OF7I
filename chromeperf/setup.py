# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
from setuptools import setup, find_packages

setup(
    name='chromeperf',
    packages=find_packages('src'),
    package_dir={'': 'src'},
    setup_requires=['pytest-runner'],
)

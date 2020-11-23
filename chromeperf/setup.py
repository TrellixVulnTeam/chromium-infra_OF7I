# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
from setuptools import setup, find_packages

setup(
    name='chromeperf',
    packages=find_packages('src'),
    package_dir={'': 'src'},
    install_requires=[
        'attrs>=20.1.0',
        'apache-beam[gcp]>=2.25.0'
        'cachetools>=4.1.1',
        'certifi>=2020.6.20',
        'chardet>=3.0.4',
        'flask>=1.1.2',
        'google-api-core>=1.22.1',
        'google-api-python-client>=1.11.0',
        'google-auth-httplib2>=0.0.4',
        'google-auth>=1.21.0',
        'google-cloud-core>=1.4.1',
        'google-cloud-datastore>=1.5.0',
        'google-python-cloud-debugger>=2.15',
        'googleapis-common-protos>=1.52.0',
        'grpcio>=1.31.0',
        'httplib2<0.18.0',
        'idna>=2.10',
        'iniconfig>=1.0.1',
        'more-itertools>=8.5.0',
        'protobuf>=3.13.0',
        'py>=1.9.0',
        'pyasn1-modules>=0.2.8',
        'pyasn1>=0.4.8',
        'pyparsing>=2.4.7',
        'pytz>=2020.1',
        'redis>=3.5.3',
        'requests>=2.24.0',
        'rsa>=4.6',
        'scipy>=1.5.4',
        'six>=1.15.0',
        'toml>=0.10.1',
        'uritemplate>=3.0.1',
        'urllib3>=1.25.10',
    ],
)

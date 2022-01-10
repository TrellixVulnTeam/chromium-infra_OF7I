# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is govered by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

runtime: python27
api_version: 1
threadsafe: no

default_expiration: "10d"

define(`_VERSION', `syscmd(`echo $_VERSION')')

ifdef(`PROD', `
instance_class: F4
automatic_scaling:
  min_idle_instances: 25
  max_pending_latency: 0.2s
')

ifdef(`STAGING', `
instance_class: F4
automatic_scaling:
  min_idle_instances: 1
  max_pending_latency: 0.2s
')

ifdef(`DEV', `
instance_class: F4
automatic_scaling:
  min_idle_instances: 1
')

handlers:
- url: /_ah/api/.*
  script: monorailapp.endpoints

- url: /robots.txt
  static_files: static/robots.txt
  upload: static/robots.txt
  expiration: "10m"

- url: /database-maintenance
  static_files: static/database-maintenance.html
  upload: static/database-maintenance.html

- url: /static/dist
  static_dir: static/dist
  mime_type: application/javascript
  secure: always
  http_headers:
    Access-Control-Allow-Origin: '*'

- url: /static/js
  static_dir: static/js
  mime_type: application/javascript
  secure: always
  http_headers:
    Access-Control-Allow-Origin: '*'

- url: /static
  static_dir: static

- url: /_ah/mail/.+
  script: monorailapp.app
  login: admin

- url: /_ah/warmup
  script: monorailapp.app
  login: admin

- url: /.*
  script: monorailapp.app
  secure: always

inbound_services:
- mail
- mail_bounce
ifdef(`PROD', `
- warmup
')
ifdef(`STAGING', `
- warmup
')

libraries:
- name: endpoints
  version: 1.0
- name: grpcio
  version: 1.0.0
- name: MySQLdb
  version: "latest"
- name: ssl # needed for google.auth.transport and GAE_USE_SOCKETS_HTTPLIB
  version: "2.7.11"

includes:
- gae_ts_mon

env_variables:
  VERSION_ID: '_VERSION'
  GAE_USE_SOCKETS_HTTPLIB : ''

vpc_access_connector:
ifdef(`DEV',`
  name: "projects/monorail-dev/locations/us-central1/connectors/redis-connector"
')
ifdef(`STAGING',`
  name: "projects/monorail-staging/locations/us-central1/connectors/redis-connector"
')
ifdef(`PROD', `
  name: "projects/monorail-prod/locations/us-central1/connectors/redis-connector"
')

skip_files:
- ^(.*/)?#.*#$
- ^(.*/)?.*~$
- ^(.*/)?.*\.py[co]$
- ^(.*/)?.*/RCS/.*$
- ^(.*/)?\..*$
- node_modules/
- venv/

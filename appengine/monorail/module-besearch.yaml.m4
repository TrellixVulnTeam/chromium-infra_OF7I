# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is govered by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

define(`_VERSION', `syscmd(`echo $_VERSION')')

service: besearch
runtime: python27
api_version: 1
threadsafe: no

ifdef(`PROD', `
instance_class: F4
automatic_scaling:
  min_idle_instances: 40
  max_pending_latency: 0.2s
')

ifdef(`STAGING', `
instance_class: F4
automatic_scaling:
  min_idle_instances: 5
  max_pending_latency: 0.2s
')

ifdef(`DEV', `
instance_class: F4
automatic_scaling:
  min_idle_instances: 5
')

handlers:
- url: /_ah/warmup
  script: monorailapp.app
  login: admin

- url: /_backend/.*
  script: monorailapp.app

- url: /_ah/start
  script: monorailapp.app
  login: admin

- url: /_ah/stop
  script: monorailapp.app
  login: admin

ifdef(`PROD', `
inbound_services:
- warmup
')
ifdef(`STAGING', `
inbound_services:
- warmup
')

libraries:
- name: endpoints
  version: 1.0
- name: grpcio
  version: 1.0.0
- name: MySQLdb
  version: "latest"
- name: ssl
  version: latest

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

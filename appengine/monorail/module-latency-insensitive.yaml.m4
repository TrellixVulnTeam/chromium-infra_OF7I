# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is govered by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

define(`_VERSION', `syscmd(`echo $_VERSION')')

service: latency-insensitive
runtime: python27
api_version: 1
threadsafe: no

default_expiration: "3600d"

ifdef(`PROD', `
instance_class: F4
automatic_scaling:
  min_idle_instances: 5
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
  min_idle_instances: 1
')

handlers:
- url: /_ah/warmup
  script: monorailapp.app
  login: admin

- url: /_ah/api/.*
  script: monorailapp.endpoints

- url: /_task/.*
  script: monorailapp.app
  login: admin

- url: /_cron/.*
  script: monorailapp.app
  login: admin

- url: /_ah/mail/.*
  script: monorailapp.app
  login: admin

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
- name: ssl
  version: latest

includes:
- gae_ts_mon

env_variables:
  VERSION_ID: '_VERSION'
  GAE_USE_SOCKETS_HTTPLIB: ''

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

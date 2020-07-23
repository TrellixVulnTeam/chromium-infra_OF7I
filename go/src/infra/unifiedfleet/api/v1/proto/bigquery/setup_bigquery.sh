#!/bin/sh
# Copyright 2020 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

# Reference: http://google3/third_party/luci_py/latest/appengine/swarming/setup_bigquery.sh

set -eu

cd "$(dirname $0)"

if ! (which bq) > /dev/null; then
  echo "Please install 'bq' from gcloud SDK"
  echo "  https://cloud.google.com/sdk/install"
  exit 1
fi

if ! (which bqschemaupdater) > /dev/null; then
  echo "Please install 'bqschemaupdater' from Chrome's infra.git"
  echo "  Checkout infra.git then run: eval \`./go/env.py\`"
  exit 1
fi

if [ $# != 1 ]; then
  echo "usage: setup_bigquery.sh <instanceid>"
  echo ""
  echo "Pass one argument which is the instance name"
  exit 1
fi

APPID=$1

echo "- Make sure the BigQuery API is enabled for the project:"
# It is enabled by default for new projects, but it wasn't for older projects.
gcloud services enable --project "${APPID}" bigquery-json.googleapis.com

# Permission is grantes via overground, skipping here

echo "- Create the dataset:"
echo ""
echo "  Warning: On first 'bq' invocation, it'll try to find out default"
echo "    credentials and will ask to select a default app; just press enter to"
echo "    not select a default."

if ! (bq --location=US mk --dataset \
  --description 'unified fleet system statistics' "${APPID}":ufs); then
  echo ""
  echo "Dataset creation failed. Assuming the dataset already exists. At worst"
  echo "the following command will fail."
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.ChromePlatformRow  \
    -table "${APPID}".ufs.chrome_platforms); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.chrome_platforms"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.VlanRow  \
    -table "${APPID}".ufs.vlans); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.vlans"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.MachineRow  \
    -table "${APPID}".ufs.machines); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.machines"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.RackRow  \
    -table "${APPID}".ufs.racks); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.racks"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.RackLSEPrototypeRow  \
    -table "${APPID}".ufs.rack_lse_prototypes); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.rack_lse_prototypes"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.MachineLSEPrototypeRow  \
    -table "${APPID}".ufs.machine_lse_prototypes); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.machine_lse_prototypes"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.MachineLSERow  \
    -table "${APPID}".ufs.machine_lses); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.machine_lses"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.RackLSERow  \
    -table "${APPID}".ufs.rack_lses); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.rack_lses"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.KVMRow  \
    -table "${APPID}".ufs.kvms); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.kvms"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.RPMRow  \
    -table "${APPID}".ufs.rpms); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.rpms"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.SwitchRow  \
    -table "${APPID}".ufs.switches); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.switches"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.DracRow  \
    -table "${APPID}".ufs.dracs); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.dracs"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.NicRow  \
    -table "${APPID}".ufs.nics); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.nics"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.DHCPConfigRow  \
    -table "${APPID}".ufs.dhcps); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.dhcps"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.IPRow  \
    -table "${APPID}".ufs.ips); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.ips"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.StateRecordRow  \
    -table "${APPID}".ufs.state_records); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.state_records"
  echo ""
  echo "and run this script again."
  exit 1
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message unifiedfleet.api.v1.proto.bigquery.ChangeEventRow  \
    -table "${APPID}".ufs.change_events); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:ufs.change_events"
  echo ""
  echo "and run this script again."
  exit 1
fi
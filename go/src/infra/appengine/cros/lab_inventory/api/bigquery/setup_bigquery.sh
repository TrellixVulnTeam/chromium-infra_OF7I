#!/bin/sh
# Copyright 2019 The LUCI Authors. All rights reserved.
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


# TODO(xixuan): The stock role "roles/bigquery.dataEditor" grants too much rights.
# Better to create a new custom role with only access "bigquery.tables.updateData".
# https://cloud.google.com/iam/docs/understanding-custom-roles
# https://cloud.google.com/iam/docs/creating-custom-roles#iam-custom-roles-create-gcloud


# https://cloud.google.com/iam/docs/granting-roles-to-service-accounts
# https://cloud.google.com/bigquery/docs/access-control
echo "- Grant access to the AppEngine app to the role account:"
gcloud projects add-iam-policy-binding "${APPID}" \
    --member serviceAccount:"${APPID}"@appspot.gserviceaccount.com \
    --role roles/bigquery.dataEditor


echo "- Create the dataset:"
echo ""
echo "  Warning: On first 'bq' invocation, it'll try to find out default"
echo "    credentials and will ask to select a default app; just press enter to"
echo "    not select a default."

if ! (bq --location=US mk --dataset \
  --description 'Lab inventory statistics' "${APPID}":inventory); then
  echo ""
  echo "Dataset creation failed. Assuming the dataset already exists. At worst"
  echo "the following command will fail."
fi

echo "- Populate the BigQuery schema:"
echo ""
echo "  Warning: On first 'bqschemaupdater' invocation, it'll request default"
echo "    credentials which is stored independently than 'bq'."
if ! (bqschemaupdater -force \
    -message apibq.LabInventory  \
    -table "${APPID}".inventory.lab); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:inventory.lab"
  echo ""
  echo "and run this script again."
  exit 1
fi

if ! (bqschemaupdater -force \
    -message apibq.HWIDInventory  \
    -table "${APPID}".inventory.hwid_server); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:inventory.hwid_server"
  echo ""
  echo "and run this script again."
  exit 1
fi

if ! (bqschemaupdater -force \
    -message apibq.ManufacturingInventory  \
    -table "${APPID}".inventory.manufacturing); then
  echo ""
  echo ""
  echo "Oh no! You may need to restart from scratch. You can do so with:"
  echo ""
  echo "  bq rm ${APPID}:inventory.manufacturing"
  echo ""
  echo "and run this script again."
  exit 1
fi

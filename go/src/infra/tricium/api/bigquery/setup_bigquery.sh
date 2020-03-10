#!/bin/bash
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Creates the datasets and updates schemas for Tricium BQ tables.
# Usage: update_views.sh [appid]
# Requirements: You must first install bq

set -eu

if ! (command -v bq) > /dev/null; then
  echo "Please install 'bq' from gcloud SDK"
  echo "  https://cloud.google.com/sdk/install"
  exit 1
fi

if ! (command -v bqschemaupdater) > /dev/null; then
  echo "Please install 'bqschemaupdater' from Chrome's infra.git"
  echo "  Checkout infra.git then run: eval $(../../../../../env.py)"
  exit 1
fi

if [ $# -ne 1 ]; then
  echo "Usage: update_views.sh appid"
  exit 1
fi

APPID="$1"

echo "Creating the datasets."
echo "Note: On first 'bq' invocation, it'll try to find out default"
echo "credentials and will ask to select a default app."

function create_datasets {
  if ! (bq --location=US mk --dataset --description "Analysis result statistics" \
    "${APPID}:analyzer"); then
    echo "Dataset 'analyzer' creation failed."
  fi

  if ! (bq --location=US mk --dataset --description "Events and user actions" \
    "${APPID}:events"); then
    echo "Dataset 'events' creation failed."
  fi
}

function update_schemas {
  echo "Updating the schema..."
  echo "Note: On first 'bqschemaupdater' run, it will request default"
  echo "credentials, which are stored independently from 'bq' permissions."

  if ! (bqschemaupdater -force -message apibq.AnalysisRun \
    -table "${APPID}.analyzer.results"); then
    echo "Failed to update ${APPID}:analyzer.results}."
    echo "You may need to delete the table with: bq rm ${APPID}:analyzer.results}"
    echo "and try again."
    exit 1
  fi
  if ! (bqschemaupdater -force -message apibq.FeedbackEvent \
    -table "${APPID}.events.feedback"); then
    echo "Failed to update ${APPID}:events.feedback}."
    echo "You may need to delete the table with: bq rm ${APPID}:analyzer.results}"
    echo "and try again."
    exit 1
  fi
}

script_dir=$(dirname "$0")
( cd "$script_dir"; create_datasets; update_schemas )

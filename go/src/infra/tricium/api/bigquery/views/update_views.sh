#!/bin/bash
# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Usage: update_views.sh appid
# Requirements: You must first install bq.

set -eu

if ! (command -v bq) > /dev/null; then
  echo "Please install 'bq' from gcloud SDK"
  echo "  https://cloud.google.com/sdk/install"
  exit 1
fi

if [ $# -eq 0 ]; then
  echo "Usage: update_views.sh appid"
  exit 1
fi
APPID="$1"

function update_view {
  viewname="$1"
  query=$(sed "s/tricium-prod/${APPID}/" < "$viewname.sql")
  echo "Updating ${viewname}..."
  bq rm -f "${APPID}:${viewname}"
  if ! (bq mk --project_id "$APPID" \
      --use_legacy_sql=false \
      --view="$query" "$viewname" ); then
    echo "Failed to update ${viewname}"
    exit 1
  fi
}

echo "Updating BigQuery views..."
# The order below is important because some views depend on other views, and
# update will fail if depended-on views aren't yet there.
update_view analyzer.comment_latency
update_view analyzer.comments_final
update_view analyzer.comments_selected
update_view analyzer.comments_with_ids
update_view analyzer.efficacy
update_view analyzer.efficacy_by_analyzer
update_view analyzer.efficacy_by_category
update_view events.comment_events
update_view events.not_useful_comments
update_view events.not_useful_proportion
update_view events.not_useful_by_analyzer
update_view events.not_useful_by_category

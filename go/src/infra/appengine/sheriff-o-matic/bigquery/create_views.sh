#!/bin/bash
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Change these values as required to set up new views.
APP_ID=sheriff-o-matic-staging
project_names_without_test_results=("chromeos" "fuchsia")

resultdb_dataset="chrome-luci-data.chromium_staging"
if [ "$APP_ID" == "sheriff-o-matic" ]; then
  resultdb_dataset="chrome-luci-data.chromium"
fi

chrome_project_condition="create_time > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY) AND REGEXP_CONTAINS(b.builder.project, \"^((chrome|chromium)(-m[0-9]+(-.*)?)?)$\")"
angle_project_condition="create_time > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY) AND ((b.builder.project = \"angle\" AND b.builder.bucket=\"ci\") OR (b.builder.project = \"chromium\" AND b.builder.bucket=\"ci\"))"

# Chrome
project_name="chrome"
echo "creating data set and views for project chrome"
bq --project_id $APP_ID mk -d "$project_name"
sed -e s/APP_ID/$APP_ID/g -e s/PROJECT_NAME/"$project_name"/g -e s/PROJECT_FILTER_CONDITIONS/"$chrome_project_condition"/g step_status_transitions_customized.sql | bq --project_id $APP_ID query --use_legacy_sql=false
sed -e s/APP_ID/$APP_ID/g -e s/PROJECT_NAME/"$project_name"/g -e s/RESULTDB_DATASET/"$resultdb_dataset"/g -e s/PROJECT_FILTER_CONDITIONS/"$chrome_project_condition"/g failing_steps_customized.sql | bq query --project_id $APP_ID --use_legacy_sql=false
sed -e s/APP_ID/$APP_ID/g -e s/PROJECT_NAME/"$project_name"/g sheriffable_failures.sql | bq --project_id $APP_ID query --use_legacy_sql=false

# Special handling for angle tree
echo "creating data set and views for project angle"
project_name="angle"
bq --project_id $APP_ID mk -d "$project_name"
sed -e s/APP_ID/$APP_ID/g -e s/PROJECT_NAME/"$project_name"/g -e s/PROJECT_FILTER_CONDITIONS/"$angle_project_condition"/g step_status_transitions_customized.sql | bq --project_id $APP_ID query --use_legacy_sql=false
sed -e s/APP_ID/$APP_ID/g -e s/PROJECT_NAME/"$project_name"/g -e s/RESULTDB_DATASET/"$resultdb_dataset"/g -e s/PROJECT_FILTER_CONDITIONS/"$angle_project_condition"/g failing_steps_customized.sql | bq query --project_id $APP_ID --use_legacy_sql=false
sed -e s/APP_ID/$APP_ID/g -e s/PROJECT_NAME/"$project_name"/g sheriffable_failures.sql | bq --project_id $APP_ID query --use_legacy_sql=false

# Other projects
for project_name in "${project_names_without_test_results[@]}"
do
    echo "creating data set and views for project: $project_name"
    bq --project_id $APP_ID mk -d "$project_name"
    sed -e s/APP_ID/$APP_ID/g -e s/PROJECT_NAME/"$project_name"/g step_status_transitions.sql | bq --project_id $APP_ID query --use_legacy_sql=false
    sed -e s/APP_ID/$APP_ID/g -e s/PROJECT_NAME/"$project_name"/g failing_steps_without_test_results.sql | bq query --project_id $APP_ID --use_legacy_sql=false
    sed -e s/APP_ID/$APP_ID/g -e s/PROJECT_NAME/"$project_name"/g sheriffable_failures.sql | bq --project_id $APP_ID query --use_legacy_sql=false
done

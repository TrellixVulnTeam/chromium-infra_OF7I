# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Definitions of weblayer crons."""

load("//lib/infra.star", "infra")

def builder(name, recipe, schedule, bucket, execution_timeout = None):
    luci.builder(
        name = name,
        bucket = bucket,
        executable = infra.recipe(recipe),
        dimensions = {
            "os": "Ubuntu-16.04",
            "cpu": "x86-64",
            "pool": "luci.infra.cron",
            "builderless": "1",
        },
        properties = {
            "total_cq_checks": 30,
            "interval_between_checks_in_secs": 120,
        },
        service_account = "chrome-weblayer-builder@chops-service-accounts.iam.gserviceaccount.com",
        execution_timeout = execution_timeout or time.hour,
        schedule = schedule,
    )
    luci.list_view_entry(
        builder = name,
        list_view = bucket,
    )

builder(
    name = "refresh-weblayer-skew-tests",
    recipe = "refresh_weblayer_skew_tests",
    schedule = "triggered",
    bucket = "tasks",
    execution_timeout = 6 * time.hour,
)

builder(
    name = "create-weblayer-skew-tests",
    recipe = "build_weblayer_version_tests_apk_cipd_pkg",
    schedule = "with 600s interval",
    bucket = "cron",
    execution_timeout = 6 * time.hour,
)

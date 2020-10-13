# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Definitions of weblayer crons."""

load("//lib/infra.star", "infra")

def cron(name, recipe, execution_timeout = None):
    luci.builder(
        name = name,
        bucket = "cron",
        executable = infra.recipe(recipe),
        dimensions = {
            "os": "Ubuntu-16.04",
            "cpu": "x86-64",
            "pool": "luci.infra.cron",
            "builderless": "1",
        },
        properties = {
            "mastername": "chromium.infra.cron",
            "total_cq_checks": 30,
            "interval_between_checks_in_secs": 120,
        },
        service_account = "chrome-weblayer-builder@chops-service-accounts.iam.gserviceaccount.com",
        execution_timeout = execution_timeout or time.hour,
        schedule = "with 600s interval",
    )
    luci.list_view_entry(
        builder = name,
        list_view = "cron",
    )

cron(
    name = "create-weblayer-skew-tests",
    recipe = "build_weblayer_version_tests_apk_cipd_pkg",
    execution_timeout = 6 * time.hour,
)

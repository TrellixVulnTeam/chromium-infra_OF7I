# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Definitions of WPT import/export crons."""

load("//lib/build.star", "build")

def cron(name, recipe, execution_timeout = None):
    luci.builder(
        name = name,
        bucket = "cron",
        executable = build.recipe(recipe),
        dimensions = {
            "os": "Ubuntu-18.04",
            "cpu": "x86-64",
            "pool": "luci.infra.cron",
        },
        properties = {
            "mastername": "chromium.infra.cron",
        },
        service_account = "wpt-autoroller@chops-service-accounts.iam.gserviceaccount.com",
        execution_timeout = execution_timeout or time.hour,
        schedule = "with 60s interval",
        experiments = {"luci.recipes.use_python3": 100},
    )
    luci.list_view_entry(
        builder = name,
        list_view = "cron",
    )

cron(name = "wpt-exporter", recipe = "wpt_export")
cron(name = "wpt-importer", recipe = "wpt_import", execution_timeout = 6 * time.hour)

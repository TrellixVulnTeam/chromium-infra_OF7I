# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Jobs that publish tarballs with Chromium source code."""

load("//lib/build.star", "build")
load("//lib/infra.star", "infra")

def builder(name, builder_dimension = None, cores = 8, **kwargs):
    luci.builder(
        name = name,
        bucket = "cron",
        service_account = "chromium-tarball-builder@chops-service-accounts.iam.gserviceaccount.com",
        dimensions = {
            "pool": "luci.infra.cron",
            "os": "Ubuntu-16.04",
            "cpu": "x86-64",
            "cores": str(cores),
            "builderless": "1",
        },
        **kwargs
    )
    luci.list_view_entry(
        builder = name,
        list_view = "cron",
    )

builder(
    name = "publish_tarball_dispatcher",
    builder_dimension = "publish_tarball",  # runs on same bots as 'publish_tarball'
    executable = build.recipe("publish_tarball"),
    execution_timeout = 10 * time.minute,
    schedule = "37 */3 * * *",  # every 3 hours
    triggers = ["publish_tarball"],
)

builder(
    name = "publish_tarball",
    executable = build.recipe("publish_tarball"),
    execution_timeout = 8 * time.hour,
    # Each trigger from 'publish_tarball_dispatcher' should result in a build.
    triggering_policy = scheduler.greedy_batching(max_batch_size = 1),
    triggers = ["Build From Tarball"],
)

builder(
    name = "Build From Tarball",
    executable = infra.recipe("build_from_tarball"),
    execution_timeout = 5 * time.hour,
    # Each trigger from 'publish_tarball' should result in a build.
    triggering_policy = scheduler.greedy_batching(max_batch_size = 1),
    cores = 32,
)

luci.notifier(
    name = "release-tarballs",
    on_failure = True,
    on_status_change = True,
    notify_emails = [
        "raphael.kubo.da.costa@intel.com",
        "thestig@chromium.org",
        "thomasanderson@chromium.org",
    ],
    notified_by = [
        "Build From Tarball",
        "publish_tarball",
    ],
)

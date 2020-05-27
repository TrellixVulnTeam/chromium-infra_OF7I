# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Gatekeeper cron."""

load('//lib/build.star', 'build')

luci.builder(
    name = 'Chromium Gatekeeper',
    bucket = 'cron',
    # TODO(crbug.com/1015181): Stop passing use_bbagent=False after
    # fix(http://crrev.com/c/2217053) lands
    executable = build.recipe('gatekeeper', use_bbagent=False),
    service_account = 'gatekeeper-builder@chops-service-accounts.iam.gserviceaccount.com',
    dimensions = {
        'builder': 'Chromium Gatekeeper',
        'os': 'Ubuntu-16.04',
        'cpu': 'x86-64',
        'pool': 'luci.infra.cron',
    },
    schedule = 'with 1m interval',
)

luci.list_view(
    name = 'gatekeeper',
    entries = ['Chromium Gatekeeper'],
)

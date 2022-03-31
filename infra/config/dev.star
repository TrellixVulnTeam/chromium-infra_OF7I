#!/usr/bin/env lucicfg
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""LUCI project configuration for the development instance of LUCI.

After modifying this file execute it ('./dev.star') to regenerate the configs.
This is also enforced by PRESUBMIT.py script.
"""

load("//lib/infra.star", "infra")

lucicfg.check_version("1.30.9", "Please update depot_tools")

# Global recipe defaults
luci.recipe.defaults.cipd_version.set("refs/heads/main")
luci.recipe.defaults.use_bbagent.set(True)

# Enable v2 bucket names in LUCI Scheduler config.
lucicfg.enable_experiment("crbug.com/1182002")

lucicfg.config(
    config_dir = "generated",
    tracked_files = [
        "cr-buildbucket-dev.cfg",
        "luci-logdog-dev.cfg",
        "luci-notify-dev.cfg",
        "luci-notify-dev/email-templates/*",
        "luci-scheduler-dev.cfg",
        "realms-dev.cfg",
        "tricium-dev.cfg",
    ],
    fail_on_warnings = True,
    lint_checks = ["default"],
)

# Just copy tricium-dev.cfg as is to the outputs.
lucicfg.emit(
    dest = "tricium-dev.cfg",
    data = io.read_file("tricium-dev.cfg"),
)

luci.project(
    name = "infra",
    dev = True,
    buildbucket = "cr-buildbucket-dev.appspot.com",
    logdog = "luci-logdog-dev.appspot.com",
    notify = "luci-notify-dev.appspot.com",
    scheduler = "luci-scheduler-dev.appspot.com",
    swarming = "chromium-swarm-dev.appspot.com",
    acls = [
        acl.entry(
            roles = [
                acl.BUILDBUCKET_READER,
                acl.LOGDOG_READER,
                acl.PROJECT_CONFIGS_READER,
                acl.SCHEDULER_READER,
            ],
            groups = "all",
        ),
        acl.entry(
            roles = acl.SCHEDULER_OWNER,
            groups = "project-infra-troopers",
        ),
        acl.entry(
            roles = acl.LOGDOG_WRITER,
            groups = "luci-logdog-chromium-dev-writers",
        ),
        acl.entry(
            roles = acl.BUILDBUCKET_TRIGGERER,
            users = "adhoc-testing@luci-token-server-dev.iam.gserviceaccount.com",
        ),
    ],
    bindings = [
        # LED users.
        luci.binding(
            roles = "role/swarming.taskTriggerer",
            groups = "mdb/chrome-troopers",
        ),
    ],
    enforce_realms_in = [
        "cr-buildbucket-dev",
        "luci-scheduler-dev",
    ],
)

luci.logdog(
    gs_bucket = "chromium-luci-logdog",
    cloud_logging_project = "luci-logdog-dev",
)

luci.bucket(name = "ci")

luci.builder.defaults.experiments.set({
    "luci.buildbucket.bbagent_getbuild": 100,
})
luci.builder.defaults.execution_timeout.set(30 * time.minute)

def ci_builder(
        name,
        os,
        recipe = "infra_continuous",
        tree_closing = False):
    infra.builder(
        name = name,
        bucket = "ci",
        executable = infra.recipe(recipe, use_python3 = True),
        os = os,
        cpu = "x86-64",
        pool = "luci.chromium.ci",
        service_account = "adhoc-testing@luci-token-server-dev.iam.gserviceaccount.com",
        triggered_by = [infra.poller()],
        notifies = ["dev tree closer"] if tree_closing else None,
    )

luci.tree_closer(
    name = "dev tree closer",
    tree_status_host = "infra-status.appspot.com",
    template = "default",
)

luci.notifier_template(
    name = "default",
    body = "{{ stepNames .MatchingFailedSteps }} on {{ buildUrl . }} {{ .Build.Builder.Builder }} from {{ .Build.Output.GitilesCommit.Id }}",
)

ci_builder(name = "infra-continuous-bionic-64", os = "Ubuntu-18.04", tree_closing = True)
ci_builder(name = "infra-continuous-win10-64", os = "Windows-10")
ci_builder(name = "infra-continuous-win11-64", os = "Windows-11")

def adhoc_builder(
        name,
        os,
        executable,
        extra_dims = None,
        properties = None,
        experiments = None,
        schedule = None,
        triggered_by = None):
    dims = {"os": os, "cpu": "x86-64", "pool": "luci.chromium.ci"}
    if extra_dims:
        dims.update(**extra_dims)
    luci.builder(
        name = name,
        bucket = "ci",
        executable = executable,
        dimensions = dims,
        properties = properties,
        experiments = experiments,
        service_account = "adhoc-testing@luci-token-server-dev.iam.gserviceaccount.com",
        build_numbers = True,
        schedule = schedule,
        triggered_by = triggered_by,
    )

adhoc_builder(
    name = "gerrit-hello-world-bionic-64",
    os = "Ubuntu-18.04",
    executable = infra.recipe("gerrit_hello_world", use_python3 = True),
    schedule = "triggered",  # triggered manually via Scheduler UI
)
adhoc_builder(
    name = "gsutil-hello-world-bionic-64",
    os = "Ubuntu-18.04",
    executable = infra.recipe("gsutil_hello_world", use_python3 = True),
    schedule = "triggered",  # triggered manually via Scheduler UI
)
adhoc_builder(
    name = "gsutil-hello-world-win10-64",
    os = "Windows-10",
    executable = infra.recipe("gsutil_hello_world", use_python3 = True),
    schedule = "triggered",  # triggered manually via Scheduler UI
)
adhoc_builder(
    name = "build-proto-linux",
    os = "Ubuntu",
    executable = luci.recipe(
        name = "futures:examples/background_helper",
        cipd_package = "infra/recipe_bundles/chromium.googlesource.com/infra/luci/recipes-py",
        use_python3 = True,
    ),
    schedule = "with 10m interval",
)
adhoc_builder(
    name = "build-proto-win",
    os = "Windows-10",
    executable = luci.recipe(
        name = "futures:examples/background_helper",
        cipd_package = "infra/recipe_bundles/chromium.googlesource.com/infra/luci/recipes-py",
        use_python3 = True,
    ),
    schedule = "with 10m interval",
)

adhoc_builder(
    name = "linux-rel-buildbucket",
    os = "Ubuntu-18.04",
    executable = luci.recipe(
        name = "placeholder",
        cipd_package = "infra/recipe_bundles/chromium.googlesource.com/infra/luci/recipes-py",
        use_python3 = True,
    ),
    properties = {
        "status": "SUCCESS",
        "steps": [
            {
                "name": "can_outlive_parent child",
                "child_build": {
                    "buildbucket": {
                        "builder": {
                            "project": "infra",
                            "bucket": "ci",
                            "builder": "linux-rel-buildbucket-child",
                        },
                    },
                    "life_time": "DETACHED",
                },
            },
            {
                "name": "cannot_outlive_parent child",
                "child_build": {
                    "id": "bounded_child",
                    "buildbucket": {
                        "builder": {
                            "project": "infra",
                            "bucket": "ci",
                            "builder": "linux-rel-buildbucket-child",
                        },
                    },
                    "life_time": "BUILD_BOUND",
                },
            },
            {
                "name": "collect children",
                "collect_children": {
                    "child_build_step_ids": ["bounded_child"],
                },
            },
        ],
    },
    schedule = "with 10m interval",
)

adhoc_builder(
    name = "linux-rel-buildbucket-child",
    os = "Ubuntu-18.04",
    executable = luci.recipe(
        name = "placeholder",
        cipd_package = "infra/recipe_bundles/chromium.googlesource.com/infra/luci/recipes-py",
        use_python3 = True,
    ),
    properties = {
        "status": "SUCCESS",
        "steps": [
            {
                "name": "hello",
                "fake_step": {
                    "duration_secs": 90,
                },
            },
        ],
    },
)

luci.notifier(
    name = "nodir-spam",
    on_success = True,
    on_failure = True,
    notify_emails = ["nodir+spam@google.com"],
    template = "test",
    notified_by = ["infra-continuous-bionic-64"],
)

luci.notifier(
    name = "luci-notify-test-alerts",
    on_success = True,
    on_failure = True,
    notify_emails = ["luci-notify-test-alerts@chromium.org"],
    template = "test",
    notified_by = ["infra-continuous-bionic-64"],
)

luci.notifier_template(
    name = "test",
    body = """{{.Build.Builder | formatBuilderID}} notification

<a href="{{buildUrl .}}">Build {{.Build.Number}}</a>
has completed.

{{template "steps" .}}
""",
)

luci.notifier_template(
    name = "steps",
    body = """Renders steps.

<ol>
{{range $s := .Build.Steps}}
  <li>{{$s.Name}}</li>
{{end}}
</ol>
""",
)

################################################################################
## Realms used by skylab-staging-bot-fleet for its pools and admin tasks.
#
# The corresponding realms in the prod universe live in "chromeos" project.
# There's no "chromeos" project in the dev universe, so we define the realms
# here instead.

SKYLAB_ADMIN_SCHEDULERS = [
    "project-chromeos-skylab-schedulers",
    "mdb/chromeos-build-deputy",
]

luci.realm(
    name = "pools/skylab",
    bindings = [
        luci.binding(
            roles = "role/swarming.poolOwner",
            groups = "administrators",
        ),
        luci.binding(
            roles = "role/swarming.poolUser",
            groups = SKYLAB_ADMIN_SCHEDULERS,
        ),
        luci.binding(
            roles = "role/swarming.poolViewer",
            groups = "chromium-swarm-dev-view-all-bots",
        ),
    ],
)

luci.realm(
    name = "skylab-staging-bot-fleet/admin",
    bindings = [
        luci.binding(
            roles = "role/swarming.taskServiceAccount",
            users = "skylab-admin-task@chromeos-service-accounts-dev.iam.gserviceaccount.com",
        ),
        luci.binding(
            roles = "role/swarming.taskTriggerer",
            groups = SKYLAB_ADMIN_SCHEDULERS,
        ),
    ],
)

# TODO(crbug.com/1238772): remove after dev configs get prepared in "chromeos"
# project.
luci.realm(
    name = "pools/chromeos",
    bindings = [
        luci.binding(
            roles = "role/swarming.poolOwner",
            groups = "administrators",
        ),
        luci.binding(
            roles = "role/swarming.poolUser",
            groups = "chromium-swarm-dev-privileged-users",
        ),
        luci.binding(
            roles = "role/swarming.poolViewer",
            groups = "chromium-swarm-dev-view-all-bots",
        ),
    ],
)
################################################################################
## Realms used for Swarming client integration tests.

luci.realm(
    name = "pools/tests",
    bindings = [
        luci.binding(
            roles = "role/swarming.poolOwner",
            groups = ["mdb/chrome-troopers"],
        ),
        luci.binding(
            roles = "role/swarming.poolViewer",
            groups = "chromium-swarm-dev-view-all-bots",
        ),
        luci.binding(
            roles = "role/swarming.poolUser",
            groups = "project-infra-tests-submitters",
        ),
    ],
)

luci.realm(
    name = "tests",
    bindings = [
        luci.binding(
            roles = "role/swarming.taskTriggerer",
            groups = "project-infra-tests-submitters",
        ),
    ],
)

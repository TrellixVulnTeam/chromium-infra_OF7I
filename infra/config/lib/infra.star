# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Functions and constants related to infra.git used by all modules."""

def poller():
    """Defines a gitiles poller polling infra.git repo."""
    return luci.gitiles_poller(
        name = "infra-gitiles-trigger",
        bucket = "ci",
        repo = infra.REPO_URL,
    )

def recipe(name, use_bbagent = True):
    """Defines a recipe hosted in the infra.git recipe bundle.

    Args:
      name: name of the recipe.
      use_bbagent: if True, execute it through bbagent.

    Returns:
      A luci.recipe(...) object.
    """
    recipe = name
    if use_bbagent:
        name += "-bbagent"
    else:
        name += "-kitchen"

    return luci.recipe(
        name = name,
        recipe = recipe,
        cipd_package = "infra/recipe_bundles/chromium.googlesource.com/infra/infra",
        use_bbagent = use_bbagent,
    )

def console_view(name, title, repo = None):
    """Defines a console view with infra header."""
    luci.console_view(
        name = name,
        title = title,
        repo = repo or infra.REPO_URL,
        header = "//data/infra_console_header.textpb",
    )

def cq_group(name, repo = None, tree_status_host = None):
    """Defines a CQ group watching refs/heads/(main|master).

    This currently watches the 'master' ref only for repos which
    are not `infra.REPO_URL`.

    Args:
      name: The human- and machine-readable name of the CQ group.
      repo: https URL of the git repo for this CQ group to monitor.
      tree_status_host: Hostname of the tree_status_host for this CQ group.
    """
    refs = ["refs/heads/main"]
    if repo and repo != infra.REPO_URL:
        refs.append("refs/heads/master")
    luci.cq_group(
        name = name,
        watch = cq.refset(
            repo = repo or infra.REPO_URL,
            refs = refs,
        ),
        tree_status_host = tree_status_host,
        retry_config = cq.RETRY_NONE,
    )

def builder(
        *,

        # Basic required stuff.
        name,
        bucket,
        executable,

        # Dimensions.
        os,
        cpu = None,
        pool = None,

        # Swarming environ.
        service_account = None,

        # Misc tweaks.
        properties = None,
        schedule = None,
        extra_dimensions = None,
        use_realms = False,

        # Triggering relations.
        triggered_by = None,

        # LUCI-Notify config.
        notifies = None):
    """Defines a basic infra builder (CI or Try).

    It is a builder that needs an infra.git checkout to do stuff.

    Depending on value of `bucket`, will chose a default pool (ci or flex try),
    the service account and build_numbers settings.

    Args:
      name: name of the builder.
      bucket: name of the bucket to put it in.
      executable: a recipe or other luci.executable(...) to run.
      os: a target OS dimension.
      cpu: a target CPU dimension.
      pool: target Swarming pool.
      service_account: a service account to run the build as.
      properties: a dict with properties to pass to the builder.
      schedule: a string with builder schedule for cron-like builders.
      extra_dimensions: a dict with additional Swarming dimensions.
      use_realms: True to launch realms-aware builds.
      triggered_by: builders that trigger this one.
      notifies: what luci.notifier(...) to notify when its done.
    """
    if bucket == "ci":
        pool = pool or "luci.flex.ci"
        service_account = service_account or infra.SERVICE_ACCOUNT_CI
        build_numbers = True
    elif bucket == "try":
        pool = pool or "luci.flex.try"
        service_account = service_account or infra.SERVICE_ACCOUNT_TRY
        build_numbers = None  # leave it unset in the generated file
    else:
        fail("unknown bucket")

    caches = [infra.cache_gclient_with_go]
    if os.startswith("Mac"):
        caches.append(infra.cache_osx_sdk)

    dimensions = {"os": os, "cpu": cpu or "x86-64", "pool": pool}
    if extra_dimensions:
        if ("cpu" in extra_dimensions or "os" in extra_dimensions or
            "pool" in extra_dimensions):
            fail("specify 'cpu', 'os', or 'pool' directly")
        dimensions.update(extra_dimensions)

    experiments = {}
    if use_realms:
        experiments["luci.use_realms"] = 100

    luci.builder(
        name = name,
        bucket = bucket,
        executable = executable,
        dimensions = dimensions,
        service_account = service_account,
        properties = properties,
        caches = caches,
        experiments = experiments,
        build_numbers = build_numbers,
        schedule = schedule,
        task_template_canary_percentage = 30,
        triggered_by = triggered_by,
        notifies = notifies,
    )

def _tree_closing_notifiers():
    return [
        luci.tree_closer(
            name = "infra tree closer",
            tree_status_host = "infra-status.appspot.com",
            template = "status",
        ),
        luci.notifier(
            name = "notify tandrii@",
            on_new_status = ["FAILURE"],
            notify_emails = ["tandrii+infra-continuous-self-appointed-gardener@google.com"],
        ),
    ]

_OS_TO_CATEGORY = {
    "Ubuntu": "linux",
    "Mac": "mac",
    "Windows": "win",
}

def category_from_os(os, short = False):
    """Given e.g. 'Ubuntu-16.10' returns e.g. 'linux|16.10'.

    Args:
      os: OS dimension name.
      short: if True, strip the version.

    Returns:
      A console category name.
    """
    os, _, ver = os.partition("-")
    category = _OS_TO_CATEGORY.get(os, os.lower())
    if not short:
        category += "|" + ver
    return category

infra = struct(
    REPO_URL = "https://chromium.googlesource.com/infra/infra",

    # Note: try account is also used by all presubmit builders in this project.
    SERVICE_ACCOUNT_TRY = "infra-try-builder@chops-service-accounts.iam.gserviceaccount.com",
    SERVICE_ACCOUNT_CI = "infra-ci-builder@chops-service-accounts.iam.gserviceaccount.com",
    cache_gclient_with_go = swarming.cache("infra_gclient_with_go"),
    cache_osx_sdk = swarming.cache("osx_sdk"),
    poller = poller,
    recipe = recipe,
    console_view = console_view,
    cq_group = cq_group,
    builder = builder,
    tree_closing_notifiers = _tree_closing_notifiers,
    category_from_os = category_from_os,
)

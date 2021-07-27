# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import contextlib

from recipe_engine import recipe_api

class InfraCheckoutApi(recipe_api.RecipeApi):
  """Stateless API for using public infra gclient checkout."""

  # Named cache shared across builders using public infra gclient checkout.
  PUBLIC_NAMED_CACHE = 'infra_gclient_with_go'
  # Ditto but for builders which use internal gclient checkout.
  INTERNAL_NAMED_CACHE = 'infra_internal_gclient_with_go'

  def checkout(self, gclient_config_name,
               patch_root=None,
               path=None,
               internal=False,
               named_cache=None,
               generate_env_with_system_python=False,
               go_version_variant=None,
               **kwargs):
    """Fetches infra gclient checkout into a given path OR named_cache.

    Arguments:
      * gclient_config_name (string) - name of gclient config.
      * patch_root (path or string) - path **inside** infra checkout to git repo
        in which to apply the patch. For example, 'infra/luci' for luci-py repo.
        If None (default), no patches will be applied.
      * path (path or string) - path to where to create/update infra checkout.
        If None (default) - path is cache with customizable name (see below).
      * internal (bool) - by default, False, meaning infra gclient checkout
          layout is assumed, else infra_internal.
          This has an effect on named_cache default and inside which repo's
          go corner the ./go/env.py command is run.
      * named_cache - if path is None, this allows to customize the name of the
        cache. Defaults to PUBLIC_NAMED_CACHE or INTERNAL_NAMED_CACHE, depending
        on `internal` argument value.
        Note: your cr-buildbucket.cfg should specify named_cache for swarming to
          prioritize bots which actually have this cache populated by prior
          runs. Otherwise, using named cache isn't particularly useful, unless
          your pool of builders is very small.
      * generate_env_with_system_python uses the bot "infra_system" python to
        generate infra.git's ENV. This is needed for bots which build the
        "infra/infra_python/${platform}" CIPD packages because they incorporate
        the checkout's VirtualEnv inside the package. This, in turn, results in
        the CIPD package containing absolute paths to the Python that was used
        to create it. In order to enable this madness to work, we ensure that
        the Python is a system Python, which resides at a fixed path. No effect
        on arm64 because the arm64 bots have no such python available.
      * go_version_variant can be set go "legacy" or "bleeding_edge" to force
        the builder to use a non-default Go version. What exact Go versions
        correspond to "legacy" and "bleeding_edge" and default is defined in
        bootstrap.py in infra.git.
      * kwargs - passed as is to bot_update.ensure_checkout.

    Returns:
      a Checkout object with commands for common actions on infra checkout.
    """
    assert gclient_config_name, gclient_config_name
    if named_cache is None:
      named_cache = (self.INTERNAL_NAMED_CACHE if internal else
                     self.PUBLIC_NAMED_CACHE)
    path = path or self.m.path['cache'].join(named_cache)
    self.m.file.ensure_directory('ensure builder dir', path)

    # arm64 bots don't have this system python stuff
    if generate_env_with_system_python and (
        self.m.platform.arch == 'arm' and self.m.platform.bits == 64):
      generate_env_with_system_python = False

    with self.m.context(cwd=path):
      self.m.gclient.set_config(gclient_config_name)
      if generate_env_with_system_python:
        sys_py = self.m.path.join(self.m.infra_system.sys_bin_path, 'python')
        if self.m.platform.is_win:
          sys_py += '.exe'
        self.m.gclient.c.solutions[0].custom_vars['infra_env_python'] = sys_py

      bot_update_step = self.m.bot_update.ensure_checkout(
          patch_root=patch_root, **kwargs)

    env_with_override = {
        'INFRA_GO_SKIP_TOOLS_INSTALL': '1',
        'GOFLAGS': '-mod-readonly',
    }
    if go_version_variant:
      env_with_override['INFRA_GO_VERSION_VARIANT'] = go_version_variant

    class Checkout(object):
      def __init__(self, m):
        self.m = m
        self._go_env = None
        self._go_env_prefixes = None
        self._go_env_suffixes = None
        self._committed = False

      @property
      def path(self):
        return path

      @property
      def bot_update_step(self):
        return bot_update_step

      @property
      def patch_root_path(self):
        assert patch_root
        return path.join(patch_root)

      def commit_change(self):
        assert patch_root
        with self.m.context(cwd=path.join(patch_root)):
          self.m.git(
              '-c', 'user.email=commit-bot@chromium.org',
              '-c', 'user.name=The Commit Bot',
              'commit', '-a', '-m', 'Committed patch',
              name='commit git patch')
        self._committed = True

      def get_commit_label(self):
        """Computes "<number>-<revision>" string identifying this commit.

        Either uses `Cr-Commit-Position` footer, if available, or falls back
        to `git number <rev>`.

        This label is used as part of Docker label name for images produced
        based on this checked out commit.

        Returns:
          A string.
        """
        props = self.bot_update_step.presentation.properties
        rev = props['got_revision']

        if 'got_revision_cp' in props:
          _, cp_num = self.m.commit_position.parse(props['got_revision_cp'])
        else:
          cwd = self.path.join('infra_internal' if internal else 'infra')
          with self.m.context(cwd=cwd, env={'CHROME_HEADLESS': '1'}):
            with self.m.depot_tools.on_path():
              result = self.m.git(
                  'number', rev,
                  name='get commit label',
                  stdout=self.m.raw_io.output(),
                  step_test_data=(
                      lambda: self.m.raw_io.test_api.stream_output('11112\n')))
          cp_num = int(result.stdout.strip())

        return '%d-%s' % (cp_num, rev[:7])

      def get_changed_files(self):
        """Lists files changed in the patch.

        This assumes that commit_change() has been called.

        Returns:
          A set of relative paths (strings) of changed files,
          including added, modified and deleted file paths.
        """
        assert patch_root
        assert self._committed
        # Grab a list of changed files.
        with self.m.context(cwd=path.join(patch_root)):
          result = self.m.git(
              'diff', '--name-only', 'HEAD', 'HEAD~',
              name='get change list',
              stdout=self.m.raw_io.output())
        files = result.stdout.splitlines()
        if len(files) < 50:
          result.presentation.logs['change list'] = sorted(files)
        else:
          result.presentation.logs['change list is too long'] = (
              '%d files' % len(files))
        return set(files)

      @staticmethod
      def gclient_runhooks():
        with self.m.context(cwd=path, env=env_with_override):
          self.m.gclient.runhooks()

      @contextlib.contextmanager
      def go_env(self):
        name = 'infra_internal' if internal else 'infra'
        self._ensure_go_env()
        with self.m.context(
            cwd=self.path.join(name, 'go', 'src', name),
            env=self._go_env,
            env_prefixes=self._go_env_prefixes,
            env_suffixes=self._go_env_suffixes):
          yield

      def _ensure_go_env(self):
        if self._go_env is not None:
          return  # already did this

        with self.m.context(cwd=self.path, env=env_with_override):
          where = 'infra_internal' if internal else 'infra'
          bootstrap = 'bootstrap_internal.py' if internal else 'bootstrap.py'
          step = self.m.python(
              'init infra go env',
              path.join(where, 'go', bootstrap),
              [self.m.json.output()],
              venv=True,
              infra_step=True,
              step_test_data=lambda: self.m.json.test_api.output({
                  'go_version': '1.66.6',
                  'env': {'GOROOT': str(path.join('golang', 'go'))},
                  'env_prefixes': {
                      'PATH': [str(path.join('golang', 'go'))],
                  },
                  'env_suffixes': {
                      'PATH': [str(path.join(where, 'go', 'bin'))],
                  },
              }))

        out = step.json.output
        step.presentation.step_text += 'Using go %s' % (out.get('go_version'),)

        self._go_env = env_with_override.copy()
        self._go_env.update(out['env'])
        self._go_env_prefixes = out['env_prefixes']
        self._go_env_suffixes = out['env_suffixes']

      @staticmethod
      def run_presubmit():
        assert patch_root
        revs = self.m.bot_update.get_project_revision_properties(patch_root)
        upstream = bot_update_step.json.output['properties'].get(revs[0])
        gerrit_change = self.m.buildbucket.build.input.gerrit_changes[0]
        with self.m.context(env={'PRESUBMIT_BUILDER': '1'}):
          return self.m.python(
              'presubmit',
              self.m.presubmit.presubmit_support_path, [
                  '--root', path.join(patch_root),
                  '--commit',
                  '--verbose', '--verbose',
                  '--issue', gerrit_change.change,
                  '--patchset', gerrit_change.patchset,
                  '--gerrit_url', 'https://' + gerrit_change.host,
                  '--gerrit_fetch',
                  '--upstream', upstream,
                  '--skip_canned', 'CheckTreeIsOpen',
              ], venv=True)

    return Checkout(self.m)

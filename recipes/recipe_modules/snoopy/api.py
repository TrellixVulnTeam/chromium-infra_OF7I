# Copyright 2022 The Chromium Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
from recipe_engine import recipe_api

# Usage of snoopy recipe_module will have significant downstream impact and to
# avoid any production outage, we are pinning the latest known good build of
# the tool here. Upstream changes are intentionally left out.
_LATEST_STABLE_VERSION = 'git_revision:636775ad273599bf5f3f5912394ceec4dd8b7aa9'

class SnoopyApi(recipe_api.RecipeApi):
  """API for interacting with Snoopy using the snoopy_broker tool."""

  def __init__(self, **kwargs):
    super(SnoopyApi, self).__init__(**kwargs)
    self._snoopy_bin = None

    if self._test_data.enabled:
      self._pid = self._test_data.get('pid', 12345)
    else:  # pragma: no cover
      self._pid = os.getpid()

  @property
  def snoopy_path(self):
    """Returns the path to snoopy_broker binary.

    When the property is accessed the first time, the latest, released
    snoopy_broker will be installed using cipd.
    """
    if self._snoopy_bin is None:
      snoopy_dir = self.m.path['start_dir'].join('snoopy')
      ensure_file = self.m.cipd.EnsureFile().add_package(
          'infra/tools/security/snoopy_broker/${platform}',
          _LATEST_STABLE_VERSION)
      self.m.cipd.ensure(snoopy_dir, ensure_file)
      self._snoopy_bin = snoopy_dir.join('snoopy_broker')
    return self._snoopy_bin

  @property
  def pid(self):
    """Returns the current process id if the recipe is running."""
    return self._pid

  def report_stage(self, stage, snoopy_url=None):
    """Reports task stage to local snoopy server.

    Args:
      * stage (str) - The stage at which task is executing currently, e.g.
        "start". Concept of task stage is native to Snoopy, this is a way of
        self-reporting phase of a task's lifecycle. This information is used in
        conjunction with process-inspected data to make security policy
        decisions.
        Valid stages: (start, fetch, compile, upload, upload-complete, test).
      * snoopy_url (Optional[str]) - URL for the local snoopy server, snoopy
        broker tool will use default if not specified.
    """
    args = [
      self.snoopy_path,
      '-report-stage',
      '-stage',
      stage,
    ]

    if snoopy_url:
      args.extend(['-backend-url', snoopy_url])

    # When task starts, they must report recipe name and recipe's process id.
    if stage == "start":
      args.extend(['-recipe', self.m.properties['recipe']])
    if stage == "start":
      args.extend(['-pid', self.pid])

    self.m.step('report_stage', args)

  def report_cipd(self, digest, pkg, iid, snoopy_url=None):
    """Reports cipd digest to local snoopy server.

    This is used to report produced artifacts hash and metadata to snoopy, it is
    used to generate provenance.

    Args:
      * digest (str) - The hash of the artifact.
      * pkg (str) - Name of the cipd package built.
      * iid (str) - Instance ID of the package.
      * snoopy_url (Optional[str]) - URL for the local snoopy server, snoopy
        broker tool will use default if not specified.
    """
    args = [
      self.snoopy_path,
      '-report-cipd',
      '-digest',
      digest,
      '-pkg-name',
      pkg,
      '-iid',
      iid,
    ]

    if snoopy_url:
      args.extend(['-backend-url', snoopy_url])

    self.m.step('report_cipd', args)

  def report_gcs(self, digest, guri, snoopy_url=None):
    """Reports cipd digest to local snoopy server.

    This is used to report produced artifacts hash and metadata to snoopy, it is
    used to generate provenance.

    Args:
      * digest (str) - The hash of the artifact.
      * guri (str) - Name of the GCS artifact built. This is the unique GCS URI,
        e.g. gs://bucket/path/to/binary.
      * snoopy_url (Optional[str]) - URL for the local snoopy server, snoopy
        broker tool will use default if not specified.
    """
    args = [
      self.snoopy_path,
      '-report-gcs',
      '-digest',
      digest,
      '-gcs-uri',
      guri,
    ]

    if snoopy_url:
      args.extend(['-backend-url', snoopy_url])

    self.m.step('report_gcs', args)

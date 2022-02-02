# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import recipe_api


# Usage of provenance recipe_module will be highly restricted in near future and
# to avoid any production outage, we are pinning the latest known good build of
# the tool here. Upstream changes are intentionally left out.
LATEST_STABLE_VERSION = 'git_revision:1b94785f1e15dc6c54840db4988d6c5bf0f4714c'


class ProvenanceApi(recipe_api.RecipeApi):
  """API for interacting with Provenance using the provenance tool."""

  def __init__(self, **kwargs):
    super(ProvenanceApi, self).__init__(**kwargs)
    self._provenance_bin = None

  @property
  def provenance_path(self):
    """Returns the path to provenance binary.

    When the property is accessed the first time, the latest, released
    provenance will be installed using cipd and verified using the provenance
    built-in to the OS image (if available).
    """
    if self._provenance_bin is None:
      provenance_dir = self.m.path['start_dir'].join('provenance')
      ensure_file = self.m.cipd.EnsureFile().add_package(
          'infra/tools/provenance/${platform}', LATEST_STABLE_VERSION)
      self.m.cipd.ensure(provenance_dir, ensure_file)
      self._provenance_bin = provenance_dir.join('provenance')
    return self._provenance_bin

  def generate(self, kms_crypto_key, input_file, output_file):
    """Generate an attestation file with a built artifact.

    Args:
      * kms_crypto_key (str) - The name of the encryption key, e.g.
        projects/chops-kms/locations/global/keyRings/[KEYRING]/cryptoKeys/[KEY]
      * input_file (Path) - The path to the input manifest file.
      * output_file (Path) - The path to the output attestation file.
    """
    args = [
        self.provenance_path,
        'generate',
        '-input',
        input_file,
        '-output',
        output_file,
        kms_crypto_key,
    ]

    self.m.step('generate', args)

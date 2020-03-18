# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import recipe_api


class CloudKMSApi(recipe_api.RecipeApi):
  """API for interacting with CloudKMS using the LUCI cloudkms tool."""

  def __init__(self, **kwargs):
    super(CloudKMSApi, self).__init__(**kwargs)
    self._cloudkms_bin = None

  @property
  def cloudkms_path(self):
    """Returns the path to LUCI cloudkms binary.

    When the property is accessed the first time, cloudkms will be installed
    using cipd.
    """
    if self._cloudkms_bin is None:
      cloudkms_dir = self.m.path['start_dir'].join('cloudkms')
      ensure_file = self.m.cipd.EnsureFile().add_package(
          'infra/tools/luci/cloudkms/${platform}', 'latest')
      self.m.cipd.ensure(cloudkms_dir, ensure_file)
      self._cloudkms_bin = cloudkms_dir.join('cloudkms')
    return self._cloudkms_bin

  def decrypt(self, kms_crypto_key, input_file, output_file):
    """Decrypt a ciphertext file with a CloudKMS key.

    Args:
      * kms_crypto_key (str) - The name of the encryption key, e.g.
        projects/chops-kms/locations/global/keyRings/[KEYRING]/cryptoKeys/[KEY]
      * input_file (Path) - The path to the input (ciphertext) file.
      * output_file (Path) - The path to the output (plaintext) file. It is
        recommended that this is inside api.path['cleanup'] to ensure the
        plaintext file will be cleaned up by recipe.
    """
    self.m.step('decrypt', [
        self.cloudkms_path, 'decrypt',
        '-input', input_file,
        '-output', output_file,
        kms_crypto_key,
    ])

  def sign(self,
           kms_crypto_key,
           input_file,
           output_file,
           service_account_creds_file=None):
    """Processes a plaintext and uploads the digest for signing by Cloud KMS.

    Args:
      * kms_crypto_key (str) - The name of the cryptographic key, e.g.
        projects/[PROJECT]/locations/[LOC]/keyRings/[KEYRING]/cryptoKeys/[KEY]
      * input_file (Path) - Path to file with data to operate on. Data for sign
        and verify cannot be larger than 64KiB.
      * output_file (Path) - Path to write output signature to a json file.
      * service_account_creds_file (str) - Path to JSON file with service 
        account credentials to use.
    """
    args = [
        self.cloudkms_path,
        'sign',
        '-input',
        input_file,
        '-output',
        output_file,
        kms_crypto_key,
    ]

    if service_account_creds_file:
      args.append('-service-account-json')
      args.append(service_account_creds_file)
    
    self.m.step('sign', args)

  def verify(self,
             kms_crypto_key,
             input_file,
             signature_file,
             output_file='-',
             service_account_creds_file=None):
    """Verify a signature that was previously created with a 
       key stored in CloudKMS.

    Args:
      * kms_crypto_key (str) - The name of the cryptographic public key,
        e.g.
        projects/[PROJECT]/locations/[LOC]/keyRings/[KEYRING]/cryptoKeys/[KEY]
      * input_file (Path) - Path to file with data to operate on. Data for sign
        and verify cannot be larger than 64KiB.
      * signature_file (Path) - Path to read signature from.
      * output_file (Path) - Path to write operation results 
        (successful verification or signature mismatch)to (use '-' for stdout).
      * service_account_creds_file (str) - Path to JSON file with service 
        account credentials to use.
    """
    args = [
        self.cloudkms_path,
        'verify',
        '-input-sig',
        signature_file,
        '-input',
        input_file,
        '-output',
        output_file,
        kms_crypto_key,
    ]

    if service_account_creds_file:
      args.append('-service-account-json')
      args.append(service_account_creds_file)

    self.m.step('verify', args)

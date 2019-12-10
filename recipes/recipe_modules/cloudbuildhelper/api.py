# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""API for calling 'cloudbuildhelper' tool.

See https://chromium.googlesource.com/infra/infra/+/master/build/images/.
"""

from collections import namedtuple

from recipe_engine import recipe_api


class CloudBuildHelperApi(recipe_api.RecipeApi):
  """API for calling 'cloudbuildhelper' tool."""

  # Reference to a docker image uploaded to a registry.
  Image = namedtuple('Image', [
      'image',          # <registry>/<name>
      'digest',         # sha256:...
      'tag',            # its canonical tag, if any
      'view_image_url', # link to GCR page, for humans, optional
      'view_build_url', # link to GCB page, for humans, optional
  ])

  # Used in place of Image to indicate that the image builds successfully, but
  # it wasn't uploaded anywhere.
  #
  # This happens if the target manifest doesn't specify a registry to upload
  # the image to. This is rare.
  NotUploadedImage = Image(None, None, None, None, None)

  def __init__(self, **kwargs):
    super(CloudBuildHelperApi, self).__init__(**kwargs)
    self._cbh_bin = None

  @property
  def command(self):
    """Path to the 'cloudbuildhelper' binary to use.

    If unset on first access, will bootstrap the binary from CIPD. Otherwise
    will return whatever was set previously (useful if 'cloudbuildhelper' is
    part of the checkout already).
    """
    if self._cbh_bin is None:
      cbh_dir = self.m.path['start_dir'].join('cbh')
      ensure_file = self.m.cipd.EnsureFile().add_package(
          'infra/tools/cloudbuildhelper/${platform}', 'latest')
      self.m.cipd.ensure(cbh_dir, ensure_file)
      self._cbh_bin = cbh_dir.join('cloudbuildhelper')
    return self._cbh_bin

  @command.setter
  def command(self, val):
    """Can be used to tell the module to use an existing binary."""
    self._cbh_bin = val

  def report_version(self):
    """Reports the version of cloudbuildhelper tool via the step text.

    Returns:
      None.
    """
    res = self.m.step(
        name='cloudbuildhelper version',
        cmd=[self.command, 'version'],
        stdout=self.m.raw_io.output(),
        step_test_data=lambda: self.m.raw_io.test_api.stream_output('\n'.join([
            'cloudbuildhelper v6.6.6',
            '',
            'CIPD package name: infra/tools/cloudbuildhelper/...',
            'CIPD instance ID:  lTJD7x...',
        ])),
    )
    res.presentation.step_text += '\n' + res.stdout

  def build(self,
            manifest,
            canonical_tag=None,
            build_id=None,
            infra=None,
            labels=None,
            tags=None,
            step_test_image=None):
    """Calls `cloudbuildhelper build <manifest>` interpreting the result.

    Args:
      * manifest (Path) - path to YAML file with definition of what to build.
      * canonical_tag (str) - tag to push the image to if we built a new image.
      * build_id (str) - identifier of the CI build to put into metadata.
      * infra (str) - what section to pick from 'infra' field in the YAML.
      * labels ({str: str}) - labels to attach to the docker image.
      * tags ([str]) - tags to unconditionally push the image to.
      * step_test_image (Image) - image to produce in training mode.

    Returns:
      Image instance or NotUploadedImage if the YAML doesn't specify a registry.

    Raises:
      StepFailure on failures.
    """
    name, _ = self.m.path.splitext(self.m.path.basename(manifest))

    cmd = [self.command, 'build', manifest]
    if canonical_tag:
      cmd += ['-canonical-tag', canonical_tag]
    if build_id:
      cmd += ['-build-id', build_id]
    if infra:
      cmd += ['-infra', infra]
    for k in sorted(labels or {}):
      cmd += ['-label', '%s=%s' % (k, labels[k])]
    for t in (tags or []):
      cmd += ['-tag', t]
    cmd += ['-json-output', self.m.json.output()]

    # Expected JSON output (may be produced even on failures).
    #
    # {
    #   "error": "...",  # error message on errors
    #   "image": {
    #     "image": "registry/name",
    #     "digest": "sha256:...",
    #     "tag": "its-canonical-tag",
    #   },
    #   "view_image_url": "https://...",  # for humans
    #   "view_build_url": "https://...",  # for humans
    # }
    try:
      res = self.m.step(
          name='cloudbuildhelper build %s' % name,
          cmd=cmd,
          step_test_data=lambda: self.test_api.build_success_output(
              step_test_image, name, canonical_tag,
          ),
      )
      if not res.json.output:  # pragma: no cover
        res.presentation.status = self.m.step.FAILURE
        raise recipe_api.InfraFailure(
            'Call succeeded, but didn\'t produce -json-output')
      out = res.json.output
      if not out.get('image'):
        return self.NotUploadedImage
      return self.Image(
          image=out['image']['image'],
          digest=out['image']['digest'],
          tag=out['image'].get('tag'),
          view_image_url=out.get('view_image_url'),
          view_build_url=out.get('view_build_url'),
      )
    finally:
      self._make_step_pretty(self.m.step.active_result, tags)

  @staticmethod
  def _make_step_pretty(r, tags):
    js = r.json.output
    if not js or not isinstance(js, dict):  # pragma: no cover
      return

    if js.get('view_image_url'):
      r.presentation.links['image'] = js['view_image_url']
    if js.get('view_build_url'):
      r.presentation.links['build'] = js['view_build_url']

    if js.get('error'):
      r.presentation.step_text += '\nERROR: %s' % js['error']
    elif js.get('image'):
      img = js['image']
      tag = img.get('tag') or (tags[0] if tags else None)
      if tag:
        ref = '%s:%s' % (img['image'], tag)
      else:
        ref = '%s@%s' % (img['image'], img['digest'])
      lines = [
          '',
          'Image: %s' % ref,
          'Digest: %s' % img['digest'],
      ]
      # Display all added tags (including the canonical one we got via `img`).
      for t in set((tags or [])+([img['tag']] if img.get('tag') else [])):
        lines.append('Tag: %s' % t)
      r.presentation.step_text += '\n'.join(lines)
    else:
      r.presentation.step_text += '\nImage builds successfully'

  def update_pins(self, path):
    """Calls `cloudbuildhelper pins-update <path>`.

    Updates the file at `path` in place if some docker tags mentioned there have
    moved since the last pins update.

    Args:
      * path (Path) - path to a `pins.yaml` file to update.

    Returns:
      List of strings with updated "<image>:<tag>" pairs, if any.
    """
    res = self.m.step(
        'cloudbuildhelper pins-update',
        [
            self.command, 'pins-update', path,
            '-json-output', self.m.json.output(),
        ],
        step_test_data=lambda: self.test_api.update_pins_output(
            updated=['some_image:tag'],
        ),
    )
    return res.json.output.get('updated') or []

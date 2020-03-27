# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from hashlib import sha256

from recipe_engine import recipe_test_api


class CloudBuildHelperTestApi(recipe_test_api.RecipeTestApi):
  def build_success_output(self, image, target='target', canonical_tag=None):
    if not image:
      img = 'example.com/fake-registry/%s' % target
      digest = 'sha256:'+sha256(target).hexdigest()[:16]+'...'
      tag = canonical_tag
      if tag == ':inputs-hash':
        tag = 'cbh-inputs-deadbead...'
    else:
      img = image.image
      digest = image.digest
      tag = image.tag

    out = {'view_build_url': 'https://example.com/build/%s' % target}
    if img:
      out['image'] = {'image': img, 'digest': digest, 'tag': tag}
      out['view_image_url'] = 'https://example.com/image/%s' % target

    return self.m.json.output(out)

  def build_error_output(self, message, target='target'):
    return self.m.json.output({
      'error': message,
      'view_build_url': 'https://example.com/build/%s' % target,
    })

  def upload_success_output(self, tarball, target='target', canonical_tag=None):
    if not tarball:
      name = 'example/%s' % target
      digest = sha256(name).hexdigest()[:16]+'...'
      bucket = 'example'
      path = 'tarballs/example/%s/%s.tar.gz' % (target, digest)
      tag = canonical_tag or '11111-deadbeef'
    else:
      name = tarball.name
      digest = tarball.sha256
      bucket = tarball.bucket
      path = tarball.path
      tag = tarball.version
    return self.m.json.output({
      'name': name,
      'sha256': digest,
      'gs': {
        'bucket': bucket,
        'name': path,
      },
      'canonical_tag': tag,
    })

  def upload_error_output(self, message):
    return self.m.json.output({'error': message})

  def update_pins_output(self, updated):
    return self.m.json.output({'updated': updated or []})

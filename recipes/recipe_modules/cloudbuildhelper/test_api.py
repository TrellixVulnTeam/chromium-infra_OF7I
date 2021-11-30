# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from hashlib import sha256

from recipe_engine import recipe_test_api


class CloudBuildHelperTestApi(recipe_test_api.RecipeTestApi):
  def build_success_output(self, image, target='target',
                           canonical_tag=None,
                           checkout_metadata=None,
                           mocked_sources=None):
    if not image:
      img = 'example.com/fake-registry/%s' % target
      digest = 'sha256:' + sha256(
          target.encode('utf-8')).hexdigest()[:16] + '...'
      tag = canonical_tag
      if tag == ':inputs-hash':
        tag = 'cbh-inputs-deadbead...'
      context_dir = None
      sources = None
    else:
      img = image.image
      digest = image.digest
      tag = image.tag
      context_dir = image.context_dir
      sources = image.sources

    sources_json = mocked_sources
    if sources_json is None and sources and checkout_metadata:
      sources_json = self._derive_sources_for_json(sources, checkout_metadata)

    out = {
        'view_build_url': 'https://example.com/build/%s' % target,
        'context_dir': context_dir or '/some/context/directory/for/%s' % target,
        'sources': sources_json or [],
    }
    if img:
      out['image'] = {'image': img, 'digest': digest, 'tag': tag}
      out['view_image_url'] = 'https://example.com/image/%s' % target
      if image and image.notify:
        out['notify'] = [
            {'kind': n.kind, 'repo': n.repo, 'script': n.script}
            for n in image.notify
        ]

    return self.m.json.output(out)

  def build_error_output(self, message, target='target'):
    return self.m.json.output({
        'error': message,
        'view_build_url': 'https://example.com/build/%s' % target,
    })

  def upload_success_output(self, tarball, target='target',
                            canonical_tag=None,
                            checkout_metadata=None,
                            mocked_sources=None):
    if not tarball:
      name = 'example/%s' % target
      digest = sha256(name.encode('utf-8')).hexdigest()[:16] + '...'
      bucket = 'example'
      path = 'tarballs/example/%s/%s.tar.gz' % (target, digest)
      tag = canonical_tag or '11111-deadbeef'
      sources = None
    else:
      name = tarball.name
      digest = tarball.sha256
      bucket = tarball.bucket
      path = tarball.path
      tag = tarball.version
      sources = tarball.sources

    sources_json = mocked_sources
    if sources_json is None and sources and checkout_metadata:
      sources_json = self._derive_sources_for_json(sources, checkout_metadata)

    return self.m.json.output({
        'name': name,
        'sha256': digest,
        'gs': {
          'bucket': bucket,
          'name': path,
        },
        'canonical_tag': tag,
        'sources': sources_json or [],
    })

  def upload_error_output(self, message):
    return self.m.json.output({'error': message})

  def update_pins_output(self, updated):
    return self.m.json.output({'updated': updated or []})

  @staticmethod
  def _derive_sources_for_json(expected_output, checkout_metadata):
    # Build a reverse map: repo URL => checkout directory.
    repo_paths = {
        entry['repository']:
            checkout_metadata.root.join(path)
            if path != '.' else checkout_metadata.root
        for path, entry in checkout_metadata.repos.items()
    }

    # Use it to figure out source dirs that should appear in the -json-output
    # to eventually be transformed into `expected_output`.
    paths = []
    for repo in expected_output:
      repo_path = repo_paths.get(repo['repository'])
      if not repo_path:  # pragma: no cover
        raise ValueError(
            'Repository %s is not part of the mocked checkout %r' %
            (repo['repository'], checkout_metadata))
      for src in repo.get('sources') or []:
        paths.append(repo_path.join(src) if src != '.' else repo_path)

    # JSON should have only strings, not Paths.
    return [str(p) for p in paths]

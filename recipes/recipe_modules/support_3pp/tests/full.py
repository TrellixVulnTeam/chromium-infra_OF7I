# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import post_process
from recipe_engine.recipe_api import Property

DEPS = [
  'recipe_engine/buildbucket',
  'recipe_engine/cipd',
  'recipe_engine/file',
  'recipe_engine/json',
  'recipe_engine/path',
  'recipe_engine/platform',
  'recipe_engine/properties',
  'recipe_engine/raw_io',
  'recipe_engine/step',

  'support_3pp',
]

KEY_PATH = ('projects/chops-kms/locations/global/keyRings/'
            'chrome-official/cryptoKeys/infra-signing-key/'
            'cryptoKeyVersions/1')

PROPERTIES = {
  'GOOS': Property(),
  'GOARCH': Property(),
  'experimental': Property(kind=bool, default=False),
  'load_dupe': Property(kind=bool, default=False),
  'package_prefix': Property(default='3pp'),
  'source_cache_prefix': Property(default='sources'),
}

def RunSteps(api, GOOS, GOARCH, experimental, load_dupe, package_prefix,
             source_cache_prefix):
  # set a cache directory to be similar to what the actual 3pp recipe does.
  # TODO(iannucci): just move the 3pp recipe into the recipe_module here...
  with api.cipd.cache_dir(api.path.mkdtemp()):
    builder = api.path['cache'].join('builder')
    api.support_3pp.set_package_prefix(package_prefix)
    api.support_3pp.set_source_cache_prefix(source_cache_prefix)
    api.support_3pp.set_experimental(experimental)

    api.step('echo package_prefix', ['echo', api.support_3pp.package_prefix()])

    # do a checkout in `builder`
    pkgs = api.support_3pp.load_packages_from_path(builder.join('package_repo'))

    if 'build_tools/tool' in pkgs:
      # For the test, also explicitly build 'build_tools/tool@1.5.0-rc1',
      # which should de-dup with the default build_tools/tool@latest.
      pkgs.add('build_tools/tool@1.5.0-rc1')

    # doing it twice should raise a DuplicatePackage exception
    if load_dupe:
      api.support_3pp.load_packages_from_path(
        builder.join('dup_repo'))

    _, unsupported = api.support_3pp.ensure_uploaded(
      pkgs, '%s-%s' % (GOOS, GOARCH))

    excluded = set()
    if 'unsupported' in pkgs:
      excluded.add('unsupported')
    if 'unsupported_no_method' in pkgs:
      excluded.add('unsupported_no_method')
    if api.platform.is_win and 'tools/posix_tool' in pkgs:
      excluded.add('tools/posix_tool')
    assert unsupported == excluded, (
        'Expected: %r. Got: %r' %(excluded, unsupported))

    # doing it again should hit caches
    api.support_3pp.ensure_uploaded(pkgs, '%s-%s' % (GOOS, GOARCH))


def GenTests(api):
  pkgs_dict = {}
  pkgs_dict['dir_deps/bottom_dep_url'] = '''
  create {
    source { url {
        download_url: "https://some.internet.example.com"
        version: "1.2.3"
    } }
    build {}
  }
  upload { pkg_prefix: "deps" }
  '''

  pkgs_dict['dir_deps/bottom_dep_git'] = '''
  create {
    source { git {
        repo: "https://chromium.googlesource.com/external/go.repo/dep"
        tag_pattern: "v%s"
    } }
    build {}
  }
  upload { pkg_prefix: "deps" }
  '''

  pkgs_dict['dir_build_tools/tool'] = '''
  create {
    source {
      git {
        repo: "https://go.repo/tool"
        tag_pattern: "v%s"
        version_join: "."
      }
      subdir: "src/go.repo/tool"
      patch_dir: "patches"
      patch_version: "chops.1"
    }
    build {
      # We use an older version of the tool to bootstrap new versions.
      tool: "build_tools/tool@0.9.0"
      dep: "deps/bottom_dep_url"
      dep: "deps/bottom_dep_git"

      install: "install.sh"
      install: "intel"
    }
    package {
      version_file: ".versions/tool.cipd_version"
    }
  }

  create {
    platform_re: "mac-.*"
    build {
      install: "install-mac.sh"
    }
    package {
      install_mode: symlink
    }
    verify {
      test: "test.py"
      test: "mac"
    }
  }

  create {
    platform_re: "windows-.*"
    verify {
      test: "test.py"
      test: "windows"
    }
  }

  create {
    platform_re: "linux-.*"
    verify {
      test: "test.py"
      test: "linux"
    }
  }

  create {
    platform_re: "linux-arm.*"
    build {
      install: "install.sh"
      install: "arm"
    }
  }

  create {
    platform_re: "linux-amd64"
    build {
      # on linux-amd64 we self-bootstrap the tool
      tool: ""  # clears tool@0.9.0
      install: "install_bootstrap.sh"
    }
  }

  upload { pkg_prefix: "build_tools" }
  '''

  pkgs_dict['dir_build_tools/git_tool'] = '''
  create {
    source {
      git {
        repo: "https://chromium.googlesource.com/external/go.repo/git_tool"
        tag_pattern: "v%s"
        version_join: "."
      }
      subdir: "src/go.repo/tool"
      patch_dir: "patches"
      patch_version: "chops.1"
    }
    build {
      # We use an older version of the tool to bootstrap new versions.
      tool: "build_tools/tool@0.9.0"
      dep: "deps/bottom_dep_url"
      dep: "deps/bottom_dep_git"

      install: "install.sh"
      install: "intel"
    }
    package {
      version_file: ".versions/tool.cipd_version"
    }
  }

  create {
    platform_re: "mac-.*"
    build {
      install: "install-mac.sh"
    }
    package {
      install_mode: symlink
    }
    verify {
      test: "test.py"
      test: "mac"
    }
  }

  create {
    platform_re: "windows-.*"
    verify {
      test: "test.py"
      test: "windows"
    }
  }

  create {
    platform_re: "linux-.*"
    verify {
      test: "test.py"
      test: "linux"
    }
  }

  create {
    platform_re: "linux-arm.*"
    build {
      install: "install.sh"
      install: "arm"
    }
  }

  create {
    platform_re: "linux-amd64"
    build {
      # on linux-amd64 we self-bootstrap the tool
      tool: ""  # clears tool@0.9.0
      install: "install_bootstrap.sh"
    }
  }

  upload { pkg_prefix: "build_tools" }
  '''

  pkgs_dict['dir_deps/deep_dep'] = '''
  create {
    source { cipd {
      pkg: "source/deep_dep"
      default_version: "1.0.0"
      original_download_url: "https://some.internet.example.com"
    } }
  }
  upload { pkg_prefix: "deps" }
  '''

  pkgs_dict['dir_deps/dep'] = '''
  create {
    source { cipd {
      pkg: "source/dep"
      default_version: "1.0.0"
      original_download_url: "https://some.internet.example.com"
    } }
    build {
      tool: "build_tools/tool"
      dep: "deps/deep_dep"
    }
  }
  upload { pkg_prefix: "deps" }
  '''

  pkgs_dict['dir_tools/pkg'] = '''
  create {
    source { script { name: "fetch.py" } }
    build {
      tool: "build_tools/tool"
      dep:  "deps/dep"
    }
  }
  upload { pkg_prefix: "tools" }
  '''

  pkgs_dict['dir_tools/pkg_checkout'] = '''
  create {
    source { script {
      name: "fetch.py"
      use_fetch_checkout_workflow: true
    } }
    build {
      tool: "build_tools/tool"
      dep:  "deps/dep"
    }
  }
  upload { pkg_prefix: "tools" }
  '''

  pkgs_dict['unsupported'] = '''
  create { unsupported: true }
  '''

  pkgs_dict['unsupported_no_method'] = '''
  create { verify { test: "verify.py" } }
  '''

  pkgs_dict['dir_tools/windows_experiment'] = r'''
  create {
    platform_re: "linux-.*|mac-.*"
    source { script { name: "fetch.py" } }
    build {}
  }

  create {
    platform_re: "windows-.*"
    experimental: true
    source { script { name: "fetch.py" } }
    build {
      install: "win_install.py"
    }
    package {
      alter_version_re: "(.*)\.windows\.\d*(.*)"
      alter_version_replace: "\\1\\2"
    }
  }

  upload { pkg_prefix: "tools" }
  '''

  pkgs_dict['dir_tools/posix_tool'] = '''
  create {
    platform_re: "linux-.*|mac-.*"
    source {
      cipd {
        pkg: "source/posix_tool"
        default_version: "1.2.0"
        original_download_url: "https://some.internet.example.com"
      }
      unpack_archive: true
    }
    build {}  # default build options
  }
  upload { pkg_prefix: "tools" }
  '''

  pkgs_dict['dir_tools/already_uploaded'] = '''
  create {
    source { cipd {
      pkg: "source/already_uploaded"
      default_version: "1.5.0-rc1"
      original_download_url: "https://some.internet.example.com"
    } }
  }
  upload { pkg_prefix: "tools" }
  '''

  # This doesn't have a 'build' step. It just fetches something (e.g. gcloud
  # SDK), and then re-uploads it.
  pkgs_dict['dir_fetch/fetch_and_package'] = '''
  create { source { script { name: "fetch.py" } } }
  upload {
    pkg_prefix: "tools"
    universal: true
  }
  '''
  pkgs = sorted(pkgs_dict.items())

  def mk_name(*parts):
    return '.'.join(parts)

  for goos, goarch in (('linux', 'amd64'),
                       ('linux', 'armv6l'),
                       ('windows', 'amd64'),
                       ('mac', 'amd64')):
    plat_name = 'win' if goos == 'windows' else goos

    sep = '\\' if goos == 'windows' else '/'
    pkg_repo_path = sep.join(['%s', '3pp.pb'])
    plat = '%s-%s' % (goos, goarch)

    test = (api.test('integration_test_%s-%s' % (goos, goarch))
      + api.platform(plat_name, 64)  # assume all hosts are 64 bits.
      + api.properties(GOOS=goos, GOARCH=goarch)
      + api.properties(key_path=KEY_PATH)
      + api.buildbucket.ci_build()
      + api.step_data('find package specs', api.file.glob_paths([
          pkg_repo_path % sep.join(pkg_dir.split('/')) for pkg_dir, _ in pkgs]))
      + api.override_step_data(mk_name(
        'building tools/already_uploaded',
        'cipd describe 3pp/tools/already_uploaded/%s' % plat
      ), api.cipd.example_describe(
        '3pp/tools/already_uploaded/%s' % plat,
        version='version:1.5.0-rc1', test_data_tags=['version:1.5.0-rc1']))
    )

    if plat_name != 'win':
      # posix_tool says it needs an archive unpacking.
      test += api.step_data(mk_name(
        'building tools/posix_tool', 'fetch sources', 'unpack_archive',
        'find archive to unpack',
      ), api.file.glob_paths(['archive.tgz']))
    else:
      test += api.step_data(mk_name(
          'building tools/windows_experiment', 'fetch.py latest',
      ), stdout=api.raw_io.output('2.0.0.windows.1'))

    for pkg_path, spec in pkgs:
      pkg_spec_dir = pkg_repo_path % sep.join(pkg_path.split('/'))
      test += api.step_data(
          mk_name("load package specs", "read %r" % pkg_spec_dir),
          api.file.read_text(spec))
    yield test

  # Test pkg_name when specs are not under "3pp" directory
  yield (api.test('load-spec')
      + api.properties(GOOS='linux', GOARCH='amd64')
      + api.step_data(
          'find package specs',
          # Since glob_paths test api sorts the file names, names the test data
          # so that 0bar comes before 3pp.pb
          api.file.glob_paths(['0bar/3pp.pb', '3pp.pb']))
      + api.step_data(
          mk_name("load package specs", "read '0bar/3pp.pb'"),
          api.file.read_text('create {} upload {pkg_prefix: "p_0bar"}'))
      + api.post_process(
          post_process.MustRun,
          mk_name("load package specs", "Compute hash for 'p_0bar/0bar'"))
      + api.expect_exception('Exception')
      + api.post_process(post_process.ResultReasonRE,
                         'Expecting the spec PB in a deeper folder')
      + api.post_process(post_process.StatusException)
      + api.post_process(post_process.DropExpectation)
  )

  # Test pkg_name when specs are under "3pp" directory
  yield (api.test('load-spec-3pp-dir')
      + api.properties(GOOS='linux', GOARCH='amd64')
      + api.step_data(
          'find package specs',
          # Since glob_paths test api sorts the file names, names the test data
          # so that 0bar comes before 3pp.pb
          api.file.glob_paths(['0bar/3pp/3pp.pb', '3pp/3pp.pb']))
      + api.step_data(
          mk_name("load package specs", "read '0bar/3pp/3pp.pb'"),
          api.file.read_text('create {} upload {pkg_prefix: "p_0bar"}'))
      + api.post_process(
          post_process.MustRun,
          mk_name("load package specs", "Compute hash for 'p_0bar/0bar'"))
      + api.expect_exception('Exception')
      + api.post_process(post_process.ResultReasonRE,
                         'Expecting the spec PB in a deeper folder')
      + api.post_process(post_process.StatusException)
      + api.post_process(post_process.DropExpectation)
  )

  yield (api.test('empty-spec')
      + api.properties(GOOS='linux', GOARCH='amd64')
      + api.step_data('find package specs',
                      api.file.glob_paths(['empty/3pp.pb']))
      + api.post_process(post_process.StatusFailure)
      + api.post_process(post_process.DropExpectation)
  )

  yield (api.test('bad-spec')
      + api.properties(GOOS='linux', GOARCH='amd64')
      + api.step_data(
          'find package specs',
          api.file.glob_paths(['bad/3pp.pb']))
         + api.step_data(mk_name("load package specs", "read 'bad/3pp.pb'"),
                      api.file.read_text('narwhal'))
      + api.expect_exception('BadParse')
      + api.post_process(post_process.StatusException)
      + api.post_process(post_process.DropExpectation)
  )

  yield (api.test('duplicate-load')
      + api.properties(GOOS='linux', GOARCH='amd64', load_dupe=True)
      + api.step_data(
          'find package specs',
          api.file.glob_paths(['something/3pp.pb']))
      + api.step_data(
          mk_name("load package specs", "read 'something/3pp.pb'"),
          api.file.read_text('create {} upload {pkg_prefix: "p_something"}'))
      + api.post_process(
          post_process.MustRun,
          mk_name("load package specs",
                  "Compute hash for 'p_something/something'"))
      + api.step_data(
          'find package specs (2)',
          api.file.glob_paths(['path/something/3pp.pb']))
      + api.step_data(
          mk_name("load package specs (2)", "read 'path/something/3pp.pb'"),
          api.file.read_text('create {} upload {pkg_prefix: "p_something"}'))
      + api.expect_exception('DuplicatePackage')
      + api.post_process(post_process.StatusException)
      + api.post_process(post_process.DropExpectation)
  )

  dep = '''
  create {
    platform_re: "linux-amd64|mac-.*"
    source { git {
        repo: "https://go.repo/dep"
        tag_pattern: "v%s"
      } }
    build {}
  }
  upload { pkg_prefix: "pkg" }
  '''

  tool = '''
  create {
    platform_re: "linux-amd64|mac-.*"
    source { git {
        repo: "https://go.repo/tool"
        version_restriction { op: LT val: "1.5rc" }
        version_restriction { op: GE val: "1.4" }
      } }
    build { tool: "pkg/dep" }
  }
  upload { pkg_prefix: "build_tools" }
  '''
  yield (api.test('building-package-failed')
      + api.properties(GOOS='linux', GOARCH='amd64')
      + api.properties(key_path=KEY_PATH)
      + api.step_data(
          'find package specs',
          api.file.glob_paths(['%s/3pp.pb' % pkg for pkg in ['dep', 'tool']]))
      + api.step_data(
          mk_name("load package specs", "read 'dep/3pp.pb'"),
          api.file.read_text(dep))
      + api.step_data(
          mk_name("load package specs", "read 'tool/3pp.pb'"),
          api.file.read_text(tool))
      + api.step_data(
          mk_name(
              'building pkg/dep', 'run installation',
              'install.sh '
              '[START_DIR]/3pp/wd/pkg/dep/linux-amd64/1.5.0-rc1/out '
              '[START_DIR]/3pp/wd/pkg/dep/linux-amd64/1.5.0-rc1/deps_prefix'),
          retcode=1)
      + api.override_step_data(
          mk_name('building build_tools/tool',
                  'fetch sources',
                  'installing tools',
                  'building pkg/dep',
                  'cipd describe 3pp/pkg/dep/linux-amd64'),
          api.cipd.example_describe(
          '3pp/pkg/dep/linux-amd64',
          version='version:1.5.0-rc1', test_data_tags=['version:1.5.0-rc1']),
      )
  )

  # test for git tag movement.
  pkg = 'dir_deps/bottom_dep_git'
  yield (api.test('catch-git-tag-movement')
      + api.properties(GOOS='linux', GOARCH='amd64', use_new_checkout=True)
      + api.step_data('find package specs',
                      api.file.glob_paths(['%s/3pp.pb' % pkg]))
      + api.step_data(
          mk_name("load package specs", "read '%s/3pp.pb'" % pkg),
          api.file.read_text(pkgs_dict[pkg]))
      + api.override_step_data(
          mk_name('building deps/bottom_dep_git',
                  'fetch sources',
                  'cipd describe 3pp/sources/git/go.repo/dep'),
        api.cipd.example_describe(
        '3pp/sources/git/go.repo/dep',
        version='version:1.5.0-rc1',
        test_data_tags=['version:1.5.0-rc1','external_hash:deadbeef']),
      )
      + api.expect_exception('AssertionError')
         + api.post_process(post_process.ResultReasonRE,
                            'External hash verification failed')
      + api.post_process(post_process.StepFailure,
                         mk_name('building deps/bottom_dep_git',
                                 'fetch sources',
                                 'Verify External Hash'))
      + api.post_process(post_process.StatusException)
      + api.post_process(post_process.DropExpectation)
  )

  yield (api.test('ambiguous-version-tag') +
         api.properties(GOOS='linux', GOARCH='amd64', use_new_checkout=True) +
         api.step_data('find package specs',
                       api.file.glob_paths(['%s/3pp.pb' % pkg])) +
         api.step_data(
             mk_name("load package specs", "read '%s/3pp.pb'" % pkg),
             api.file.read_text(pkgs_dict[pkg])) + api.step_data(
                 mk_name('building deps/bottom_dep_git',
                         'cipd describe 3pp/deps/bottom_dep_git/linux-amd64'),
                 api.json.output({
                     "error": ("ambiguity when resolving the tag, " +
                               "more than one instance has it"),
                     "result": None
                 }),
                 retcode=1))

  # test for source url package.
  pkg = 'dir_deps/bottom_dep_url'
  yield (api.test('source-url-test')
      + api.properties(GOOS='linux', GOARCH='amd64', use_new_checkout=True)
      + api.step_data('find package specs',
                      api.file.glob_paths(['%s/3pp.pb' % pkg]))
      + api.step_data(
          mk_name("load package specs", "read '%s/3pp.pb'" % pkg),
          api.file.read_text(pkgs_dict[pkg]))
      + api.post_process(
          post_process.MustRun,
          mk_name("building deps/bottom_dep_url",
                  "fetch sources",
                  "GET https://some.internet.example.com"))
      + api.post_process(post_process.StatusSuccess)
      + api.post_process(post_process.DropExpectation)
  )

  yield (api.test('empty-source_cache_prefix')
      + api.properties(GOOS='linux', GOARCH='amd64', source_cache_prefix='',
                       use_new_checkout=True)
      + api.expect_exception('AssertionError')
      + api.post_process(post_process.ResultReasonRE,
                         'A non-empty source cache prefix is required.')
      + api.post_process(post_process.StatusException)
      + api.post_process(post_process.DropExpectation)
  )

  yield (api.test('experimental-mode')
      + api.properties(GOOS='linux', GOARCH='amd64', experimental=True)
      + api.post_process(post_process.StepCommandRE, 'echo package_prefix',
                         ['echo', 'experimental/3pp'])
      + api.post_process(post_process.StatusSuccess)
      + api.post_process(post_process.DropExpectation)
  )

  yield (api.test('empty-package_prefix-in-experimental-mode')
      + api.properties(GOOS='linux', GOARCH='amd64', experimental=True,
                       package_prefix='')
      + api.post_process(post_process.StepCommandRE, 'echo package_prefix',
                         ['echo', 'experimental'])
      + api.post_process(post_process.StatusSuccess)
      + api.post_process(post_process.DropExpectation)
  )

  # Tools may need to depend on themselves for a host version to use when
  # cross-compiling.
  spec = '''
  create {
    source { script { name: "fetch.py" } }
  }
  create {
    platform_re: "linux-arm.*"
    build { tool: "tools/self_dependency" }
  }
  upload { pkg_prefix: "tools" }
  '''
  yield (api.test('cross-compile-self-dep') + api.platform('linux', 64) +
         api.properties(GOOS='linux', GOARCH='arm64')
         + api.properties(key_path=KEY_PATH) + api.step_data(
             'find package specs',
             api.file.glob_paths(['dir_build_tools/self_dependency/3pp.pb'])) +
         api.step_data(
             mk_name("load package specs",
                     "read 'dir_build_tools/self_dependency/3pp.pb'"),
             api.file.read_text(spec)))

  def links_include(check, step_odict, step, link_name):
    check(
        'step result for %s contained link named %s' % (step, link_name),
        link_name in step_odict[step].links)

  yield (api.test('link-in-do-upload')
      + api.properties(GOOS='linux', GOARCH='amd64', use_new_checkout=True)
      + api.step_data('find package specs',
                      api.file.glob_paths(['dir_deps/bottom_dep_git/3pp.pb',
                                           'dir_deps/bottom_dep_url/3pp.pb']))
      + api.step_data(mk_name("load package specs",
                              "read 'dir_deps/bottom_dep_git/3pp.pb'"),
                      api.file.read_text(pkgs_dict['dir_deps/bottom_dep_git']))
      + api.step_data(mk_name("load package specs",
                              "read 'dir_deps/bottom_dep_url/3pp.pb'"),
                      api.file.read_text(pkgs_dict['dir_deps/bottom_dep_url']))
      + api.override_step_data(
          mk_name('building deps/bottom_dep_git',
                  'do upload',
                  'register 3pp/deps/bottom_dep_git/linux-amd64'),
          api.json.output({
              'result': {
                  'instance_id': 'instance-id-bottom_dep_git',
                  'package': '3pp/deps/bottom_dep_git/linux-amd64'
              },
          }))
      + api.post_process(links_include,
                         mk_name('building deps/bottom_dep_git', 'do upload'),
                         'instance-id-bottom_dep_git')
      + api.override_step_data(
          mk_name('building deps/bottom_dep_url',
                  'do upload',
                  'cipd describe 3pp/deps/bottom_dep_url/linux-amd64'),
          api.json.output({
              'result': {
                  'pin': {
                      'instance_id': 'instance-id-bottom_dep_url',
                      'package': '3pp/deps/bottom_dep_url/linux-amd64'
                  },
                  'registered_by': 'user_a',
                  'registered_ts': 1234,
              },
          }))
      + api.post_process(links_include,
                         mk_name('building deps/bottom_dep_url', 'do upload'),
                         'instance-id-bottom_dep_url')
      + api.post_process(post_process.StatusSuccess)
      + api.post_process(post_process.DropExpectation)
  )

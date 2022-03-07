# 3pp package definitions

This is a collection of "third-party package" definitions.

See the [support_3pp] recipe module docs for the format of package
definitions.

[support_3pp]: /recipes/README.recipes.md#recipe_modules-support_3pp

# Writing and testing new package definitions

To go along with the support_3pp documentation above, this section will
give some higher-level advice on writing package definitions.

## Package requirements

3pp packages are designed to be hermetic, meaning that they should typically
not depend on any other 3pp package at runtime. This means that any library
dependencies other than the core system libraries (more on this below) should
be statically linked. This may require giving appropriate options to a
configure script, or in some more extreme cases, patching the build system.

3pp packages are also designed to be relocatable, meaning that they can be
deployed to any location on disk. Be aware if the package hardcodes any
paths into the compiled code or into configuration files; if so, you may
need to patch it to avoid this.

### System libraries

Packages may depend on system libraries. Currently, we target the following:

* Linux: [manylinux2014](https://www.python.org/dev/peps/pep-0599/) for
         linux-amd64. For other Linux targets, we follow
         [dockcross](https://github.com/dockcross/dockcross) latest.
* macOS: Version 10.11 (mac-amd64), Version 11.0 (mac-arm64)
* Windows: Windows 7

## Naming conventions

For infra 3pp, we use the `static_libs` prefix for static libraries, and the
`tools` prefix for executables. `build_support` is used infrequently; this is
for packages which are for consumption only by the 3pp build system itself.

## Versioning

Package version tags in CIPD are immutable. Therefore, in order to trigger a
new build of a package, the version number must change. If the upstream
(source) version of a package is staying the same, but you are making a change
to the build script/environment or applying patches, add (or increment) the
patch_version field in the spec to give it a new version string.

## Platforms

3pp supports multiple architectures of Windows, Mac, and Linux. It is generally
best to use platform_re to only build packages on platforms where they are
actually needed. This reduces the amount of time spent debugging failures,
both now and later.

## Testing

### Try jobs
The preferred way to test package definitions is to upload your CL to
Gerrit and do a CQ dry run. This will trigger the 3pp try builders which
run on all platforms. The try jobs do not upload the built packages anywhere,
but you can inspect the build status to see which files would be packaged,
and any error messages.

### Building stuff locally

See [./run_locally.sh](./run_locally.sh). You can pass `help` as the first
argument for the lowdown.

For Linux, run_locally.sh requires docker to be installed. For googlers, please
refer to go/docker.

Building the package locally will allow you to actually inspect the package
output, if needed, as well as upload it to the experimental/ prefix in CIPD.

### run_remotely

[./run_remotely.sh](./run_remotely.sh) is another option which works similar to
the try jobs, but gives you more control over the Swarming task definition.
You may want to use run_remotely if you want to tweak recipe code or properties.

# CIPD Sources

If possible, prefer to use git, url, or script methods. If none of these
are workable for a package, cipd source may be used.

Some third-party packages distribute their releases via source tarballs or zips.
Sometimes this is done via http or ftp.

To ingest a new tarball/zip:
  * Download the official tarball release from the software site.
    * pick one that is compressed with gzip, bzip2, xz, zstd, or is a zip file.
    * If there's no such tarball, consider expanding compression support
      in the `recipe_engine/archive` module.
  * Put the tarball in an empty directory by itself (don't unpack it). The
    name of the archive doesn't matter. Your directory should now look like:

      some/dir/
          pkgname-1.2.3.tar.gz

  * Now run:

      $ PKG_NAME=pkgname
      $ VERSION=1.2.3
      $ cipd create  \
        -in some/dir \
        -name infra/third_party/source/$PKG_NAME \
        -tag version:$VERSION

  * You can now use the source in a 3pp package definition like:

      source {
        cipd {
          pkg: "infra/third_party/source/pkgname"
          default_version: "1.2.3"
          original_download_url: "https://original.source.url.example.com"
        }
        # Lets 3pp recipe know to expect a single tarball/zip
        unpack_archive: true
      }

  * By default the 3pp recipe also expects unpacked archives to unpack their
    actual contents (files) to a subdirectory (in the Unix world this is typical
    for tarballs to have all files under a folder named the same thing as the
    tarball itself). The 3pp recipe will remove these 'single directories' and
    move all contents to the top level directory. To avoid this behavior, see
    the `no_archive_prune` option.

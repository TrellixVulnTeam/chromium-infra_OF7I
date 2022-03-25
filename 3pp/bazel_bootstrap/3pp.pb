# For Linux and Mac the download is a self-contained executable, not
# packaged in a tarball or zipfile. For Windows the download is a
# zipfile containing a self-contained executable. This explains the
# different configs for the different platforms.

create {
  platform_re: "(linux|mac)-(amd64|arm64)"
  source {
    script { name: "fetch.py" }
    unpack_archive: false
    cpe_base_address: "cpe:/a:google:bazel"
  }

  build {}

  package {
    version_file: ".versions/bazel_bootstrap.version"
  }
}

create {
  platform_re: "windows-(amd64|arm64)"
  source {
    script { name: "fetch.py" }
    unpack_archive: true
    cpe_base_address: "cpe:/a:google:bazel"
  }

  package {
    version_file: ".versions/bazel_bootstrap.version"
  }
}

upload { pkg_prefix: "tools" }

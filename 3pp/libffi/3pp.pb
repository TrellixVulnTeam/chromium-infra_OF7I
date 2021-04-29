create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://github.com/libffi/libffi/archive/v3.2.1.tar.gz"
      version: "3.2.1"
    }
    patch_version: "chromium.1"
    unpack_archive: true
  }
  build {
    tool: "tools/autoconf"
    tool: "tools/automake"
    tool: "tools/libtool"
    tool: "tools/texinfo"
    tool: "tools/sed"
  }
}

create {
  platform_re: "mac-.*"
  source {
    patch_dir: "mac_patches"
  }
}

upload { pkg_prefix: "static_libs" }

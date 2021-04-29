create {
  verify { test: "python_test.py" }
  source { patch_version: "chromium.32" }
}

create {
  platform_re: "linux-.*|mac-.*"
  source {
    # Python 2 is officially done, and 2.7.18 is the last official release.
    url {
      download_url: "https://www.python.org/ftp/python/2.7.18/Python-2.7.18.tgz"
      version: "2.7.18",
      extension: ".tgz"
    }
    unpack_archive: true
    patch_dir: "patches"
  }
  build {
    tool: "build_support/pip_bootstrap"
    tool: "tools/autoconf"
    tool: "tools/sed"            # Used by python's makefiles
  }
}

create {
  platform_re: "mac-.*"
  source {
    patch_dir: "patches"
    patch_dir: "mac_patches"
  }
  build {
    dep: "static_libs/bzip2"
    dep: "static_libs/ncurses"
    dep: "static_libs/openssl"
    dep: "static_libs/readline"
    dep: "static_libs/sqlite"
    dep: "static_libs/zlib"
  }
}

create {
  platform_re: "linux-.*"
  build {
    dep: "static_libs/bzip2"
    dep: "static_libs/ncurses"
    dep: "static_libs/openssl"
    dep: "static_libs/readline"
    dep: "static_libs/sqlite"
    dep: "static_libs/zlib"

    # On Linux, we need to explicitly build libnsl; on other platforms, it is
    # part of 'libc'.
    dep: "static_libs/nsl"
  }
}

create {
  platform_re: "windows-.*"
  source { script { name: "fetch.py" } }
  build {
    tool: "build_support/pip_bootstrap"
    tool: "tools/lessmsi"

    install: "install_win.sh"
  }
  verify { test: "python_test.py" }
}

upload { pkg_prefix: "tools" }

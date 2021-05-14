create {
  platform_re: "linux-.*|mac-.*",
  source {
    url {
      download_url: "https://ftp.gnu.org/pub/gnu/libtool/libtool-2.4.6.tar.gz"
      version: "2.4.6"
    }
    unpack_archive: true
    patch_dir: "patches"
    patch_version: "chromium.3"
  }
  build {
    tool: "tools/help2man"
    tool: "tools/sed"
  }
}

upload { pkg_prefix: "tools" }

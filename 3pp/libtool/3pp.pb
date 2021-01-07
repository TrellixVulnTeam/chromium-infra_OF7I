create {
  platform_re: "linux-.*|mac-.*",
  source {
    url {
      download_url: "https://ftp.gnu.org/pub/gnu/libtool/libtool-2.4.6.tar.gz"
      version: "2.4.6"
    }
    unpack_archive: true
    patch_dir: "patches"
  }
  build {
    tool: "tools/help2man"
  }
}

upload { pkg_prefix: "tools" }

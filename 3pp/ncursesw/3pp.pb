create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://ftp.gnu.org/gnu/ncurses/ncurses-6.0.tar.gz"
      version: "6.0"
    }
    unpack_archive: true
    patch_dir: "patches"
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

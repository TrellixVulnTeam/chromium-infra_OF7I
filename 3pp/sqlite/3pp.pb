create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "http://sqlite.org/2017/sqlite-autoconf-3190300.tar.gz"
      version: "3.19.3"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://zlib.net/zlib-1.2.11.tar.gz"
      version: "1.2.11"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

create {
  # Only used on linux
  platform_re: "linux-.*"
  source {
    url {
      download_url: "https://github.com/thkukuk/libnsl/archive/libnsl-1.0.4.tar.gz"
      version: "1.0.4"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

create {
  # Only used on linux
  platform_re: "linux-.*"
  source {
    url {
      download_url: "https://github.com/thkukuk/libnsl/releases/download/v1.3.0/libnsl-1.3.0.tar.xz"
      version: "1.3.0"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

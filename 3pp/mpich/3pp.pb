create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://www.mpich.org/static/downloads/3.4.1/mpich-3.4.1.tar.gz"
      version: "3.4.1"
    }
    unpack_archive: true
    patch_version: "chromium.4"
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

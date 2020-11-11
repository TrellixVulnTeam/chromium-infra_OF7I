create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://sourceforge.net/projects/bzip2/files/bzip2-1.0.6.tar.gz"
      version: "1.0.6"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

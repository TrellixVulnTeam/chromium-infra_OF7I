create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://ftp.pcre.org/pub/pcre/pcre2-10.23.tar.gz"
      version: "10.23"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

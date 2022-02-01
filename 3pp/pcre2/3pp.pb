create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://sourceforge.net/projects/pcre/files/pcre2/10.23/pcre2-10.23.tar.gz/download"
      version: "10.23"
    }
    unpack_archive: true
    cpe_base_address: "cpe:/a:pcre:pcre2"
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

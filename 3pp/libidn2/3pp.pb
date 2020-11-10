create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://ftp.gnu.org/gnu/libidn/libidn2-2.0.4.tar.gz"
      version: "2.0.4"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

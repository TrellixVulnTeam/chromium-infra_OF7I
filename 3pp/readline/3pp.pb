create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://ftp.gnu.org/gnu/readline/readline-7.0.tar.gz"
      version: "7.0"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

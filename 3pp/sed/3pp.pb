create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://ftp.gnu.org/gnu/sed/sed-4.2.2.tar.gz"
      version: "4.2.2"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "tools" }

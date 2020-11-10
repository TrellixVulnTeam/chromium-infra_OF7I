create {
  platform_re: ".*-arm.*|.*-mips.*"
  unsupported: true
}

create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://ftp.gnu.org/gnu/texinfo/texinfo-6.5.tar.gz"
      version: "6.5"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "tools" }

create {
  platform_re: ".*-arm.*|.*-mips.*"
  unsupported: true
}

create {
  platform_re: "linux-.*|mac-.*",
  source {
    url {
      download_url: "https://ftp.gnu.org/gnu/binutils/binutils-2.31.tar.gz"
      version: "2.31"
    }
    unpack_archive: true
  }
  build {
    tool: "texinfo"
  }
}

upload { pkg_prefix: "tools" }

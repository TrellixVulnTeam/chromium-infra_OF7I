create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://www.infradead.org/~tgr/libnl/files/libnl-3.2.25.tar.gz"
      version: "3.2.25"
    }
    unpack_archive:true
  }

  build {
    tool: "tools/flex"
  }
}

upload { pkg_prefix: "tools" }
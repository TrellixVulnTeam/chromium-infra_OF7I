create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://ftp.gnu.org/gnu/ed/ed-1.15.tar.lz"
      version: "1.15"
      extension: ".tar.lz"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "tools" }


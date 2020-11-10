create {
  platform_re: "linux-amd64|mac-.*"
  source {
    url {
      download_url: "https://ftp.gnu.org/gnu/autoconf/autoconf-2.69.tar.gz"
      version: "2.69"
    }
    unpack_archive: true
    patch_dir: "patches"
    patch_version: "chromium1"
  }
  build {}
}

upload { pkg_prefix: "tools" }

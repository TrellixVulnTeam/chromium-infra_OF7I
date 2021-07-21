create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://github.com/westes/flex/releases/download/v2.6.4/flex-2.6.4.tar.gz"
      version: "2.6.4"
    }
    unpack_archive: true
  }

  build {
    tool: "tools/gettext"
    tool: "tools/help2man"
  }
}

upload { pkg_prefix: "tools" }
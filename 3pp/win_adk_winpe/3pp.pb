create {
  platform_re: "windows-.*"
  source {
    url {
      download_url: "https://go.microsoft.com/fwlink/?linkid=2120253"
      version: "10.1.19041.1"
      extension: ".exe"
    }
  }
  build {}
}

upload { pkg_prefix: "tools" }

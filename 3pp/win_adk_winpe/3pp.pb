create {
  platform_re: "windows-.*"
  source {
    url {
      download_url: "https://go.microsoft.com/fwlink/?linkid=2166133"
      version: "10.1.22000.1"
      extension: ".exe"
    }
  }
}

upload { pkg_prefix: "tools" }

create {
  platform_re: "mac-.*"
  source {
    git {
      repo: "https://github.com/apple/swift-format.git"
      tag_pattern: "0.%s00.0"
      tag_filter_re: "0.50500.0"
    }
    cpe_base_address: "cpe:/a:swiftformat_project:swiftformat"
    patch_version: "chromium.1"
  }
  build {}
}

upload { pkg_prefix: "tools" }


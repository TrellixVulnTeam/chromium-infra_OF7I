create {
  platform_re: "mac-.*"
  source {
    git {
      repo: "https://github.com/apple/swift-format.git"
      tag_pattern: "0.%s00.0"
    }
  }
  build {}
}

upload { pkg_prefix: "tools" }


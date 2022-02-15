create {
  platform_re: "linux-amd64"
  source {
    git {
      repo: "https://github.com/facebook/zstd.git"
      tag_pattern: "v%s"
    }
  }
  build {
  }
}

upload { pkg_prefix: "static_libs" }


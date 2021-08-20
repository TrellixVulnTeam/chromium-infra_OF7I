create {
  source {
    git {
      repo: "https://github.com/grpc/grpc-go"
      tag_pattern: "v%s"
      tag_filter_re: "v([0-9]+\.?)+$"
    }
  }

  build { tool: "tools/go" }
}

upload { pkg_prefix: "go/github.com/grpc" }

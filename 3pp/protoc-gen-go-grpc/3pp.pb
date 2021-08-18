create {
  source {
    git {
      repo: "https://github.com/grpc/grpc-go"
      tag_pattern: "v%s"
    }
  }

  build { tool: "tools/go" }
}

upload { pkg_prefix: "go/github.com/grpc" }

create {
  # We are currently building this package only for linux platform.
  platform_re: "linux-.*"
}

create {
  source {
    git {
      repo: "https://chromium.googlesource.com/external/github.com/protocolbuffers/protobuf-go"
      tag_pattern: "v%s"
    }

    subdir: "src/google.golang.org/protobuf"
  }

  build { tool: "tools/go" }
}

upload { pkg_prefix: "go/github.com/protocolbuffers" }

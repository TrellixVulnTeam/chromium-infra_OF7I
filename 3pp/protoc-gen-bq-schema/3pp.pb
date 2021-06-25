create {
  # We are currently building this package only for linux platform.
  platform_re: "linux-.*"
}

create {
  source {
    git {
      repo: "https://github.com/GoogleCloudPlatform/protoc-gen-bq-schema"
      tag_pattern: "v%s"
    }
  }

  build { tool: "tools/go" }
}

upload { pkg_prefix: "go/github.com/GoogleCloudPlatform/protoc-gen-bq-schema" }

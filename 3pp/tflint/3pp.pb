create {
  # We are currently building this package only for linux platform.
  platform_re: "linux-.*"
}

create {
  source {
    script {
      name: "source.sh"
    }
  }

  build { tool: "tools/go" }
}

upload { pkg_prefix: "go/github.com/terraform-linters" }

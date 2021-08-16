create {
  source {
    git {
      repo: "https://chromium.googlesource.com/external/github.com/bazelbuild/buildtools"
    }
  }

  build { tool: "tools/go" }
}

upload { pkg_prefix: "tools" }

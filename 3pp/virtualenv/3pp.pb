create {
  source {
    git {
      repo: "https://github.com/pypa/virtualenv.git"
      version_restriction {
        op: EQ
        val: "16.7.10"
      }
    }
  }
  build {}
}

upload {
  pkg_prefix: "tools"
  universal: true
}

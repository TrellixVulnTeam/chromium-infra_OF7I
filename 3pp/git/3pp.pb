create {
  verify { test: "git_test.py" }
  source { patch_version: "chromium.6" }
}

create {
  platform_re: "linux-.*|mac-.*"

  source { git {
    repo: "https://chromium.googlesource.com/external/github.com/git/git"
    tag_pattern: "v%s"
  }}

  build {
    tool: "autoconf"
    tool: "sed"
    tool: "gettext"

    dep: "zlib"
    dep: "curl"
    dep: "pcre2"
    dep: "libexpat"
  }
}

create {
  platform_re: "windows-.*"
  source { script { name: "fetch_win.py" }}
  build { install: "install_win.sh" }
  package {
    # On windows we actually source the version of git from the git-on-windows
    # project, which maintains its own patch suffix of the form ".windows.XX".
    #
    # Unfortunately, we only support deploying a SINGLE tag across all
    # platforms, which means that we need the tagged package to match
    # everywhere.
    #
    # So, we remove the .windows.XX suffix here; if git-for-windows produces
    # a new patch version that you need, bump the 'patch_version' at the top of
    # this file. You'll get new builds on other platforms, too, but ¯\_(ツ)_/¯.
    alter_version_re: "(.*)\.windows\.\d*(.*)"
    alter_version_replace: "\\1\\2"
  }
}

upload { pkg_prefix: "tools" }

 create {
   platform_re: "linux-amd64"
   source { git {
     repo: "https://github.com/libexpat/libexpat"
     # libexpat release tags look like "R_1_2_3"
     tag_pattern: "R_%s"
     version_join: "_"
    }
   }
   build {
    tool: "autoconf"
   }
 }

 upload { pkg_prefix: "static_libs" }

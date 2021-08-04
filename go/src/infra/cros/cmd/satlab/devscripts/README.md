# Development scripts

This directory contains development scripts that execute before
or after Makefile targets, but only if the script exists.

These scripts are intended to be specific to the developer who
writes them, please do not commit any scripts in this directory.

Here is a sample `pre-satlab` script.

```
#!/bin/bash -x

go fmt ./... || exit 1
go test ./... 1>/dev/null || exit 1
```

Here is a sample post-satlab script that copies the executable
to the user's home directory.

```
#!/bin/bash -x

# install file to home directory
cp ./satlab ~/satlab
```

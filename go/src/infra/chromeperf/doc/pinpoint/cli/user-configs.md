# Pinpoint CLI user configurations

The `pinpoint` command supports reading a YAML configuration file that's
referred to by the following means:

- `PINPOINT_USER_CONFIG` environment variable.
- `$XDG_CONFIG_HOME/pinpoint/config.yaml` as the default location (this is
  platform dependent)

## Configuration schema

NOTE: For up-to-date details, see the file
[config.go](https://source.chromium.org/chromium/infra/infra/+/master:go/src/infra/chromeperf/pinpoint/cli/config.go)
and in-source documentation.

The following YAML keys are supported:

| Key | Type | Description |
|-----|------|-------------|
| `endpoint` | `string` | The fully qualified domain name of the gRPC service to connect to. |
| `wait` | `bool` | Whether to always wait for scheduled or retrieved jobs to finish. |
| `download_results` | `bool` | Whether to download the results of jobs when getting or starting them. |
| `open_results` | `bool` | Whether to open the downloaded results with the default web browser. |
| `results_dir` | `string` | The default directory where results should be downloaded (overrides the default `/tmp` or equivalent directory in non-Unix platforms). |
| `quiet` | `bool` | Whether to suppress progress output. |
| `presets_file` | `string` | The default file hosting presets. Learn more about presets at [job-presets.md](job-presets.md) |

## Querying configuration

These user configurations replace the defaults that would apply had the
related flags been provided.  To see what the current configuration options
for a user are, you can invoke the `config` subcommand to see the flags that
would have been default-applied when invoking subcommands.

```bash
$ pinpoint config
```

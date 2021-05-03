# Job presets

The `pinpoint` utility supports loading presets which define named configurations that specify essential inputs for jobs. This makes it more convenient to refer to more memorable names instead of remembering a set of flags.

The presets file is a YAML structured file that's found through the following means (in order):

1. The `--presets-file` flag provided when invoking `pinpoint`.
1. If defined in [user configurations](user-configs.md), the value of
   `presets_file`.
1. `.pinpoint-presets.yaml` in the current working directory, where `pinpoint` was invoked.

## Preset schema

NOTE: For the most up-to-date schema, see
[presets.go](https://source.chromium.org/chromium/infra/infra/+/master:go/src/infra/chromeperf/pinpoint/cli/presets.go)
and the field mappings between YAML structures and the Go structs used to
represent the data.

The YAML file must have a top-level map named `presets`, where keys map to the following supported maps:

### telemetry-experiment

| Key | Type | Description |
|-----|------|-------------|
| `config` | `string` | The configuration (bot) to run the job in. |
| `story_selection`| `story-selection` | See [story-selection](#story-selection) for more. |
| `benchmark` | `string` | The name of the Telemetry benchmark to run. |
| `measurement` | `string` | The measurement to select for in the A/B experiment. |
| `grouping_label` | `string` | The grouping label to select stories with. |
| `extra_args` | `[string]` | A list of flags provided to the Telemetry benchmark runner. |

### story-selection

| Key | Type | Description |
|-----|------|-------------|
| `story` | `string` | A specific story to run within the Telemetry benchmark. |
| `story_tags` | `[string]` | A list of story tags to run. |

## Using presets

To use a named preset in commands that start jobs, use the `--preset` flag to
select the named preset.  If you provide flags that override settings in the
named preset, the flags provided in the command invocation take precedence.

## Example

### Simple cases

Given a preset file:

```yaml
# .pinpoint-presets.yaml
presets:
  sample:
    telemetry_experiment:
      config: linux-perf
      story_selection:
        story_tags:
          - all
      benchmark: some-benchmark
      extra_args:
          - --extra_tracing_categories
          - some-category
          - --additional-flag
```

When we invoke the `pinpoint` tool the following way in a git repository that
is managed with `depot_tools` where the current branch already has a CL:

```bash
$ pinpoint experiment-telemetry-start --preset=sample
```

This will start a Telemetry A/B experiment for `some-benchmark` with on the
`linux-perf` configuration using the `all` story tag, with the provided extra
arguments defined in `extra_args`.

TIP: This is most useful when the presets are checked into the repository,
allowing for sharing the configuration across multiple users.

### Using YAML features

Because we're using YAML, we can leverage features of the spec that allow us
to reduce repetition in the presets.  It might be that there are common parts
of experiments that you and your team would like to run consistently, a
config with minimal repetition might look like:

```yaml
# .pinpoint-presets.yaml
presets:
  basic: &basic
    telemetry_experiment:
      config: linux-perf
      story_selection:
        story_tags: ["all"]
      benchmark: basic
      extra_args:
        - --pageset-repeats=10
  basic-mobile:
    <<: *basic
    telemetry_experiment:
      config: android-pixel2-perf
```

This example uses a feature called YAML anchors to reduce the repetition --
this way when the `basic` preset is changed, all the ones that merge in the
definition of `basic` do not need to repeat the changes.

With the above configuration, you can then run the `pinpoint` tool selecting either the `basic` or `basic-mobile` preset.

```bash
$ pinpoint experiment-telemetry-start --preset=basic
...
$ pinpoint experiment-telemetry-start --preset=basic-mobile
```

### Using different preset files

It's possible that a single preset file might get confusing and might grow beyond a manageable size. Thankfully we can specify which preset file to use when loading presets:

```bash
$ pinpoint experiment-telemetry-start \
    --preset-file=complex-presets.yaml \
    --preset=default
...
$ pinpoint experiment-telemetry-start \
    --preset-file=simple-presets.yaml \
    --preset=default
```
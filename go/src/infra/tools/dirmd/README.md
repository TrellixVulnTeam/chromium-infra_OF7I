# Directory metadata

DIR_METADATA files are a source-focused mechanism by which owners can
provide users of their code important information. They map a directory to
external entities, including:

* The Monorail component where bugs should be filed.
* The team responsible for the code.
* The OS type.

This infrastructure is overall generic. It has Chromium-specific proto fields,
but their usage is not required, and overall this infra can be used by other
projects as well.

## DIR_METADATA files

DIR_METADATA files are structured text-protobuf files that are amenable to
programmatic interaction. Example:

```
team_email: "team@chromium.org"
os: LINUX
monorail {
  project: "chromium"
  component: "Infra"
}
```

For file schema, see [`Metadata` message](./proto/dir_metadata.proto).

### Inheritance

DIR_METADATA files apply to all contents of a directory including its
subdirectories. By default, individual fields are inherited by subdirectories.
In the following, example, the value of `monorail.project` field in directory
`a/b` is "chromium".

**a/DIR_METADATA**
```
monorail {
  project: "chromium"
  component: "Component"
}
```

**a/b/DIR_METADATA**
```
monorail { component: "Component>Foo" }
```

## dirmd tool

`dirmd` is a tool to work with DIR_METADATA files. Features:

* Gather all metadata in a given directory tree and export it to a single
  JSON file. Optionally remove all redundant metadata, or instead compute
  inherited metadata.
* Compute inherited metadata for a given set of directories.
* Convert from text proto to JSON which is easier to interpret by programs
  that don't have an easy access to the protobuf files.
* Validate a given set of files. Used in PRESUBMIT.
* Fall back to legacy `OWNERS` files, so that metadata can migrate off of
  OWNERS files smoothly.

The tool is deployed via depot_tools and is available as a
[CIPD package](https://chrome-infra-packages.appspot.com/p/infra/tools/dirmd).

The tool also hosts implementation of the
[metadata-exporter](https://ci.chromium.org/p/chromium/builders/ci/metadata-exporter)
builder.

Source code: [./cmd/dirmd](./cmd/dirmd).

## Library

Go package [infra/tools/dirmd](https://source.chromium.org/chromium/infra/infra/+/master:go/src/infra/tools/dirmd/?q=dirmd)
can be used work with METADATA files programmatically.
The `dirmd` executable is a thin wrapper around it.

## Continuous export

[metadata-exporter](https://ci.chromium.org/p/chromium/builders/ci/metadata-exporter)
builder contunuously exports metadata from the src.git to
[gs://chrome-metadata/metadata_reduced.json](https://storage.googleapis.com/chrome-metadata/metadata_reduced.json)
and
[gs://chrome-metadata/metadata_computed.json](https://storage.googleapis.com/chrome-metadata/metadata_computed.json).
`metadata_reduced.json` is semantic equivalent of `metadata_computed.json`, but
it has all redundant information removed (see
[MappingForm.REDUCED](https://source.chromium.org/chromium/infra/infra/+/master:go/src/infra/tools/dirmd/proto/mapping.proto;l=28?q=mappingform&sq=)).
As of July 2020, the update latency is up to 20 min.


## Links

* [Original design doc](https://docs.google.com/document/d/17WMlceIMwge2ZiCvBWaBuk0w60YgieBd-ly3I8XsbzU/preview).

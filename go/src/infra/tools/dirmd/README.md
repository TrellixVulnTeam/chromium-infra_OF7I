# DIR_METADATA files

DIR_METADATA files are a source-focused mechanism by which owners can
provide users of their code important information, including:

* The team responsible for the code.
* The Monorail component where bugs should be filed.
* The OS type.

DIR_METADATA files are structured protobuf files that are amenable to
programmatic interaction.

## Usage

DIR_METADATA files apply to all contents of a directory including its
subdirectories.

By default, individual fields are inherited by subdirectories. In the following,
example, the value of `monorail.project` field in directory `a/b` is "chromium".

**a/DIR_METADATA**
```
monorail {
  project: "chromium"
  component: "Component"
}
team_email: "team@chromium.org"
os: OS_LINUX
```

**a/b/DIR_METADATA**
```
monorail {
  component: "Component>Foo"
}
team_email: "foo-team@chromium.org"
```

## File schema

For file schema, see `Metadata` message in
[dir_metadata.proto](./proto/dir_metadata.proto).

## Links

* [Original design doc](https://docs.google.com/document/d/17WMlceIMwge2ZiCvBWaBuk0w60YgieBd-ly3I8XsbzU/preview).

# Shivas developer guide

*This document is intended to be a reference for any developers looking to
modify or add functionality to shivas. It provides necessary details for most of
the use cases that will be encountered when modifying shivas. If you are looking
for [user-manual](go/shivas-manual-os), the built-in `-help` would give the most
updated and correct information. Please contact us at
browser-fleet-automation@google.com if you find something missing in this
document.*

[TOC]

[go/shivas-dev](go/shivas-dev)

Shivas is a command line tool that is used to communicate with UFS. In addition
shivas also supports communicating with various other services including
inventory V2, and swarming. The tool is
[released](https://g3doc.corp.google.com/company/teams/chrome/ops/fleet/software/unified-fleet-system/tooling/shivas-installation.md?cl=head#installation)
on `cipd` and therefore requires `depot_tools` to be installed. Anyone can
compile the tool and run it, however the services that tool connects to will
require user authorization.

## Compiling Shivas
Shivas is shipped with a makefile that handles most of the compilation. Compile
shivas using `make` and delete the compiled files using `make clean`. It is also
possible to compile shivas using `go build`.

**Note:**\
It is possible to compile shivas for testing locally. This will make shivas hit
`localhost` for UFS endpoints. You need to run UFS locally to be able use shivas
this way. Compile shivas for local use by running
```
go build -tags='dev'
```

## Source code survey
* **cmdhelp**: Contains most of the help description strings. It's a good place
  to dump all the strings that will be used in help menu.
* **internal**: Contains the logic to run all the commands in shivas
* **site**: Contains server urls and some common flags for use in commands.
* **utils**: Contains a few helper functions that are shared between the
  commands.

## UFS API (quick look)

UFS APIs are generally given as a set of four operations per entity. They are
* Create:\
  Takes the complete proto message describing the entity for update
  and returns the updated proto message\
  Example:\
    `CreateAsset`RPC takes the asset proto to be created
    ```
      message CreateAssetRequest {
        // The asset to register
        models.asset asset = 1 [(google.api.field_behavior) = REQUIRED];
      }
    ```

* Read: (& List)\
  Read takes the primary key of the entity to be retrieved and returns the
  requested entity. This will always return a single entity.\
  Example:\
    `GetAsset` RPC takes the asset name to be retrieved.
    ```
      message GetAssetRequest {
        // The name of the asset to retrieve.
        string name = 1 [
          (google.api.field_behavior) = REQUIRED,
          (google.api.resource_reference) = { type: "unified-fleet-system.appspot.com/asset" }
        ];
      }
    ```
  List is generally used when the result of the query is more than one. List
  allows for filters, which are usually used in order to obtain the result.
  The results of the query might also be broken up into multiple messages.
  These are handled using a cursor to get rest of the data as required\
  Example:\
    `ListAsset` RPC takes page_size, page_token(cursor), filter and keys_only
    flag and returns a list of assets along with an optional token for the
    next_page
    ```
      message ListAssetsRequest {
        // The maximum number of assets to return. The service may return fewer than
        // this value.
        // If unspecified, at most 100 assets will be returned.
        // The maximum value is 1000; values above 1000 will be coerced to 1000.
        int32 page_size = 1;

        // A page token, received from a previous `ListAssets` call.
        // Provide this to retrieve the subsequent page.
        //
        // When paginating, all other parameters provided to `ListAssets` must match
        // the call that provided the page token.
        string page_token = 2;

        // filter takes the filtering condition
        string filter = 3;

        // if this is true, only keys will be returned else the entire object
        // will be returned. By setting this to true, the list call be will faster.
        bool keysOnly = 4;
      }

      message ListAssetsResponse {
        // The assets from datastore.
        repeated models.asset assets = 1;

        // A token, which can be sent as `page_token` to retrieve the next page.
        // If this field is omitted, there are no subsequent pages.
        string next_page_token = 2;
      }
    ```
* Update:\
    This type of operation generally takes two inputs, entity and a list of
    strings called update_mask.\
    Example:
    ```
      message UpdateAssetRequest {
        // The asset to update.
        models.asset asset = 1 [(google.api.field_behavior) = REQUIRED];

        // The list of fields to be updated.
        google.protobuf.FieldMask update_mask = 2;
      }

    ```
    There are two types of update operations that are available.
    * Full Update:\
      Whatever data exists on the given entity is overwritten. This would only
      be performed if `update_mask == nil`\
      Example:\
        Consider existing asset `my-laptop`
        ```
          {
            "name": "my-laptop",
            "type": "DUT",
            "model": "bluebird",
            "location": {
                    "aisle": "",
                    "row": "14",
                    "rack": "chromeos6-row14-rack23",
                    "rackNumber": "23",
                    "shelf": "",
                    "position": "16",
                    "barcodeName": "",
                    "zone": "ZONE_CHROMEOS6"
            },
            "info": {
                    "assetTag": "my-laptop",
                    "serialNumber": "laptop-serial-xoxo",
                    "model": "bluebird",
                    "buildTarget": "octopus",
                    "referenceBoard": "octopus",
                    "ethernetMacAddress": "ff:ee:dd:cc:bb:aa",
                    "sku": "2",
            },
          }
        ```
        Updating it with
        ```
          {
            "name": "my-laptop",
            "type": "LABSTATION",
            "model": "bluebird",
            "location": {
                    "aisle": "",
                    "row": "1",
                    "rack": "chromeos6-row1-rack21",
                    "rackNumber": "21",
                    "shelf": "",
                    "position": "15",
                    "barcodeName": "",
                    "zone": "ZONE_CHROMEOS6"
            },
          }
        ```
        Will replace entire row with the new configuration. This results in the
        info field being reset.
    * Partial Update:\
      A set of field masks are provided in `update_mask`.\
      Example:\
        Consider the following update run on the same entity as above
        ```
          {
            "asset" : {
              "name": "my-laptop",
              "type": "DUT",
              "model": "eve",
              "location": {
                      "aisle": "10",
                      "position": "15",
                      "zone": "ZONE_CHROMEOS6"
              },
            },
            "update_mask": ["type", "model", "location.aisle"]
          }
        ```
        This will only update the `type`, `model` and `location.aisle`. The
        `position` and `zone` fields are ignored. This will update the given
        entry as
        ```
          {
            "name": "my-laptop",
            "type": "DUT",
            "model": "eve",
            "location": {
                    "aisle": "10",
                    "row": "1",
                    "rack": "chromeos6-row1-rack21",
                    "rackNumber": "21",
                    "shelf": "",
                    "position": "15",
                    "barcodeName": "",
                    "zone": "ZONE_CHROMEOS6"
            },
          }
        ```
* Delete:\
    Only take the primary key for the entity and deletes the record\
    Example:
    ```
      message DeleteAssetRequest {
        // The name of the asset to delete
        string name = 1 [
          (google.api.field_behavior) = REQUIRED,
          (google.api.resource_reference) = { type: "unified-fleet-system.appspot.com/Asset" }
        ];
      }
    ```

Entities tracked by UFS are generally required to support these 4 operations.
This is commonly referred to as CRUD operations

**Note:**
* There is another operation that is applicable to a significant number of
  entities. Rename basically creates a new entry for the given entity and deletes
  the old one. Rename is also implemented using a generic template. Check
  [`generic_rename_cmd.go`](https://chromium.googlesource.com/infra/infra/+/main/go/src/infra/cmd/shivas/utils/rename/generic_rename_cmd.go)
* UFS provides other RPCs that do not do any CRUD operations. These are
  generally not used through shivas.

## Shivas input types
* Command Line flags:\
    Most user friendly mode of input provided by shivas. If the command is
    modifying/updating an entity, then this probably maps to a partial update
    RPC.
* JSON file:\
    A JSON file representing a proto required as an input to RPC is given. Error
    is thrown if JSON is malformed or doesn't match the proto format. Check help
    for the JSON format.
    Note: `update` commands usually do a full update on JSON. This is because
    the expected input for the the command is not the Request proto (which
    contains the update_mask). But the entity proto itself. Check the help for
    the particular command to see what is the JSON file expectations
* CSV file:\
    A CSV file representing a list of entities is given as input to the command.
    The format of the CSV file varies from command to command. CSV files
    generally do not support updating all the possible options in the proto.
    Check help for the subset of options that they support.

### Update Command line flags
Command line flags can be added to any given command. The flags are created
using [flags](https://pkg.go.dev/flag) package. Strings are by far the most
common ones, but bools, strings slices, ints ... etc,. are also supported. It is
also possible to create a custom flags type. Make sure to validate the inputs
for the new flag.

#### Default flags
All shivas commands include three default flagsets. These are auth flags that
handle the authorization options, `env` flags that handle processing the
environment vars, `common` flags which include `verbose` option and `output`
flags handle the output format for shivas. Include these in your command as
required.

### Update CSV format
This is generally not recommended as it is likely to break something downstream.
If you have to update the CSV fields, they are usually recorded as `mcsvFields`.
CSV updates are treated as if command line updates were performed quickly in
succession.

**Note:**
* Send an email to ufs-announce@google.com with the request. This will allow
  relevant parties to respond with their concerns.
* CC chrome-fleet-software@google.com on the bug to get approval for this.

## Testing with UFS locally
It is possible to run UFS locally and test your changes to the code. UFS is
compiled as a binary which in theory can be run on any computer. This can be
done by running `make dev` in the UFS source code. This will allow you to test
your changes locally with a few catches.

* You need a service account to run UFS, this would usually mean that you need
  to run luci-auth to get a certificate. If you don't have the permissions the
  error log includes the fix for it.
* There is no mock database service for this. Which means you can potentially
  update prod database if you are not careful. Use the `-dev` flag with shivas
  to avoid updating the prod
* You'll need to build shivas with `go build -tags='dev'` to hit the local
  endpoints
* UFS enforces realm permissions. This means unless you are in the right groups
  you will not be able to modify the data. Which means you will get permission
  error even if you are running both service and shivas locally. Contact your
  manager to get proper permissions if this happens.

If this is your first time you will also need to provide creds to Shivas.

```
shivas login
```

## Namespaces in UFS
UFS differentiates between the `browser` or `os` namespace to perform updates to
the relevant databases. This is done in shivas by setting `SHIVAS_NAMESPACE` to
either `browser` or `os`. You can also include `env` flags in your command that
will allow for `-namespace` flag that can be set wile running the command.\
If you wish your command to run only one one of these namespaces, you need to
explicitly set the namespace in your context as follows
```
ctx = utils.SetupContext(ctx, ufsUtil.OSNamespace)
```
where ufsUtil is the utils package in UFS.

## Barcode reader support
Shivas supports using barcode reader for updates. This is done to speed up the
work for certain workflows. Barcode reader can only be used in certain
circumstances, this is due to the obvious limitations with amount of data that
can be read through the barcode. By default barcode readers work like a keyboard
input device. Whatever is scanned to barcode is dumped to `stdin` of the process
terminated by a new line. This allows us to use barcode readers in certain
processes to speed up the work. Shivas handles barcode reader based updates by
calling the required RPC as soon as all the data is available to perform the
operation.

Current use cases include registering a bunch of assets into a rack in OS lab.
Deleting a list of assets. Updating location data for a bunch of assets and
other such processes.

Check `-scan` functionality in add/update assets for some implementation
details.

## Shivas support for X language/Shivas wrappers
Shivas was intended to be used as an user tool. That is no longer true and
various wrappers in multiple languages has been written for shivas. It is
generally recommended to call the UFS API directly if possible. But as UFS RPCs
are implemented using gRPC and it might not be supported in your situation, some
wrappers were inevitable for shivas. Before implementing a new wrapper for
shivas, please check if any of the following is available.
* gRPC support for your OS/Language
* use code search to look for existing wrappers

If no alternative is available, please announce your intention to shivas
maintainers at ufs-announce@google.com. Check `-version` option in shivas and
use it to avoid unexpected crashes when someone changes the output format. And
finally add your implementation to the list below

* [**chrome-golo**/*WindowsPowerShell*](https://chrome-internal.googlesource.com/chrome-golo/chrome-golo/+/main/scripts/WindowsPowerShell/Modules/Shivas):
  intended for use in Windows VMs and bots

## Testing shivas
Shivas code is generally not tested. This was because it was never intended to
be used in it's current context and was initially conceived as a light weight
tool for updating UFS. That no longer being the case, it is recommended that you
add unit tests to any new command that gets added to shivas. Owners approval
will be hard to get without any tests written for the tool. We will be really
happy if you can add unit tests to existing code.


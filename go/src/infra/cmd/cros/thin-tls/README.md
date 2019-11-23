# thin-tls

thin-tls is a thin/fake implementation of the TLS API.

This is part of go/cros-f20.  As part of F20, we will provide a test
lab services API for test drivers to use, rather than being tightly
coupled with the lab services implementation.  thin-tls provides a
dumb implementation of the API that can be used for testing,
experimentation, and implementing test driver support.

## Building

See the [Go README](../../../../../README.md) to set up the Go environment.

Inside the Go environment, run `make` to build the thin-tls server and client.

## Config

The config file is in JSON format.

Keys:

- `dutHostname`: The hostname of the DUT for SSH.

Your user SSH configuration should be able to SSH to the DUT with just `ssh HOSTNAME`.

## Usage

After creating a config, start running the `thin-tls` service.  Check
the command help for optional flags.  If your config file is not
`thin-tls-config.json`, you need to specify the config path
explicitly.

You can then interact with the TLS API service, by default on port 50051.

The `client` program provides an experimental CLI wrapper around some
of the RPCs.  This should only be used for experimenting interactively
and **MUST NOT** be used to integrate with the API service.  Check the
command help for usage.

## Implementation notes

### SSH

The `DutShell` API is implemented by creating an `ssh` subprocess.  It
relies on your user SSH configuration for any necessary credentials or
options.

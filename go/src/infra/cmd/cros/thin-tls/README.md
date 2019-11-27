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

- `dutHostname`: The hostname of the DUT for SSH.  Your user SSH
  configuration should be able to SSH to the DUT with just `ssh
  HOSTNAME`.
- `rpmMachine`, `powerOutlet`, `powerunitHostname`, `hydraHostname`:
  These are all used for RPM.  See `rpm_client` for the meaning of
  these.

## Usage

After creating a config, start running the `thin-tls` service.  Check
the command help for optional flags.  If your config file is not
`thin-tls-config.json`, you need to specify the config path
explicitly.

You can then interact with the TLS API service, by default on port 50051.

Note: Because this is intended only for testing, parts of the implementation
depend on existing setup in your environment.  Read the Implementation
notes section below about what you need set up outside of thin-tls.

The `client` program provides an experimental CLI wrapper around some
of the RPCs.  This should only be used for experimenting interactively
and **MUST NOT** be used to integrate with the API service.  Check the
command help for usage.

## Implementation notes

### SSH

The `DutShell` API is implemented by creating an `ssh` subprocess.  It
relies on your user SSH configuration for any necessary credentials or
options.

### RPM

The RPM service is implemented by creating an `rpm_client` subprocess.
THis should be installed as part of go/lab-tools.

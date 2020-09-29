package cloudtasks

// Package cloudtasks provides a client interface for GCP's Cloud Tasks API,
// with a standard set of options set by command line flags.
import (
	"context"
	"flag"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"

	"infra/libs/grpcclient"
)

// Options describes the client configuration for cloudtasks.
type Options struct {
	*grpcclient.Options
}

// NewOptionsFromFlags returns an Options instance populated from command line flag options.
func NewOptionsFromFlags() *Options {
	ret := &Options{&grpcclient.Options{}}
	ret.registerFlags(flag.CommandLine)
	return ret
}

// NewClient returns a cloudtasks Client according to Options settings.
func (c *Options) NewClient(ctx context.Context) (*cloudtasks.Client, error) {
	opts, err := c.DefaultClientOptions(ctx)
	if err != nil {
		return nil, err
	}

	client, err := cloudtasks.NewClient(ctx, opts...)
	return client, err
}

func (c *Options) registerFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Address, "cloudtasks-address", "cloudtasks.googleapis.com:443", "Address for cloudttasks service")
	fs.IntVar(&c.DefaultTimeoutMs, "cloudtasks-timeout-ms", 25*1000, "Default RPC timeout for cloudttasks service")
}

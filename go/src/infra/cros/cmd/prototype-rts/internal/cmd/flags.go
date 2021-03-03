package cmd

import "github.com/maruel/subcommands"

type flags struct {
	subcommands.CommandRunBase

	progressSinkPort int
	tlsCommonPort    int
}

func (f *flags) InitRTSFlags() {
	if f == nil {
		*f = flags{}
	}
	f.Flags.IntVar(&f.progressSinkPort, "progress-sink-port", 0, "Port for the local ProgressSink gRPC server. The default value implies a random port selection.")
	f.Flags.IntVar(&f.tlsCommonPort, "tls-common-port", 0, "Port for the local TLSCommon gRPC server. The default value implies a random port selection.")
}

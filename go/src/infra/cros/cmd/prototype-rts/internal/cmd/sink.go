package cmd

import (
	"context"
	"fmt"
	"net"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/config/go/api/test/rtd/v1"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// StartProgressSink starts the ProgressSinkService by itself.
func StartProgressSink() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "start-progress-sink",
		ShortDesc: "Starts the ProgressSinkService without any other RTS services.",
		LongDesc:  "Starts the ProgressSinkService without any other RTS services.",
		CommandRun: func() subcommands.CommandRun {
			c := &sinkCommand{}
			c.Flags.IntVar(&c.port, "port", 0, "Port on which to bind the GRPC server")
			return c
		},
	}
}

type sinkCommand struct {
	subcommands.CommandRunBase

	port int
}

func (sc *sinkCommand) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, sc, env)
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", sc.port))
	if err != nil {
		logging.Errorf(ctx, "prototype-rts: %s", err)
		return 1
	}
	logging.Infof(ctx, "gRPC listening on %v", l.Addr())
	s := server{}
	if err := s.Serve(l); err != nil {
		logging.Errorf(ctx, "prototype-rts: %s", err)
		return 1
	}
	return 0
}

type server struct {
	rtd.UnimplementedProgressSinkServer
}

func (s server) Serve(l net.Listener) error {
	server := grpc.NewServer()
	// Register reflection service to support grpc_cli usage.
	reflection.Register(server)
	rtd.RegisterProgressSinkServer(server, &s)
	return server.Serve(l)
}

func (s server) ReportResult(ctx context.Context, req *rtd.ReportResultRequest) (*rtd.ReportResultResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportResult not implemented")
}
func (s server) ReportLog(srv rtd.ProgressSink_ReportLogServer) error {
	return status.Errorf(codes.Unimplemented, "method ReportLog not implemented")
}
func (s server) ArchiveArtifact(ctx context.Context, req *rtd.ArchiveArtifactRequest) (*rtd.ArchiveArtifactResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ArchiveArtifact not implemented")
}

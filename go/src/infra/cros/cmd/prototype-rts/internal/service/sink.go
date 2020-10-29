package service

import (
	"fmt"
	"net"

	"go.chromium.org/chromiumos/config/go/api/test/rtd/v1"
	"go.chromium.org/luci/common/logging"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type fakeProgressSinkService struct {
	rtd.UnimplementedProgressSinkServer
}

// LaunchProgressSink starts a fake progress sink service.
func LaunchProgressSink(ctx context.Context, psPort int32) (*grpc.Server, int32, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", psPort))
	if err != nil {
		return nil, 0, err
	}
	logging.Infof(ctx, "ProgressSinkService gRPC listening on %v", l.Addr())
	return fakeProgressSinkService{}.Serve(ctx, l), extractPortOrDie(l.Addr()), nil
}

func (s fakeProgressSinkService) Serve(ctx context.Context, l net.Listener) *grpc.Server {
	server := grpc.NewServer()
	// Register reflection service to support grpc_cli usage.
	reflection.Register(server)
	rtd.RegisterProgressSinkServer(server, &s)
	// Start the server in a background thread, since the Serve() call blocks.
	go func() {
		if err := server.Serve(l); err != nil {
			logging.Errorf(ctx, "ProgressSinkService failed: %v", err)
		}
	}()
	return server
}

func (*fakeProgressSinkService) ReportResult(ctx context.Context, req *rtd.ReportResultRequest) (*rtd.ReportResultResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportResult not implemented")
}
func (*fakeProgressSinkService) ReportLog(srv rtd.ProgressSink_ReportLogServer) error {
	return status.Errorf(codes.Unimplemented, "method ReportLog not implemented")
}
func (*fakeProgressSinkService) ArchiveArtifact(ctx context.Context, req *rtd.ArchiveArtifactRequest) (*rtd.ArchiveArtifactResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ArchiveArtifact not implemented")
}

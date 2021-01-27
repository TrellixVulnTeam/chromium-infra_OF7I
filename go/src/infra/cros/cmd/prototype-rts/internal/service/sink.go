package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"go.chromium.org/chromiumos/config/go/api/test/rtd/v1"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	// Must use log rather than logging, since logging doesn't end up printing
	// to stderr from this location.
	log.Printf("got ReportResultRequest: %v", req)
	return &rtd.ReportResultResponse{}, nil
}
func (*fakeProgressSinkService) ReportLog(srv rtd.ProgressSink_ReportLogServer) error {
	// Must use log rather than logging, since logging doesn't end up printing
	// to stderr from this location.
	log.Printf("starting ReportLog receiver")
	for {
		req, err := srv.Recv()
		if err == io.EOF {
			log.Printf("received ReportLog EOF")
			break
		}
		if err != nil {
			return err
		}
		log.Printf("received ReportLog: %v", req)
	}
	log.Printf("done with ReportLog receiver")
	return nil
}
func (*fakeProgressSinkService) ArchiveArtifact(ctx context.Context, req *rtd.ArchiveArtifactRequest) (*rtd.ArchiveArtifactResponse, error) {
	// Must use log rather than logging, since logging doesn't end up printing
	// to stderr from this location.
	log.Printf("got ArchiveArtifactRequest: %v", req)
	return &rtd.ArchiveArtifactResponse{}, nil
}

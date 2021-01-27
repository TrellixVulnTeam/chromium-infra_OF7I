package service

import (
	"context"
	"fmt"
	"log"
	"net"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type fakeTLSCommonService struct {
	tls.UnimplementedCommonServer
}

// Type-check against the external interface.
var _ tls.CommonServer = &fakeTLSCommonService{}

// LaunchTLSCommon starts a fake TLS common service.
func LaunchTLSCommon(ctx context.Context, tlsPort int32) (*grpc.Server, int32, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", tlsPort))
	if err != nil {
		return nil, 0, err
	}
	logging.Infof(ctx, "fakeTLSCommonService gRPC listening on %v", l.Addr())
	return fakeTLSCommonService{}.Serve(ctx, l), extractPortOrDie(l.Addr()), nil
}

func (s fakeTLSCommonService) Serve(ctx context.Context, l net.Listener) *grpc.Server {
	server := grpc.NewServer()
	// Register reflection service to support grpc_cli usage.
	reflection.Register(server)
	tls.RegisterCommonServer(server, &s)
	// Start the server in a background thread, since the Serve() call blocks.
	go func() {
		if err := server.Serve(l); err != nil {
			logging.Errorf(ctx, "ProgressSinkService failed: %v", err)
		}
	}()
	return server
}

func extractPortOrDie(addr net.Addr) int32 {
	switch addrType := addr.(type) {
	case *net.UDPAddr:
		return int32(addrType.Port)
	case *net.TCPAddr:
		return int32(addrType.Port)
	default:
		log.Panicf("unexpected net.Addr type: %v", addrType)
		panic("can't happen")
	}
}

func (*fakeTLSCommonService) ExecDutCommand(req *tls.ExecDutCommandRequest, srv tls.Common_ExecDutCommandServer) error {
	return status.Errorf(codes.Unimplemented, "method ExecDutCommand not implemented")
}
func (*fakeTLSCommonService) ProvisionDut(ctx context.Context, req *tls.ProvisionDutRequest) (*longrunning.Operation, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Provision not implemented")
}

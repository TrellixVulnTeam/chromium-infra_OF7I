// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"time"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"golang.org/x/crypto/ssh"

	"infra/libs/lro"
	"infra/libs/sshpool"
)

type tlwServer struct {
	tls.UnimplementedWiringServer
	grpcServ *grpc.Server
	lroMgr   *lro.Manager
	tMgr     *tunnelManager
	tPool    *sshpool.Pool
}

func newTLWServer() tlwServer {
	s := tlwServer{
		grpcServ: grpc.NewServer(),
		lroMgr:   lro.New(),
		tPool:    sshpool.New(getSSHClientConfig()),
		tMgr:     newTunnelManager(),
	}
	tls.RegisterWiringServer(s.grpcServ, &s)
	longrunning.RegisterOperationsServer(s.grpcServ, s.lroMgr)
	return s
}

func (s tlwServer) Serve(l net.Listener) error {
	return s.grpcServ.Serve(l)
}

// Close closes all open server resources.
func (s tlwServer) Close() {
	s.tMgr.Close()
	s.tPool.Close()
	s.lroMgr.Close()
	s.grpcServ.GracefulStop()
}

func (s tlwServer) OpenDutPort(ctx context.Context, req *tls.OpenDutPortRequest) (*tls.OpenDutPortResponse, error) {
	addr, err := lookupHost(req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	return &tls.OpenDutPortResponse{
		Address: addr,
		Port:    req.GetPort(),
	}, nil
}

func (s tlwServer) ExposePortToDut(ctx context.Context, req *tls.ExposePortToDutRequest) (*tls.ExposePortToDutResponse, error) {
	localServicePort := req.GetLocalPort()
	dutName := req.GetDutName()
	if dutName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "DutName cannot be empty")
	}
	addr, err := lookupHost(dutName)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, err.Error())
	}
	callerIP, err := getCallerIP(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, err.Error())
	}
	localService := net.JoinHostPort(callerIP, strconv.Itoa(int(localServicePort)))
	remoteDeviceClient, err := s.tPool.Get(net.JoinHostPort(addr, "22"))
	if err != nil {
		return nil, status.Errorf(codes.Aborted, err.Error())
	}
	t, err := s.tMgr.NewTunnel(localService, "127.0.0.1:0", remoteDeviceClient)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "Error setting up SSH tunnel: %s", err)
	}
	listenAddr := t.RemoteAddr().(*net.TCPAddr)
	response := &tls.ExposePortToDutResponse{
		ExposedAddress: listenAddr.IP.String(),
		ExposedPort:    int32(listenAddr.Port),
	}
	return response, nil
}

func (s tlwServer) CacheForDut(ctx context.Context, req *tls.CacheForDutRequest) (*longrunning.Operation, error) {
	rawURL := req.GetUrl()
	if rawURL == "" {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("CacheForDut: unsupported url %s in request", rawURL))
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("CacheForDut: unsupported url %s in request", rawURL))
	}
	dutName := req.GetDutName()
	if dutName == "" {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("CacheForDut: unsupported DutName %s in request", dutName))
	}
	op := s.lroMgr.NewOperation()
	go s.cache(ctx, parsedURL, dutName, op.Name)
	return op, status.Error(codes.OK, "Started: CacheForDut Operation.")
}

// cache implements the logic for the CacheForDut method and runs as a goroutine.
func (s *tlwServer) cache(ctx context.Context, parsedURL *url.URL, dutName, opName string) {
	log.Printf("CacheForDut: Started Operation = %v", opName)
	// Devserver URL to be used. In the "real" CacheForDut implementation,
	// devservers should be resolved here.
	const baseURL = "http://chromeos6-devserver2:8888/download/"
	if err := s.lroMgr.SetResult(opName, &tls.CacheForDutResponse{
		Url: baseURL + parsedURL.Host + parsedURL.Path,
	}); err != nil {
		log.Printf("CacheForDut: failed while updating result due to: %s", err)
	}
	log.Printf("CacheForDut: Operation Completed = %v", opName)
}

// lookupHost is a helper function that looks up the IP address of the provided
// host by using the local resolver.
func lookupHost(hostname string) (string, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("No IP addresses found for %s", hostname)
	}
	return addrs[0], nil
}

// getCallerIP gets the peer IP address from the provide context.
func getCallerIP(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("Error determining IP address")
	}
	callerIP, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return "", fmt.Errorf("Error determining IP address: %s", err)
	}
	return callerIP, nil
}

func getSSHClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(sshSigner)},
	}
}

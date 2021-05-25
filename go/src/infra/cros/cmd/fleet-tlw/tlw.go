// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"golang.org/x/crypto/ssh"

	"infra/cros/cmd/fleet-tlw/internal/cache"
	"infra/libs/lro"
	"infra/libs/sshpool"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

type tlwServer struct {
	tls.UnimplementedWiringServer
	lroMgr    *lro.Manager
	tMgr      *tunnelManager
	dutPool   *sshpool.Pool
	proxyPool *sshpool.Pool
	cFrontend *cache.Frontend
	ufsClient ufsapi.FleetClient
}

func newTLWServer(ufsService string, proxySSHSigner ssh.Signer, serviceAcctJSON string) (*tlwServer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ufsClient, err := ufsapi.NewClient(ctx, ufsapi.ServiceName(ufsService), ufsapi.ServiceAccountJSONPath(serviceAcctJSON), ufsapi.UserAgent("fleet-tlw/6.0.0"))
	if err != nil {
		return nil, errors.Reason("newTLWServer: %s", err).Err()
	}
	ce, err := cache.NewUFSEnv(ufsClient)
	if err != nil {
		return nil, errors.Reason("newTLWServer: %s", err).Err()
	}

	s := &tlwServer{
		lroMgr:    lro.New(),
		dutPool:   sshpool.New(getSSHClientConfig()),
		proxyPool: sshpool.New(getSSHClientConfigForProxy(proxySSHSigner)),
		tMgr:      newTunnelManager(),
		cFrontend: cache.NewFrontend(ce),
		ufsClient: ufsClient,
	}
	return s, nil
}

func (s *tlwServer) registerWith(g *grpc.Server) {
	tls.RegisterWiringServer(g, s)
	longrunning.RegisterOperationsServer(g, s.lroMgr)
}

// Close closes all open server resources.
func (s *tlwServer) Close() {
	s.tMgr.Close()
	s.dutPool.Close()
	s.proxyPool.Close()
	s.lroMgr.Close()
}

func (s *tlwServer) OpenDutPort(ctx context.Context, req *tls.OpenDutPortRequest) (*tls.OpenDutPortResponse, error) {
	addr, err := lookupHost(req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	return &tls.OpenDutPortResponse{
		Address: addr,
		Port:    req.GetPort(),
	}, nil
}

func (s *tlwServer) ExposePortToDut(ctx context.Context, req *tls.ExposePortToDutRequest) (*tls.ExposePortToDutResponse, error) {
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
	if req.GetRequireRemoteProxy() {
		exposedAddr, exposedPort, err := s.exposePortUsingProxy(addr, localService)
		if err != nil {
			return nil, status.Errorf(codes.Aborted, "Error setting up SSH tunnel to proxy: %s", err)
		}
		return &tls.ExposePortToDutResponse{
			ExposedAddress: exposedAddr,
			ExposedPort:    exposedPort,
		}, nil
	}
	exposedAddr, exposedPort, err := s.exposePortDirectly(addr, localService)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "Error setting up SSH tunnel: %s", err)
	}
	return &tls.ExposePortToDutResponse{
		ExposedAddress: exposedAddr,
		ExposedPort:    exposedPort,
	}, nil
}

func (s *tlwServer) exposePortUsingProxy(dutAddr, localService string) (string, int32, error) {
	// Use cache.Frontend here since we are depending on the Virtual IPs of
	// the caching backends.
	// TODO(crbug/1145811) Refactor the code to create a new package
	// 'lab subnet' which both CacheForDut and ExposePortToDut can use.
	// The new package  'lab subnet' can accept an IP/subnet mask and return
	// the servers in that subnet. Then CacheForDut and ExposePortToDut can
	// define their own logic to select one from them.

	// Pass a random string so we can get an random proxy server.
	cachingURL, err := s.cFrontend.AssignBackend(dutAddr, strconv.Itoa(rand.Int()))
	if err != nil {
		return "", 0, err
	}
	// cFrontend returns a URL in the format http://<ip>:<port>. Extract the
	// <ip> from this URL.
	u, err := url.Parse(cachingURL)
	if err != nil {
		return "", 0, err
	}
	proxyServerIP, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return "", 0, err
	}
	remoteDeviceClient, err := s.proxyPool.Get(net.JoinHostPort(proxyServerIP, "2222"))
	if err != nil {
		return "", 0, err
	}
	t, err := s.tMgr.NewTunnel(localService, "127.0.0.1:0", remoteDeviceClient)
	if err != nil {
		return "", 0, err
	}
	return proxyServerIP, int32(t.RemoteAddr().(*net.TCPAddr).Port), nil

}

func (s *tlwServer) exposePortDirectly(dutAddr, localService string) (string, int32, error) {
	remoteDeviceClient, err := s.dutPool.Get(net.JoinHostPort(dutAddr, "22"))
	if err != nil {
		return "", 0, err
	}
	t, err := s.tMgr.NewTunnel(localService, "127.0.0.1:0", remoteDeviceClient)
	if err != nil {
		return "", 0, err
	}
	listenAddr := t.RemoteAddr().(*net.TCPAddr)
	return listenAddr.IP.String(), int32(listenAddr.Port), nil
}

func (s *tlwServer) CacheForDut(ctx context.Context, req *tls.CacheForDutRequest) (*longrunning.Operation, error) {
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
	addr, err := lookupHost(dutName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("CacheForDut: lookup IP of %q: %s", dutName, err.Error()))
	}
	log.Printf("CacheForDut: the IP of %q is %q", dutName, addr)
	op := s.lroMgr.NewOperation()
	go s.cache(context.TODO(), parsedURL, addr, op.Name)
	return op, status.Error(codes.OK, "Started: CacheForDut Operation.")
}

// cache implements the logic for the CacheForDut method and runs as a goroutine.
func (s *tlwServer) cache(ctx context.Context, parsedURL *url.URL, addr, opName string) {
	log.Printf("CacheForDut: Started Operation = %v", opName)

	path := fmt.Sprintf("%s%s", parsedURL.Host, parsedURL.Path)
	// TODO (guocb): return a url.URL instead of string.
	cs, err := s.cFrontend.AssignBackend(addr, path)
	if err != nil {
		log.Printf("CacheForDut: %s", err)
		if err := s.lroMgr.SetError(opName, status.New(codes.FailedPrecondition, err.Error())); err != nil {
			log.Printf("CacheForDut: failed to set error on %s: %s", opName, err)
		}
		return
	}

	u := fmt.Sprintf("%s/download/%s", strings.TrimSuffix(cs, "/"), path)
	log.Printf("CacheForDut: result URL: %s", u)
	if err := s.lroMgr.SetResult(opName, &tls.CacheForDutResponse{Url: u}); err != nil {
		log.Printf("CacheForDut: failed to set result on %s: %s", opName, err)
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

func getSSHClientConfigForProxy(sshSigner ssh.Signer) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            "chromeos-test",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(sshSigner)},
	}
}

func (s *tlwServer) GetDut(ctx context.Context, req *tls.GetDutRequest) (*tls.Dut, error) {
	name := req.GetName()
	if name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "GetDut: empty name in request")
	}
	ctx = setupUFSContext(ctx)
	licenses, err := getUFSDeviceLicenses(ctx, s.ufsClient, name)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "GetDut: %s", err.Error())
	}

	return &tls.Dut{
		Name:     name,
		Licenses: licenses,
	}, nil
}

func setupUFSContext(ctx context.Context) context.Context {
	md := metadata.Pairs(ufsUtil.Namespace, ufsUtil.OSNamespace)
	return metadata.NewOutgoingContext(ctx, md)
}

// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This package creates the configuration files for nginx and keepalived used
// in the caching backend in Chrome OS fleet labs.
package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/grpc/metadata"

	models "infra/unifiedfleet/api/v1/models"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

var (
	keepalivedConfigFilePath = flag.String("keepalived-conf", "/mnt/conf/keepalived/keepalived.conf", "Path to where keepalived.conf should be created.")
	nginxConfigFilePath      = flag.String("nginx-conf", "/mnt/conf/nginx/nginx.conf", "Path to where nginx.conf should be created.")
	serviceAccountJSONPath   = flag.String("service-account", "/creds/service_accounts/artifacts-downloader-service-account.json", "Path to the service account JSON key file.")
	ufsService               = flag.String("ufs-host", "ufs.api.cr.dev", "Host of the UFS service.")
	cacheSizeInGB            = flag.Uint("nginx-cache-size", 750, "The size of nginx cache in GB.")
	gsaServerCount           = flag.Uint("gsa-server-count", 1, "The number of upstream gs_archive_server instances to be added in nginx-conf.")
	gsaInitialPort           = flag.Uint("gsa-initial-port", 18000, "The port number for the first instance of the gs_archive_server in nginx.conf. Port number will increase by 1 for all subsequent entries.")
	keepalivedInterface      = flag.String("keepalived-interface", "bond0", "The interface keepalived listens on.")
)

var (
	nodeIP   = os.Getenv("NODE_IP")
	nodeName = os.Getenv("NODE_NAME")
)

func main() {
	if err := innerMain(); err != nil {
		log.Fatalf("Exiting due to an error: %s", err)
	}
	log.Printf("Exiting successfully")
}

func innerMain() error {
	flag.Parse()
	if nodeIP == "" {
		return fmt.Errorf("environment variable NODE_IP missing,")
	}
	if nodeName == "" {
		return fmt.Errorf("environment variable NODE_NAME missing,")
	}
	log.Println("Getting caching service information from UFS...")
	services, err := getCachingServices()
	if err != nil {
		return err
	}
	service, ok := findService(services, nodeIP, nodeName)
	if !ok {
		log.Println("Could not find caching service for this node in UFS")
		log.Println("Creating non-operational nginx.conf...")
		if err := ioutil.WriteFile(*nginxConfigFilePath, []byte(noOpNginxTemplate), 0644); err != nil {
			return err
		}
		log.Println("Creating non-operational keepalived.conf...")
		if err := ioutil.WriteFile(*keepalivedConfigFilePath, []byte(noOpKeepalivedTemplate), 0644); err != nil {
			return err
		}
		return nil
	}
	vip := nodeVirtualIP(service)
	n := nginxConfData{
		// TODO(sanikak): Define types to make the unit clearer.
		// E.g.  type gigabyte int.
		CacheSizeInGB:  int(*cacheSizeInGB),
		GSAServerCount: int(*gsaServerCount),
		GSAInitialPort: int(*gsaInitialPort),
		VirtualIP:      vip,
	}
	k := keepalivedConfData{
		VirtualIP: vip,
		Interface: *keepalivedInterface,
	}
	switch {
	case nodeIP == service.GetPrimaryNode() || nodeName == service.GetPrimaryNode():
		n.UpstreamHost = net.JoinHostPort(service.GetSecondaryNode(), strconv.Itoa(int(service.GetPort())))
		k.UnicastPeer = service.GetSecondaryNode()
		// Keepalived configuration uses the following non-inclusive language.
		k.State = "MASTER"
		k.Priority = 150
	case nodeIP == service.GetSecondaryNode() || nodeName == service.GetSecondaryNode():
		k.UnicastPeer = service.GetPrimaryNode()
		k.State = "BACKUP"
		k.Priority = 99
	default:
		return fmt.Errorf("node is neither the primary nor the secondary")
	}
	log.Println("Creating nginx.conf...")
	nData, err := buildConfig(nginxTemplate, n)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(*nginxConfigFilePath, []byte(nData), 0644); err != nil {
		return err
	}
	log.Println("Creating keepalived.conf...")
	kData, err := buildConfig(keepalivedTemplate, k)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(*keepalivedConfigFilePath, []byte(kData), 0644); err != nil {
		return err
	}
	return nil
}

// getCachingServiceFromUFS gets all caching services listed in the UFS.
func getCachingServices() ([]*models.CachingService, error) {
	ctx := context.Background()
	md := metadata.Pairs("namespace", "os")
	ctx = metadata.NewOutgoingContext(ctx, md)
	o := auth.Options{
		Method:                 auth.ServiceAccountMethod,
		ServiceAccountJSONPath: *serviceAccountJSONPath,
	}
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, o)
	hc, err := a.Client()
	if err != nil {
		return nil, fmt.Errorf("could not establish HTTP client: %s", err)
	}
	ic := ufsapi.NewFleetPRPCClient(&prpc.Client{
		C:    hc,
		Host: *ufsService,
		Options: &prpc.Options{
			UserAgent: "caching-backend/3.0.0", // Any UserAgent with version below 3.0.0 is unsupported by UFS.
		},
	})
	res, err := ic.ListCachingServices(ctx, &ufsapi.ListCachingServicesRequest{})
	if err != nil {
		return nil, fmt.Errorf("query to UFS failed: %s", err)
	}
	return res.GetCachingServices(), nil
}

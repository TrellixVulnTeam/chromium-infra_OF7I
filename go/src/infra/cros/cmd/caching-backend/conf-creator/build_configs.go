// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net"
	"strings"

	models "infra/unifiedfleet/api/v1/models"
)

// Below constants indicate the role a node has in the caching backend.
type role int

const (
	coordinator role = iota
	backup
)

// nginxConfData contains information about the node which is necessary to
// create the nginx.conf file.
type nginxConfData struct {
	UpstreamHost   string
	VirtualIP      string
	WorkerCount    uint
	CacheSizeInGB  int
	GSAServerCount int
	GSAInitialPort int
}

// Ports returns a slice of ports for the gs_archive_server upstream or backup
// list.
func (n nginxConfData) Ports() []int {
	var ports []int
	for i := 0; i < n.GSAServerCount; i += 1 {
		ports = append(ports, n.GSAInitialPort+i)
	}
	return ports
}

// keepalivedConfData contains information about the node which is necessary to
// create the keepalived.conf file.
type keepalivedConfData struct {
	Interface   string
	UnicastPeer string
	VirtualIP   string
	State       string
	Priority    int32
}

// buildConfig generates the final template data.
func buildConfig(configTmpl string, configData interface{}) (string, error) {
	var buf bytes.Buffer
	tmpl := template.Must(template.New("base").Parse(configTmpl))
	if err := tmpl.Execute(&buf, configData); err != nil {
		return "", fmt.Errorf("error while executing template: %s", err)
	}
	return buf.String(), nil
}

// findService finds the correct caching service for the current
// node from a list of caching services.
func findService(services []*models.CachingService, nodeIP, nodeName string) (_ *models.CachingService, ok bool) {
	for _, service := range services {
		if nodeIP == service.GetPrimaryNode() || nodeName == service.GetPrimaryNode() {
			return service, true
		}
		if nodeIP == service.GetSecondaryNode() || nodeName == service.GetSecondaryNode() {
			return service, true
		}
	}
	return nil, false
}

// nodeVirtualIP gets the virtual IP of the current node.
func nodeVirtualIP(service *models.CachingService) (string, error) {
	// The service name is in the format cachingservice/<hostname>. So do the
	// required string manipulation to obtain the name.
	splitName := strings.Split(service.GetName(), "/")
	name := splitName[len(splitName)-1]
	vip, err := lookupHost(name)
	if err != nil {
		return "", fmt.Errorf("get node virtual IP of %q: %s", name, err)
	}
	return vip, nil
}

// lookupHost looks up the IP address of the provided host by using the local
// resolver.
func lookupHost(hostname string) (string, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return "", fmt.Errorf("lookup IP of %q: %s", hostname, err)
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("lookup IP of %q: No addresses found", hostname)
	}
	return addrs[0], nil
}

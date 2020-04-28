// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"go.chromium.org/chromiumos/infra/proto/go/lab"
	authclient "go.chromium.org/luci/auth"
	gitilesapi "go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/server/auth"
	"infra/libs/cros/git"
	fleet "infra/libs/fleet/protos/go"
)

// Host, project and branch to get dhcpd.conf file
var host string = "chrome-internal.googlesource.com"
var project string = "chromeos/chromeos-admin"
var branch string = "master"
var path string = "puppet/modules/lab/files/dhcp-server/dhcpd.conf"

// GetHostname returns the hostname of input ChromeOSDevice.
func GetHostname(d *lab.ChromeOSDevice) string {
	switch t := d.GetDevice().(type) {
	case *lab.ChromeOSDevice_Dut:
		return d.GetDut().GetHostname()
	case *lab.ChromeOSDevice_Labstation:
		return d.GetLabstation().GetHostname()
	default:
		panic(fmt.Sprintf("Unknown device type: %v", t))
	}
}

// GetLocation attempts to parse the input string and return a Location object.
// Default location is updated with values from the string. This is done
// because the barcodes do not specify the complete location of the asset
func GetLocation(input string) (loc *fleet.Location) {
	//loc = c.defaultLocation()
	loc = &fleet.Location{}
	// Extract lab if it exists
	for _, exp := range labs {
		labStr := exp.FindString(input)
		if labStr != "" {
			loc.Lab = labStr
		}
	}
	// Extract row if it exists
	for _, exp := range rows {
		rowStr := exp.FindString(input)
		if rowStr != "" {
			loc.Row = num.FindString(rowStr)
			break
		}
	}
	// Extract rack if it exists
	for _, exp := range racks {
		rackStr := exp.FindString(input)
		if rackStr != "" {
			loc.Rack = num.FindString(rackStr)
			break
		}
	}
	// Extract position if it exists
	for _, exp := range hosts {
		positionStr := exp.FindString(input)
		if positionStr != "" {
			loc.Position = num.FindString(positionStr)
			break
		}
	}
	return loc
}

// GetMacHostMappingFromDHCPConf downloads the dhcp conf from chromeos-admin
// repo. Parses the file and returns Mac:Hostname mapping
func GetMacHostMappingFromDHCPConf(ctx context.Context) (map[string]string, error) {
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf, auth.WithScopes(authclient.OAuthScopeEmail, gitilesapi.OAuthScope))
	if err != nil {
		return nil, err
	}
	client, err := git.NewClient(ctx, &http.Client{Transport: t}, "", host, project, branch)
	if err != nil {
		return nil, err
	}
	res, err := client.GetFile(ctx, path)

	return getMacHostMapping(res), nil
}

// Extract macaddress:hostname mapping from dhcp conf file
func getMacHostMapping(conf string) map[string]string {
	/* Rough parser designed to only extract mac:host mappings
	 * using regex. There are 3 expressions to extract the info
	 * required. First expression re extracts a host configuration
	 * with mac address. Second one is to extract only hostname
	 * and the third extracts hardware ethernet mac address
	 */
	// (?s) include newline and white spaces
	// [^\#],[^\{] negated character class for # and {
	var re = regexp.MustCompile(`(?m)^[^\#\r\n]*host[^\{]*\{[^\}]*hardware` +
		` ethernet[^\}]*\}`)
	var hn = regexp.MustCompile(`host .*{`)
	var ma = regexp.MustCompile(`(?m)^[^\#\r\n]*hardware ethernet` +
		` ([a-fA-F0-9]{2}\:){5}[a-fA-F0-9]{2}[ \t]*;`)
	c := re.FindAllString(conf, -1)
	res := make(map[string]string)
	for _, ent := range c {
		hostname := hn.FindString(ent)
		hostname = strings.TrimSpace(hostname)
		hostname = strings.TrimLeft(hostname, "host ")
		hostname = strings.TrimRight(hostname, " {")
		hostname = strings.TrimSpace(hostname)
		mac := ma.FindString(ent)
		mac = strings.TrimSpace(mac)
		mac = strings.TrimLeft(mac, "hardware ethernet ")
		mac = strings.TrimRight(mac, ";")
		mac = strings.TrimSpace(mac)
		if hostname != "" && mac != "" {
			res[mac] = hostname
		}
	}
	return res
}

/* Regular expressions to match various parts of the input string - START */

var num = regexp.MustCompile(`[0-9]+`)

var labs = []*regexp.Regexp{
	regexp.MustCompile(`chromeos[\d]*`),
}

var rows = []*regexp.Regexp{
	regexp.MustCompile(`ROW[\d]*`),
	regexp.MustCompile(`row[\d]*`),
}

var racks = []*regexp.Regexp{
	regexp.MustCompile(`RACK[\d]*`),
	regexp.MustCompile(`rack[\d]*`),
}

var hosts = []*regexp.Regexp{
	regexp.MustCompile(`HOST[\d]*`),
	regexp.MustCompile(`host[\d]*`),
}

/* Regular expressions to match various parts of the input string - END */

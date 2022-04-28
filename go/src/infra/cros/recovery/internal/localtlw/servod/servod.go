// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package servod provides functions to manage connection and communication with servod daemon on servo-host.
package servod

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	xmlrpc_value "go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/localtlw/ssh"
	"infra/cros/recovery/internal/localtlw/xmlrpc"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
	"infra/libs/sshpool"
)

const (
	// Waiting 60 seconds when starting servod daemon.
	startServodTimeout = 60
	// Waiting 3 seconds when stopping servod daemon.
	stopServodTimeout = 3
	// GPIO control for USB device on servo-host
	ImageUsbkeyDev = "image_usbkey_dev"
	// GPIO control for USB multiplexer
	ImageUsbkeyDirection = "image_usbkey_direction"
	// GPIO control value that causes USB drive to be attached to DUT.
	ImageUsbkeyTowardsDUT = "dut_sees_usbkey"
)

// status of servod daemon on servo-host.
type status string

const (
	servodUndefined  status = "UNDEFINED"
	servodRunning    status = "RUNNING"
	servodStopping   status = "STOPPING"
	servodNotRunning status = "NOT_RUNNING"
)

// servod holds information to manage servod daemon.
type servod struct {
	// Servo-host hostname or IP address where servod daemon will be running.
	host string
	// Port allocated for running servod.
	port int32
	// Function to receive parameters to start servod.
	getParams func() ([]string, error)
	// Proxy offers forward tunnel connections via SSH to the servod instance on Servo-Host.
	// Labs are restricted to SSH connection and port 22.
	proxy *proxy
}

// Prepare prepares servod before call it to run commands.
// If servod is running: do nothing.
// If servod is not running: start servod.
func (s *servod) Prepare(ctx context.Context, pool *sshpool.Pool) error {
	stat, err := s.getStatus(ctx, pool)
	if err != nil {
		return errors.Annotate(err, "prepare servod").Err()
	}
	switch stat {
	case servodNotRunning:
		err = s.start(ctx, pool)
		if err != nil {
			return errors.Annotate(err, "prepare servod").Err()
		}
		return nil
	case servodRunning:
		return nil
	}
	return errors.Reason("prepare servod %s:%d: fail to start", s.host, s.port).Err()
}

// getStatus return status of servod daemon on the servo-host.
func (s *servod) getStatus(ctx context.Context, pool *sshpool.Pool) (status, error) {
	r := ssh.Run(ctx, pool, s.host, fmt.Sprintf("status servod PORT=%d", s.port))
	if r.ExitCode == 0 {
		if strings.Contains(strings.ToLower(r.Stdout), "start/running") {
			return servodRunning, nil
		} else if strings.Contains(strings.ToLower(r.Stdout), "stop/waiting") {
			return servodStopping, nil
		}
	} else if strings.Contains(strings.ToLower(r.Stderr), "unknown instance") {
		return servodNotRunning, nil
	}
	log.Debugf(ctx, "Status check: %s", r.Stderr)
	return servodUndefined, errors.Reason("servo status %q: fail to check status", s.host).Err()
}

// start starts servod daemon on servo-host.
func (s *servod) start(ctx context.Context, pool *sshpool.Pool) error {
	params, err := s.getParams()
	if err != nil {
		return errors.Annotate(err, "start servod").Err()
	}
	// check if the servo host is labstation.
	// TODO(@otabek): remove checking the labstation logic once data and params
	// will be passed by tlw.InitServodRequest struct.
	r := ssh.Run(ctx, pool, s.host, "cat /etc/lsb-release | grep CHROMEOS_RELEASE_BOARD")
	if r.ExitCode != 0 {
		return errors.Reason("start servod: checking chrome os relase board: %s", r.Stderr).Err()
	}
	cmd := strings.Join(append([]string{"start", "servod"}, params...), " ")
	if r := ssh.Run(ctx, pool, s.host, cmd); r.ExitCode != 0 {
		return errors.Reason("start servod: %s", r.Stderr).Err()
	}
	// Use servodtool to check whether the servod is started.
	log.Debugf(ctx, "Start servod: use servodtool to check and wait the servod on labstation device to be fully started.")
	if r := ssh.Run(ctx, pool, s.host, fmt.Sprintf("servodtool instance wait-for-active -p %d", s.port)); r.ExitCode != 0 {
		return errors.Reason("start servod: servodtool check: %s", r.Stderr).Err()
	}
	return nil
}

// Stop stops servod daemon on servo-host.
func (s *servod) Stop(ctx context.Context, pool *sshpool.Pool) error {
	r := ssh.Run(ctx, pool, s.host, fmt.Sprintf("stop servod PORT=%d", s.port))
	if r.ExitCode != 0 {
		log.Debugf(ctx, "stop servod: %s", r.Stderr)
		return errors.Reason("stop servod: %s", r.Stderr).Err()
	} else {
		// Wait to teardown the servod.
		log.Debugf(ctx, "Stop servod: waiting %d seconds to fully teardown the daemon.", stopServodTimeout)
		time.Sleep(stopServodTimeout * time.Second)
	}
	return nil
}

// Call performs execution commands by servod daemon by XMLRPC connection.
func (s *servod) Call(ctx context.Context, pool *sshpool.Pool, timeout time.Duration, method string, args []*xmlrpc_value.Value) (r *xmlrpc_value.Value, rErr error) {
	if s.proxy == nil {
		p, err := newProxy(pool, s.host, s.port)
		if err != nil {
			return nil, errors.Annotate(err, "call servod").Err()
		}
		s.proxy = p
	}
	newAddr := s.proxy.LocalAddr()
	host, portString, err := net.SplitHostPort(newAddr)
	if err != nil {
		return nil, errors.Annotate(err, "call servod %q", newAddr).Err()
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		return nil, errors.Annotate(err, "call servod %q", newAddr).Err()
	}
	c := xmlrpc.New(host, port)
	return Call(ctx, c, timeout, method, args)
}

// Close closes using resource.
func (s *servod) Close() error {
	if s.proxy != nil {
		if err := s.proxy.Close(); err != nil {
			return errors.Annotate(err, "close servod").Err()
		}
	}
	return nil
}

// Call calls xmlrpc service with provided method and arguments.
func Call(ctx context.Context, c *xmlrpc.XMLRpc, timeout time.Duration, method string, args []*xmlrpc_value.Value) (r *xmlrpc_value.Value, rErr error) {
	var iArgs []interface{}
	for _, ra := range args {
		iArgs = append(iArgs, ra)
	}
	call := xmlrpc.NewCallTimeout(timeout, method, iArgs...)
	val := &xmlrpc_value.Value{}
	if err := c.Run(ctx, call, val); err != nil {
		return nil, errors.Annotate(err, "call servod %q: %s", c.Addr(), method).Err()
	}
	return val, nil
}

// GenerateParams generates command's params based on options.
// Example output:
//  "BOARD=${VALUE}" - name of DUT board.
//  "MODEL=${VALUE}" - name of DUT model.
//  "PORT=${VALUE}" - port specified to run servod on servo-host.
//  "SERIAL=${VALUE}" - serial number of root servo.
//  "CONFIG=cr50.xml" - special config for extra ability of CR50.
//  "REC_MODE=1" - start servod in recovery-mode, if root device found then servod will start event not all components detected.
func GenerateParams(o *tlw.ServodOptions) []string {
	var parts []string
	if o == nil {
		return parts
	}
	if o.ServodPort > 0 {
		parts = append(parts, fmt.Sprintf("PORT=%d", o.ServodPort))
	}
	if o.DutBoard != "" {
		parts = append(parts, fmt.Sprintf("BOARD=%s", o.DutBoard))
		if o.DutModel != "" {
			parts = append(parts, fmt.Sprintf("MODEL=%s", o.DutModel))
		}
	}
	if o.ServoSerial != "" {
		parts = append(parts, fmt.Sprintf("SERIAL=%s", o.ServoSerial))
	}
	if o.ServoDual {
		parts = append(parts, "DUAL_V4=1")
	}
	if o.UseCr50Config {
		parts = append(parts, "CONFIG=cr50.xml")
	}
	if o.RecoveryMode {
		parts = append(parts, "REC_MODE=1")
	}
	return parts
}

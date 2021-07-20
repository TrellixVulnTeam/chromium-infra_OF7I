// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package tlslib provides the canonical implementation of a common TLS server.
package tlslib

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"golang.org/x/crypto/ssh"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// provisionFailed - A flag file to indicate provision failures.
	// The file's location in stateful means that on successful update
	// it will be removed.  Thus, if this file exists, it indicates that
	// we've tried and failed in a previous attempt to update.
	// The file will be created every time a OS provision is kicked off.
	provisionFailed = "/var/tmp/provision_failed"
)

func (s *Server) provision(req *tls.ProvisionDutRequest, opName string) {
	log.Printf("provision: started %v", opName)

	// Set a timeout for provisioning.
	// TODO(kimjae): Tie the context with timeout to op by passing to lroMgr.
	ctx, cancel := context.WithTimeout(s.ctx, time.Hour)
	defer cancel()

	defer func() {
		provisionDutCounter.Add(ctx, 1)
		log.Printf("provision: finished %v", opName)
	}()

	setError := func(opErr *status.Status) {
		if err := s.lroMgr.SetError(opName, opErr); err != nil {
			log.Printf("provision: failed to set Operation error, %s", err)
		}
	}

	p, err := newProvisionState(s, req)
	if err != nil {
		setError(newOperationError(
			codes.InvalidArgument,
			fmt.Sprintf("provision: failed to create provisionState, %s", err),
			tls.ProvisionDutResponse_REASON_INVALID_REQUEST.String()))
		return
	}

	// Verify that the DUT is reachable.
	addr, err := s.getSSHAddr(ctx, req.GetName())
	if err != nil {
		setError(newOperationError(
			codes.InvalidArgument,
			fmt.Sprintf("provision: DUT SSH address unattainable prior to provisioning, %s", err),
			tls.ProvisionDutResponse_REASON_INVALID_REQUEST.String()))
		return
	}

	// Connect to the DUT.
	disconnect, err := p.connect(ctx, addr)
	if err != nil {
		setError(newOperationError(
			codes.FailedPrecondition,
			fmt.Sprintf("provision: DUT unreachable prior to provisioning (SSH client), %s", err),
			tls.ProvisionDutResponse_REASON_DUT_UNREACHABLE_PRE_PROVISION.String()))
		return
	}
	defer disconnect()

	// Create a marker so the lab knows to repair the device on failure.
	if err := runCmd(p.c, "touch "+provisionFailed); err != nil {
		log.Printf("Failed to create provisionFailed file, %s", err)
	}

	// Provision the OS.
	select {
	case <-ctx.Done():
		setError(newOperationError(
			codes.DeadlineExceeded,
			"provision: timed out before provisioning OS",
			tls.ProvisionDutResponse_REASON_PROVISIONING_TIMEDOUT.String()))
		return
	default:
	}
	// Get the current builder path.
	// If the builder path is missing or fails to be retrieved, continue to provision.
	builderPath, err := getBuilderPath(p.c)
	if err != nil {
		log.Printf("provision: failed to get pre-provision builder path, %s", err)
	}
	// Only provision the OS if the DUT is not on the requested OS.
	if builderPath != p.targetBuilderPath {
		t := time.Now()
		if err := p.provisionOS(ctx); err != nil {
			setError(newOperationError(
				codes.Aborted,
				fmt.Sprintf("provision: failed to provision OS, %s", err),
				tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED.String()))
			return
		}
		log.Printf("provision: time to provision OS took %v", time.Since(t))

		t = time.Now()
		if !req.GetPreserveStateful() {
			if err := p.wipeStateful(ctx); err != nil {
				setError(newOperationError(
					codes.Aborted,
					fmt.Sprintf("provision: failed to wipe stateful, %s", err),
					tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED.String()))
				return
			}
			// After a reboot, need a new client connection.
			sshCtx, cancel := context.WithTimeout(ctx, 300*time.Second)
			defer cancel()

			disconnect, err := p.connect(sshCtx, addr)
			if err != nil {
				setError(newOperationError(
					codes.Aborted,
					fmt.Sprintf("provision: failed to connect to DUT after wipe reboot, %s", err),
					tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED.String()))
				return
			}
			defer disconnect()
		}
		log.Printf("provision: time to wipe stateful took %v", time.Since(t))

		t = time.Now()
		if err := p.provisionStateful(ctx); err != nil {
			setError(newOperationError(
				codes.Aborted,
				fmt.Sprintf("provision: failed to provision stateful, %s", err),
				tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED.String()))
			return
		}
		log.Printf("provision: time to provision stateful took %v", time.Since(t))

		// After a reboot, need a new client connection.
		sshCtx, cancel := context.WithTimeout(ctx, 300*time.Second)
		defer cancel()

		disconnect, err := p.connect(sshCtx, addr)
		if err != nil {
			setError(newOperationError(
				codes.Aborted,
				fmt.Sprintf("provision: failed to connect to DUT after reboot, %s", err),
				tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED.String()))
			return
		}
		defer disconnect()

		t = time.Now()
		if err := p.verifyOSProvision(); err != nil {
			setError(newOperationError(
				codes.Aborted,
				fmt.Sprintf("provision: failed to verify OS provision, %s", err),
				tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED.String()))
			return
		}
		log.Printf("provision: time to verify provision took %v", time.Since(t))
	} else {
		log.Printf("provision: Operation=%s skipped as DUT is already on builder path %s", opName, builderPath)
	}

	// Provision DLCs.
	select {
	case <-ctx.Done():
		setError(newOperationError(
			codes.DeadlineExceeded,
			"provision: timed out before provisioning DLCs",
			tls.ProvisionDutResponse_REASON_PROVISIONING_TIMEDOUT.String()))
		return
	default:
	}
	if err := p.provisionDLCs(ctx, req.GetDlcSpecs()); err != nil {
		setError(newOperationError(
			codes.Aborted,
			fmt.Sprintf("provision: failed to provision DLCs, %s", err),
			tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED.String()))
		return
	}

	if err := s.lroMgr.SetResult(opName, &tls.ProvisionDutResponse{}); err != nil {
		log.Printf("provision: failed to set Opertion result, %s", err)
	}
}

func (s *Server) provisionLacros(req *tls.ProvisionLacrosRequest, opName string) {
	log.Printf("provisionLacros: started %v", opName)
	defer log.Printf("provisionLacros: finished %v", opName)

	// Set a timeout for provisioning Lacros.
	// TODO(kimjae): Tie the context with timeout to op by passing to lroMgr.
	ctx, cancel := context.WithTimeout(s.ctx, time.Hour)
	defer cancel()

	setError := func(opErr *status.Status) {
		if err := s.lroMgr.SetError(opName, opErr); err != nil {
			log.Printf("provision: failed to set Operation error, %s", err)
		}
	}

	p, err := newProvisionLacrosState(s, req)
	if err != nil {
		setError(newOperationError(
			codes.InvalidArgument,
			fmt.Sprintf("provisionLacros: failed to create provisionLacrosState, %s", err),
			tls.ProvisionLacrosResponse_REASON_INVALID_REQUEST.String()))
		return
	}

	// Verify that the DUT is reachable.
	addr, err := s.getSSHAddr(ctx, req.GetName())
	if err != nil {
		setError(newOperationError(
			codes.InvalidArgument,
			fmt.Sprintf("provisionLacros: DUT SSH address unattainable prior to provisioning Lacros, %s", err),
			tls.ProvisionLacrosResponse_REASON_INVALID_REQUEST.String()))
		return
	}

	// Connect to the DUT.
	disconnect, err := p.connect(ctx, addr)
	if err != nil {
		setError(newOperationError(
			codes.FailedPrecondition,
			fmt.Sprintf("provisionLacros: DUT unreachable prior to provisioning Lacros (SSH client), %s", err),
			tls.ProvisionLacrosResponse_REASON_DUT_UNREACHABLE_PRE_PROVISION.String()))
		return
	}
	defer disconnect()

	// Provision Lacros onto the DUT.
	if err := p.provisionLacros(ctx); err != nil {
		setError(newOperationError(
			codes.Aborted,
			fmt.Sprintf("provisionLacros: failed to provision Lacros onto the DUT, %s", err),
			tls.ProvisionLacrosResponse_REASON_PROVISIONING_FAILED.String()))
		return
	}

	// Lacros provisioning was successful!
	if err := s.lroMgr.SetResult(opName, &tls.ProvisionLacrosResponse{}); err != nil {
		log.Printf("provisionLacros: failed to set Opertion result for %v, %s", opName, err)
	}
}

// runCmd interprets the given string command in a shell and returns the error if any.
func runCmd(c *ssh.Client, cmd string) error {
	s, err := c.NewSession()
	if err != nil {
		return fmt.Errorf("runCmd: failed to create session, %v", err)
	}
	defer s.Close()

	s.Stdout = os.Stdout
	s.Stderr = os.Stderr

	log.Printf("Running command: %s", cmd)
	err = s.Run(cmd)

	if err != nil {
		return fmt.Errorf("runCmd: failed to run command, %v", err)
	}

	return nil
}

// runCmdOutput interprets the given string command in a shell and returns stdout.
func runCmdOutput(c *ssh.Client, cmd string) (string, error) {
	s, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer s.Close()

	stdoutBuf := new(strings.Builder)

	s.Stdout = io.MultiWriter(os.Stdout, stdoutBuf)
	s.Stderr = os.Stderr

	log.Printf("Running command: %s", cmd)
	err = s.Run(cmd)

	stdoutStr := stdoutBuf.String()
	if err != nil {
		return "", fmt.Errorf("runCmd: failed to run command, %v", err)
	}

	return stdoutStr, err
}

// newOperationError is a helper in creating Operation_Error and marshals ErrorInfo.
func newOperationError(c codes.Code, msg, reason string) *status.Status {
	s := status.New(c, msg)
	s, err := s.WithDetails(&errdetails.ErrorInfo{
		Reason: reason,
	})
	if err != nil {
		panic("Failed to set status details")
	}
	return s
}

func pathExists(c *ssh.Client, path string) (bool, error) {
	exists, err := runCmdOutput(c, fmt.Sprintf("[ -e %s ] && echo -n 1 || echo -n 0", path))
	if err != nil {
		return false, fmt.Errorf("path exists: failed to check if %s exists, %s", path, err)
	}
	return exists == "1", nil
}

// stopSystemDaemon stops system daemons than can interfere with provisioning.
func stopSystemDaemons(c *ssh.Client) {
	if err := runCmd(c, "stop ui"); err != nil {
		log.Printf("Stop system daemon: failed to stop UI daemon, %s", err)
	}
	if err := runCmd(c, "stop update-engine"); err != nil {
		log.Printf("Stop system daemon: failed to stop update-engine daemon, %s", err)
	}
}

func clearTPM(c *ssh.Client) error {
	return runCmd(c, "crossystem clear_tpm_owner_request=1")
}

func rebootDUT(ctx context.Context, c *ssh.Client) error {
	// Reboot, ignoring the SSH disconnection.
	_ = runCmd(c, "reboot")

	// Wait so following commands don't run before an actual reboot has kicked off
	// by waiting for the client connection to shutdown or a timeout.
	wait := make(chan interface{})
	go func() {
		_ = c.Wait()
		close(wait)
	}()
	select {
	case <-wait:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("rebootDUT: timeout waiting waiting for reboot")
	}
}

func runLabMachineAutoReboot(c *ssh.Client) {
	const (
		labMachineFile = statefulPath + "/.labmachine"
	)
	err := runCmd(c, fmt.Sprintf("FILE=%s ; [ -f $FILE ] || ( touch $FILE ; start autoreboot )", labMachineFile))
	if err != nil {
		log.Printf("run lab machine autoreboot: failed to run autoreboot, %s", err)
	}
}

var reBuilderPath = regexp.MustCompile(`CHROMEOS_RELEASE_BUILDER_PATH=(.*)`)

func getBuilderPath(c *ssh.Client) (string, error) {
	// Read the /etc/lsb-release file.
	lsbRelease, err := runCmdOutput(c, "cat /etc/lsb-release")
	if err != nil {
		return "", fmt.Errorf("get builder path: %s", err)
	}

	// Find the os version within the /etc/lsb-release file.
	match := reBuilderPath.FindStringSubmatch(lsbRelease)
	if match == nil {
		return "", errors.New("get builder path: no builder path found in lsb-release")
	}
	return match[1], nil
}

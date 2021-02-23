// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package tlslib provides the canonical implementation of a common TLS server.
package tlslib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"golang.org/x/crypto/ssh"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			tls.ProvisionDutResponse_REASON_INVALID_REQUEST))
		return
	}

	// Verify that the DUT is reachable.
	addr, err := s.getSSHAddr(ctx, req.GetName())
	if err != nil {
		setError(newOperationError(
			codes.InvalidArgument,
			fmt.Sprintf("provision: DUT SSH address unattainable prior to provisioning, %s", err),
			tls.ProvisionDutResponse_REASON_INVALID_REQUEST))
		return
	}

	// Connect to the DUT.
	disconnect, err := p.connect(ctx, addr)
	if err != nil {
		setError(newOperationError(
			codes.FailedPrecondition,
			fmt.Sprintf("provision: DUT unreachable prior to provisioning (SSH client), %s", err),
			tls.ProvisionDutResponse_REASON_DUT_UNREACHABLE_PRE_PROVISION))
		return
	}
	defer disconnect()

	// Provision the OS.
	select {
	case <-ctx.Done():
		setError(newOperationError(
			codes.DeadlineExceeded,
			"provision: timed out before provisioning OS",
			tls.ProvisionDutResponse_REASON_PROVISIONING_TIMEDOUT))
		return
	default:
	}
	// Get the current builder path.
	builderPath, err := getBuilderPath(p.c)
	if err != nil {
		setError(newOperationError(
			codes.Aborted,
			fmt.Sprintf("provision: failed to get the builder path from DUT, %s", err),
			tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED))
		return
	}
	// Only provision the OS if the DUT is not on the requested OS.
	if builderPath != p.targetBuilderPath {
		if err := p.provisionOS(ctx); err != nil {
			setError(newOperationError(
				codes.Aborted,
				fmt.Sprintf("provision: failed to provision OS, %s", err),
				tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED))
			return
		}

		// After a reboot, need a new client connection.
		sshCtx, cancel := context.WithTimeout(context.TODO(), 300*time.Second)
		defer cancel()

		disconnect, err := p.connect(sshCtx, addr)
		if err != nil {
			setError(newOperationError(
				codes.Aborted,
				fmt.Sprintf("provision: failed to connect to DUT after reboot, %s", err),
				tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED))
			return
		}
		defer disconnect()

		if err := p.verifyOSProvision(); err != nil {
			setError(newOperationError(
				codes.Aborted,
				fmt.Sprintf("provision: failed to verify OS provision, %s", err),
				tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED))
			return
		}
	} else {
		log.Printf("provision: Operation=%s skipped as DUT is already on builder path %s", opName, builderPath)
	}

	// Provision DLCs.
	select {
	case <-ctx.Done():
		setError(newOperationError(
			codes.DeadlineExceeded,
			"provision: timed out before provisioning DLCs",
			tls.ProvisionDutResponse_REASON_PROVISIONING_TIMEDOUT))
		return
	default:
	}
	if err := p.provisionDLCs(ctx, req.GetDlcSpecs()); err != nil {
		setError(newOperationError(
			codes.Aborted,
			fmt.Sprintf("provision: failed to provision DLCs, %s", err),
			tls.ProvisionDutResponse_REASON_PROVISIONING_FAILED))
		return
	}

	if err := s.lroMgr.SetResult(opName, &tls.ProvisionDutResponse{}); err != nil {
		log.Printf("provision: failed to set Opertion result, %s", err)
	}
}

// runCmd interprets the given string command in a shell and returns the error if any.
func runCmd(c *ssh.Client, cmd string) error {
	s, err := c.NewSession()
	if err != nil {
		return err
	}
	defer s.Close()
	b, err := s.CombinedOutput(cmd)
	if err != nil {
		err = fmt.Errorf("runCmd: %v, output: %q", err, b)
	}
	return err
}

// runCmdOutput interprets the given string command in a shell and returns stdout.
func runCmdOutput(c *ssh.Client, cmd string) (string, error) {
	s, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	b, err := s.Output(cmd)
	return string(b), err
}

// newOperationError is a helper in creating Operation_Error and marshals ErrorInfo.
func newOperationError(c codes.Code, msg string, reason tls.ProvisionDutResponse_Reason) *status.Status {
	s := status.New(c, msg)
	s, err := s.WithDetails(&errdetails.ErrorInfo{
		Reason: reason.String(),
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

func rebootDUT(c *ssh.Client) error {
	// Reboot in the background, giving time for the ssh invocation to cleanly terminate.
	return runCmd(c, "(sleep 2 && reboot) &")
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

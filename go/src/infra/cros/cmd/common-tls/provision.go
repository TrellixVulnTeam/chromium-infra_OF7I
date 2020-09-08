// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"golang.org/x/crypto/ssh"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
)

var rePartitionNumber = regexp.MustCompile(`.*([0-9]+)`)
var reBuilderPath = regexp.MustCompile(`CHROMEOS_RELEASE_BUILDER_PATH=(.*)`)

// runCmd interprets the given string command in a shell and returns the error if any.
func runCmd(c *client, cmd string) error {
	s, err := c.NewSession()
	if err != nil {
		return err
	}
	defer s.Close()
	err = s.Run(cmd)
	switch err.(type) {
	case *ssh.ExitError, *ssh.ExitMissingError:
		c.knownGood = true
	default:
		c.knownGood = false
	}
	return err
}

// runCmdOutput interprets the given string command in a shell and returns stdout and error if any.
func runCmdOutput(c *client, cmd string) (string, error) {
	s, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	b, err := s.Output(cmd)
	switch err.(type) {
	case *ssh.ExitError, *ssh.ExitMissingError:
		c.knownGood = true
	default:
		c.knownGood = false
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// newOperationResponse is a helper in creating Operation_Response and marshals ProvisionResponse.
func newOperationResponse() *longrunning.Operation_Response {
	a, _ := ptypes.MarshalAny(&tls.ProvisionResponse{})
	return &longrunning.Operation_Response{
		Response: a,
	}
}

// newOperationError is a helper in creating Operation_Error and marshals ErrorInfo.
func newOperationError(c codes.Code, msg string, reason tls.ProvisionResponse_Reason) *longrunning.Operation_Error {
	errInfo := &errdetails.ErrorInfo{
		Reason: reason.String(),
	}
	a, _ := ptypes.MarshalAny(errInfo)
	return &longrunning.Operation_Error{
		Error: &longrunning.Status{
			Code:    int32(c),
			Message: msg,
			Details: []*any.Any{a},
		},
	}
}

// stopSystemDaemon stops system daemons than can interfere with provisioning.
func stopSystemDaemons(c *client) {
	if err := runCmd(c, "stop ui"); err != nil {
		log.Printf("Stop system daemon: failed to stop UI daemon, %s", err)
	}
	if err := runCmd(c, "stop update-engine"); err != nil {
		log.Printf("Stop system daemon: failed to stop update-engine daemon, %s", err)
	}
}

func getBuilderPath(c *client) (string, error) {
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

// getGsCacheURL returns the http URL of GS Cache server appended with imagePath as the path.
func getGsCacheURL(imagePath string) (string, error) {
	gsURL, err := url.Parse("http://chromeos6-devserver4:8888/download")
	if err != nil {
		return "", fmt.Errorf("get GS Cache URL: %s", err)
	}
	u, err := url.Parse(imagePath)
	if err != nil {
		return "", fmt.Errorf("get GS Cache URL: %s", err)
	}
	gsURL.Path = path.Join(gsURL.Path, u.Host, u.Path)
	return gsURL.String(), nil
}

// rootDev holds root device related information.
type rootDev struct {
	disk      string
	partDelim string
	partNum   string
}

func getRootDev(c *client) (rootDev, error) {
	var r rootDev
	// Example 1: "/dev/nvme0n1p3"
	// Example 2: "/dev/sda3"
	curRoot, err := runCmdOutput(c, "rootdev -s")
	if err != nil {
		return r, fmt.Errorf("get root device: failed to get current root, %s", err)
	}
	curRoot = strings.TrimSpace(curRoot)

	// Example 1: "/dev/nvme0n1"
	// Example 2: "/dev/sda"
	rootDisk, err := runCmdOutput(c, "rootdev -s -d")
	if err != nil {
		return r, fmt.Errorf("get root device: failed to get root disk, %s", err)
	}
	r.disk = strings.TrimSpace(rootDisk)

	// Handle /dev/mmcblk0pX, /dev/sdaX, etc style partitions.
	// Example 1: "3"
	// Example 2: "3"
	match := rePartitionNumber.FindStringSubmatch(curRoot)
	if match == nil {
		return r, fmt.Errorf("get root device: failed to match partition number from %s", curRoot)
	}
	r.partNum = match[1]

	// Example 1: "p3"
	// Example 2: "3"
	rootPartNumWithDelim := strings.TrimPrefix(curRoot, r.disk)

	// Example 1: "p"
	// Example 2: ""
	r.partDelim = strings.TrimSuffix(rootPartNumWithDelim, r.partNum)

	return r, nil
}

const (
	partitionNumKernelA = "2"
	partitionNumKernelB = "4"
	partitionNumRootA   = "3"
	partitionNumRootB   = "5"
)

type partitionInfo struct {
	// The active + inactive kernel device partitions (e.g. /dev/nvme0n1p2).
	activeKernel   string
	inactiveKernel string
	// The active + inactive root device partitions (e.g. /dev/nvme0n1p3).
	activeRoot   string
	inactiveRoot string
}

// getPartitions returns the next kernel and root partitions to update.
func getPartitions(c *client) (partitionInfo, error) {
	var partInfo partitionInfo
	r, err := getRootDev(c)
	if err != nil {
		return partInfo, fmt.Errorf("get partitions to update: failed to get root device, %s", err)
	}

	// Determine the next kernel and root.
	rootDiskPartDelim := r.disk + r.partDelim
	switch r.partNum {
	case partitionNumRootA:
		return partitionInfo{
			activeKernel:   rootDiskPartDelim + partitionNumKernelA,
			inactiveKernel: rootDiskPartDelim + partitionNumKernelB,
			activeRoot:     rootDiskPartDelim + partitionNumRootA,
			inactiveRoot:   rootDiskPartDelim + partitionNumRootB,
		}, nil
	case partitionNumRootB:
		return partitionInfo{
			activeKernel:   rootDiskPartDelim + partitionNumKernelB,
			inactiveKernel: rootDiskPartDelim + partitionNumKernelA,
			activeRoot:     rootDiskPartDelim + partitionNumRootB,
			inactiveRoot:   rootDiskPartDelim + partitionNumRootA,
		}, nil
	default:
		return partInfo, fmt.Errorf("get partitions to update: unexpected root partition number of %s", r.partNum)
	}
}

const (
	fetchUngzipConvertCmd = "curl %s | gzip -d | dd of=%s obs=2M"
)

// installKernel updates kernelPartition on disk.
func installKernel(c *client, imagePath, kernPartition string) error {
	// TODO(crbug.com/1077056): Use CacheForDut from TLW server for images that
	// need to be fetched. (e.g. kernel, root, stateful, DLCs, etc)
	pathPrefix, err := getGsCacheURL(imagePath)
	if err != nil {
		return fmt.Errorf("install kernel: failed to get GS Cache URL, %s", err)
	}
	return runCmd(c, fmt.Sprintf(fetchUngzipConvertCmd, path.Join(pathPrefix, "full_dev_part_KERN.bin.gz"), kernPartition))
}

// installRoot updates rootPartition on disk.
func installRoot(c *client, imagePath, rootPartition string) error {
	// TODO(crbug.com/1077056): Use CacheForDut from TLW server for images that
	// need to be fetched. (e.g. kernel, root, stateful, DLCs, etc)
	pathPrefix, err := getGsCacheURL(imagePath)
	if err != nil {
		return fmt.Errorf("install root: failed to get GS Cache URL, %s", err)
	}
	return runCmd(c, fmt.Sprintf(fetchUngzipConvertCmd, path.Join(pathPrefix, "full_dev_part_ROOT.bin.gz"), rootPartition))
}

const (
	statefulPath           = "/mnt/stateful_partition"
	updateStatefulFilePath = statefulPath + "/.update_available"
)

// installStateful updates the stateful partition on disk (finalized after a reboot).
func installStateful(c *client, imagePath string) error {
	// TODO(crbug.com/1077056): Use CacheForDut from TLW server for images that
	// need to be fetched. (e.g. kernel, root, stateful, DLCs, etc)
	pathPrefix, err := getGsCacheURL(imagePath)
	if err != nil {
		return fmt.Errorf("install stateful: failed to get GS Cache URL, %s", err)
	}
	return runCmd(c, strings.Join([]string{
		fmt.Sprintf("rm -rf %[1]s %[2]s/var_new %[2]s/dev_image_new", updateStatefulFilePath, statefulPath),
		fmt.Sprintf("curl %s | tar --ignore-command-error --overwrite --directory=%s -xzf -", path.Join(pathPrefix, "stateful.tgz"), statefulPath),
		fmt.Sprintf("echo -n clobber > %s", updateStatefulFilePath),
	}, " && "))
}

func revertStatefulInstall(c *client) {
	const (
		varNew      = "var_new"
		devImageNew = "dev_image_new"
	)
	err := runCmd(c, fmt.Sprintf("rm -rf %s %s %s", path.Join(statefulPath, varNew), path.Join(statefulPath, devImageNew), updateStatefulFilePath))
	if err != nil {
		log.Printf("revert stateful install: failed to revert stateful installation, %s", err)
	}
}

func installPartitions(c *client, imagePath string, partitions partitionInfo) []error {
	kernelErr := make(chan error)
	rootErr := make(chan error)
	statefulErr := make(chan error)
	go func() {
		kernelErr <- installKernel(c, imagePath, partitions.inactiveKernel)
	}()
	go func() {
		rootErr <- installRoot(c, imagePath, partitions.inactiveRoot)
	}()
	go func() {
		statefulErr <- installStateful(c, imagePath)
	}()

	var provisionErrs []error
	if err := <-kernelErr; err != nil {
		provisionErrs = append(provisionErrs, err)
	}
	if err := <-rootErr; err != nil {
		provisionErrs = append(provisionErrs, err)
	}
	if err := <-statefulErr; err != nil {
		revertStatefulInstall(c)
		provisionErrs = append(provisionErrs, err)
	}
	return provisionErrs
}

func postInstall(c *client, partitions partitionInfo) error {
	return runCmd(c, strings.Join([]string{
		"tmpmnt=$(mktemp -d)",
		fmt.Sprintf("mount -o ro %s ${tmpmnt}", partitions.inactiveRoot),
		fmt.Sprintf("${tmpmnt}/postinst %s", partitions.inactiveRoot),
		"{ umount ${tmpmnt} || true; }",
		"{ rmdir ${tmpmnt} || true; }",
	}, " && "))
}

func revertPostInstall(c *client, partitions partitionInfo) {
	if err := runCmd(c, fmt.Sprintf("/postinst %s 2>&1", partitions.activeRoot)); err != nil {
		log.Printf("revert post install: failed to revert postinst, %s", err)
	}
}

func clearTPM(c *client) error {
	return runCmd(c, "crossystem clear_tpm_owner_request=1")
}

func runLabMachineAutoReboot(c *client) {
	const (
		labMachineFile = statefulPath + "/.labmachine"
	)
	err := runCmd(c, fmt.Sprintf("FILE=%s ; [ -f $FILE ] || ( touch $FILE ; start autoreboot )", labMachineFile))
	if err != nil {
		log.Printf("run lab machine autoreboot: failed to run autoreboot, %s", err)
	}
}

func rebootDUT(c *client) error {
	// Reboot in the background, giving time for the ssh invocation to cleanly terminate.
	return runCmd(c, "(sleep 2 && reboot) &")
}

func revertProvisionOS(c *client, partitions partitionInfo) {
	revertStatefulInstall(c)
	revertPostInstall(c, partitions)
}

// provisionOS will provision the OS, but on failure it will set op.Result to longrunning.Operation_Error
// and return the same error message
func provisionOS(c *client, op *longrunning.Operation, imagePath string) error {
	partitions, err := getPartitions(c)
	if err != nil {
		errMsg := fmt.Sprintf("provisionOS: failed to get kernel and root partitions, %s", err)
		op.Result = newOperationError(codes.Aborted, errMsg, tls.ProvisionResponse_REASON_PROVISIONING_FAILED)
		return errors.New(errMsg)
	}
	stopSystemDaemons(c)
	if errs := installPartitions(c, imagePath, partitions); len(errs) > 0 {
		errMsg := fmt.Sprintf("provisionOS: failed to provision the OS, %s", errs)
		op.Result = newOperationError(codes.Aborted, errMsg, tls.ProvisionResponse_REASON_PROVISIONING_FAILED)
		return errors.New(errMsg)
	}
	if err := postInstall(c, partitions); err != nil {
		revertProvisionOS(c, partitions)
		errMsg := fmt.Sprintf("provisionOS: failed to set next kernel, %s", err)
		op.Result = newOperationError(codes.Aborted, errMsg, tls.ProvisionResponse_REASON_PROVISIONING_FAILED)
		return errors.New(errMsg)
	}
	if err := clearTPM(c); err != nil {
		revertProvisionOS(c, partitions)
		errMsg := fmt.Sprintf("provisionOS: failed to clear TPM owner, %s", err)
		op.Result = newOperationError(codes.Aborted, errMsg, tls.ProvisionResponse_REASON_PROVISIONING_FAILED)
		return errors.New(errMsg)
	}
	runLabMachineAutoReboot(c)
	if err := rebootDUT(c); err != nil {
		revertProvisionOS(c, partitions)
		errMsg := fmt.Sprintf("provisionOS: failed reboot DUT, %s", err)
		op.Result = newOperationError(codes.Aborted, errMsg, tls.ProvisionResponse_REASON_PROVISIONING_FAILED)
		return errors.New(errMsg)
	}
	return nil
}

func verifyOSProvision(c *client, op *longrunning.Operation, expectedBuilderPath string) error {
	actualBuilderPath, err := getBuilderPath(c)
	if err != nil {
		errMsg := fmt.Sprintf("verify OS provision: failed to get builder path, %s", err)
		op.Result = newOperationError(codes.Aborted, errMsg, tls.ProvisionResponse_REASON_PROVISIONING_FAILED)
		return errors.New(errMsg)
	}
	if actualBuilderPath != expectedBuilderPath {
		errMsg := fmt.Sprintf("verify OS provision: DUT is on builder path %s when expected builder path is %s, %s",
			actualBuilderPath, expectedBuilderPath, err)
		op.Result = newOperationError(codes.Aborted, errMsg, tls.ProvisionResponse_REASON_PROVISIONING_FAILED)
		return errors.New(errMsg)
	}
	return nil
}

// parseImagePath on successfully parsing Image path oneof returns the path.
func parseImagePath(req *tls.ProvisionRequest) (string, error) {
	// Verify the incoming request path oneof is valid.
	switch t := req.GetImage().GetPathOneof().(type) {
	// Requests with gs_path_prefix should be in the format:
	// gs://chromeos-image-archive/eve-release/R86-13388.0.0
	case *tls.ProvisionRequest_ChromeOSImage_GsPathPrefix:
		return req.GetImage().GetGsPathPrefix(), nil
	default:
		return "", fmt.Errorf("parse image path oneof: unsupported ImagePathOneof in ProvisionRequest, %T", t)
	}
}

// parseTargetBuilderPath returns the Chrome OS builder path in the format a/xxx.yy.z
// Acceptable formats must include builder paths.
func parseTargetBuilderPath(imagePath string) (string, error) {
	u, uErr := url.Parse(imagePath)
	if uErr != nil {
		return "", fmt.Errorf("parse target builder path: failed to parse path %s, %s", imagePath, uErr)
	}
	d, version := path.Split(u.Path)
	return path.Join(path.Base(d), version), nil
}

func (s *server) provision(req *tls.ProvisionRequest, op *longrunning.Operation) {
	log.Printf("Provisioning: Started Operation=%v", op)

	// Set a timeout for provisioning.
	// TODO(kimjae): Tie the context with timeout to op.
	ctx, cancel := context.WithTimeout(context.TODO(), time.Hour)
	defer cancel()

	// Always sets the op to done and update the Operation with lroMgr.
	defer func() {
		op.Done = true
		if err := s.lroMgr.update(op); err != nil {
			log.Printf("Provision goroutine (not fatal): %s", err)
		}
		log.Printf("Provisioning: Finished Operation=%v", op)
	}()

	imagePath, err := parseImagePath(req)
	if err != nil {
		op.Result = newOperationError(
			codes.InvalidArgument,
			fmt.Sprintf("provision: unsupported ProvisionRequest_ChromeOSImage_PathOneof in request, %s", err),
			tls.ProvisionResponse_REASON_INVALID_REQUEST)
		return
	}

	targetBuilderPath, err := parseTargetBuilderPath(imagePath)
	if err != nil {
		op.Result = newOperationError(
			codes.InvalidArgument,
			fmt.Sprintf("provision: bad ProvisionRequest_ChromeOSImage_GsPathPrefix in request, %s", err),
			tls.ProvisionResponse_REASON_INVALID_REQUEST)
		return
	}

	// Verify that the DUT is reachable.
	addr, err := s.getSSHAddr(ctx, req.GetName())
	if err != nil {
		op.Result = newOperationError(
			codes.InvalidArgument,
			fmt.Sprintf("provision: DUT SSH address unattainable prior to provisioning, %s", err),
			tls.ProvisionResponse_REASON_INVALID_REQUEST)
		return
	}

	// Connect to the DUT.
	c, err := s.clientPool.Get(addr)
	if err != nil {
		op.Result = newOperationError(
			codes.FailedPrecondition,
			fmt.Sprintf("provision: DUT unreachable prior to provisioning (SSH client), %s", err),
			tls.ProvisionResponse_REASON_DUT_UNREACHABLE_PRE_PROVISION)
		return
	}
	c.knownGood = true
	defer s.clientPool.Put(addr, c)

	// Provision the OS.
	select {
	case <-ctx.Done():
		op.Result = newOperationError(
			codes.DeadlineExceeded,
			"provision: timed out before provisioning OS",
			tls.ProvisionResponse_REASON_PROVISIONING_TIMEDOUT)
		return
	default:
	}
	// Get the current builder path.
	builderPath, err := getBuilderPath(c)
	if err != nil {
		op.Result = newOperationError(
			codes.Aborted,
			fmt.Sprintf("provision: failed to get the builder path from DUT, %s", err),
			tls.ProvisionResponse_REASON_PROVISIONING_FAILED)
		return
	}
	// Only provision the OS if the DUT is not on the requested OS.
	if builderPath != targetBuilderPath {
		if err := provisionOS(c, op, imagePath); err != nil {
			return
		}
		// After a reboot, need a new client connection so close the old one.
		c.knownGood = false
		// Try to reconnect for a least 300 seconds.
		// TODO(kimjae): Make this connection verification into a function.
		for i := 0; i < 300; i++ {
			c, err = s.clientPool.Get(addr)
			if err != nil {
				// Try to reconnect again after a delay.
				time.Sleep(time.Second)
				continue
			}
			c.knownGood = true
			defer s.clientPool.Put(addr, c)
			break
		}
		if err := verifyOSProvision(c, op, targetBuilderPath); err != nil {
			return
		}
	} else {
		log.Printf("Provision skipped as DUT is already on builder path %s", builderPath)
	}

	// TODO(crbug.com/1077056): Provision DLCs.
}

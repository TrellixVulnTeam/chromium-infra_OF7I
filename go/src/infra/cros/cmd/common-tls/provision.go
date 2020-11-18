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

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"golang.org/x/crypto/ssh"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var rePartitionNumber = regexp.MustCompile(`.*([0-9]+)`)
var reBuilderPath = regexp.MustCompile(`CHROMEOS_RELEASE_BUILDER_PATH=(.*)`)

// runCmd interprets the given string command in a shell and returns the error if any.
func runCmd(c *ssh.Client, cmd string) error {
	s, err := c.NewSession()
	if err != nil {
		return err
	}
	defer s.Close()
	err = s.Run(cmd)
	return err
}

// runCmdOutput interprets the given string command in a shell and returns stdout and error if any.
func runCmdOutput(c *ssh.Client, cmd string) (string, error) {
	s, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	b, err := s.Output(cmd)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// newOperationError is a helper in creating Operation_Error and marshals ErrorInfo.
func newOperationError(c codes.Code, msg string, reason tls.ProvisionResponse_Reason) *status.Status {
	s := status.New(c, msg)
	s, err := s.WithDetails(&errdetails.ErrorInfo{
		Reason: reason.String(),
	})
	if err != nil {
		panic("Failed to set status details")
	}
	return s
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

func getRootDev(c *ssh.Client) (rootDev, error) {
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
	switch r.partNum {
	case partitionNumRootA, partitionNumRootB:
		break
	default:
		return r, fmt.Errorf("get root device: invalid partition number %s", r.partNum)
	}

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
func getPartitions(r rootDev) partitionInfo {
	// Determine the next kernel and root.
	rootDiskPartDelim := r.disk + r.partDelim
	switch r.partNum {
	case partitionNumRootA:
		return partitionInfo{
			activeKernel:   rootDiskPartDelim + partitionNumKernelA,
			inactiveKernel: rootDiskPartDelim + partitionNumKernelB,
			activeRoot:     rootDiskPartDelim + partitionNumRootA,
			inactiveRoot:   rootDiskPartDelim + partitionNumRootB,
		}
	case partitionNumRootB:
		return partitionInfo{
			activeKernel:   rootDiskPartDelim + partitionNumKernelB,
			inactiveKernel: rootDiskPartDelim + partitionNumKernelA,
			activeRoot:     rootDiskPartDelim + partitionNumRootB,
			inactiveRoot:   rootDiskPartDelim + partitionNumRootA,
		}
	default:
		panic(fmt.Sprintf("Unexpected root partition number of %s", r.partNum))
	}
}

// clearInactiveDLCVerifiedMarks will clear the verified marks for all DLCs in the inactive slots.
func clearInactiveDLCVerifiedMarks(c *ssh.Client, r rootDev) error {
	// Stop dlcservice daemon in order to not interfere with clearing inactive verified DLCs.
	if err := runCmd(c, "stop dlcservice"); err != nil {
		log.Printf("clear inactive verified DLC marks: failed to stop dlcservice daemon, %s", err)
	}
	defer func() {
		if err := runCmd(c, "start dlcservice"); err != nil {
			log.Printf("clear inactive verified DLC marks: failed to start dlcservice daemon, %s", err)
		}
	}()

	activeSlot := r.getActiveDLCSlot()
	inactiveSlot := getOtherDLCSlot(activeSlot)
	err := runCmd(c, fmt.Sprintf("rm -f %s", path.Join(dlcCacheDir, "*", "*", string(inactiveSlot), dlcVerified)))
	if err != nil {
		return fmt.Errorf("clear inactive verified DLC marks: failed remove inactive verified DLCs, %s", err)
	}

	return nil
}

const (
	fetchUngzipConvertCmd = "curl %s | gzip -d | dd of=%s obs=2M"
)

// installKernel updates kernelPartition on disk.
func installKernel(c *ssh.Client, imagePath, kernPartition string) error {
	// TODO(crbug.com/1077056): Use CacheForDut from TLW server for images that
	// need to be fetched. (e.g. kernel, root, stateful, DLCs, etc)
	pathPrefix, err := getGsCacheURL(imagePath)
	if err != nil {
		return fmt.Errorf("install kernel: failed to get GS Cache URL, %s", err)
	}
	return runCmd(c, fmt.Sprintf(fetchUngzipConvertCmd, path.Join(pathPrefix, "full_dev_part_KERN.bin.gz"), kernPartition))
}

// installRoot updates rootPartition on disk.
func installRoot(c *ssh.Client, imagePath, rootPartition string) error {
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
func installStateful(c *ssh.Client, imagePath string) error {
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

func revertStatefulInstall(c *ssh.Client) {
	const (
		varNew      = "var_new"
		devImageNew = "dev_image_new"
	)
	err := runCmd(c, fmt.Sprintf("rm -rf %s %s %s", path.Join(statefulPath, varNew), path.Join(statefulPath, devImageNew), updateStatefulFilePath))
	if err != nil {
		log.Printf("revert stateful install: failed to revert stateful installation, %s", err)
	}
}

func installPartitions(c *ssh.Client, imagePath string, partitions partitionInfo) []error {
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

func postInstall(c *ssh.Client, partitions partitionInfo) error {
	return runCmd(c, strings.Join([]string{
		"tmpmnt=$(mktemp -d)",
		fmt.Sprintf("mount -o ro %s ${tmpmnt}", partitions.inactiveRoot),
		fmt.Sprintf("${tmpmnt}/postinst %s", partitions.inactiveRoot),
		"{ umount ${tmpmnt} || true; }",
		"{ rmdir ${tmpmnt} || true; }",
	}, " && "))
}

func revertPostInstall(c *ssh.Client, partitions partitionInfo) {
	if err := runCmd(c, fmt.Sprintf("/postinst %s 2>&1", partitions.activeRoot)); err != nil {
		log.Printf("revert post install: failed to revert postinst, %s", err)
	}
}

func clearTPM(c *ssh.Client) error {
	return runCmd(c, "crossystem clear_tpm_owner_request=1")
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

func rebootDUT(c *ssh.Client) error {
	// Reboot in the background, giving time for the ssh invocation to cleanly terminate.
	return runCmd(c, "(sleep 2 && reboot) &")
}

func revertProvisionOS(c *ssh.Client, partitions partitionInfo) {
	revertStatefulInstall(c)
	revertPostInstall(c, partitions)
}

// provisionOS will provision the OS, but on failure it will set op.Result to longrunning.Operation_Error
// and return the same error message
func provisionOS(c *ssh.Client, imagePath string, r rootDev) error {
	stopSystemDaemons(c)

	// Only clear the inactive verified DLC marks if the DLCs exist.
	dlcsExist, err := pathExists(c, dlcLibDir)
	if err != nil {
		return fmt.Errorf("provisionOS: failed to check if DLC is enabled, %s", err)
	}
	if dlcsExist {
		if err := clearInactiveDLCVerifiedMarks(c, r); err != nil {
			return fmt.Errorf("provisionOS: failed to clear inactive verified DLC marks, %s", err)
		}
	}

	partitions := getPartitions(r)
	if errs := installPartitions(c, imagePath, partitions); len(errs) > 0 {
		return fmt.Errorf("provisionOS: failed to provision the OS, %s", errs)
	}
	if err := postInstall(c, partitions); err != nil {
		revertProvisionOS(c, partitions)
		return fmt.Errorf("provisionOS: failed to set next kernel, %s", err)
	}
	if err := clearTPM(c); err != nil {
		revertProvisionOS(c, partitions)
		return fmt.Errorf("provisionOS: failed to clear TPM owner, %s", err)
	}
	runLabMachineAutoReboot(c)
	if err := rebootDUT(c); err != nil {
		revertProvisionOS(c, partitions)
		return fmt.Errorf("provisionOS: failed reboot DUT, %s", err)
	}
	return nil
}

func verifyOSProvision(c *ssh.Client, expectedBuilderPath string) error {
	actualBuilderPath, err := getBuilderPath(c)
	if err != nil {
		return fmt.Errorf("verify OS provision: failed to get builder path, %s", err)
	}
	if actualBuilderPath != expectedBuilderPath {
		return fmt.Errorf("verify OS provision: DUT is on builder path %s when expected builder path is %s, %s",
			actualBuilderPath, expectedBuilderPath, err)
	}
	return nil
}

type dlcSlot string

const (
	dlcSlotA dlcSlot = "dlc_a"
	dlcSlotB dlcSlot = "dlc_b"
)

const (
	dlcCacheDir    = "/var/cache/dlc"
	dlcImage       = "dlc.img"
	dlcLibDir      = "/var/lib/dlcservice/dlc"
	dlcPackage     = "package"
	dlcVerified    = "verified"
	dlcserviceUtil = "dlcservice_util"
)

func pathExists(c *ssh.Client, path string) (bool, error) {
	exists, err := runCmdOutput(c, fmt.Sprintf("[ -e %s ] && echo -n 1 || echo -n 0", path))
	if err != nil {
		return false, fmt.Errorf("path exists: failed to check if %s exists, %s", path, err)
	}
	return exists == "1", nil
}

func (r rootDev) getActiveDLCSlot() dlcSlot {
	switch r.partNum {
	case partitionNumRootA:
		return dlcSlotA
	case partitionNumRootB:
		return dlcSlotB
	default:
		panic(fmt.Sprintf("Invalid partition number %s", r.partNum))
	}
}

func getOtherDLCSlot(slot dlcSlot) dlcSlot {
	switch slot {
	case dlcSlotA:
		return dlcSlotB
	case dlcSlotB:
		return dlcSlotA
	default:
		panic(fmt.Sprintf("Invalid DLC slot %s", slot))
	}
}

func isDLCVerified(c *ssh.Client, spec *tls.ProvisionRequest_DLCSpec, slot dlcSlot) (bool, error) {
	dlcID := spec.GetId()
	verified, err := pathExists(c, path.Join(dlcLibDir, dlcID, string(slot), dlcVerified))
	if err != nil {
		return false, fmt.Errorf("is DLC verfied: failed to check if DLC %s is verified, %s", dlcID, err)
	}
	return verified, nil
}

func installDLC(c *ssh.Client, spec *tls.ProvisionRequest_DLCSpec, imagePath, dlcOutputDir string, slot dlcSlot) error {
	verified, err := isDLCVerified(c, spec, slot)
	if err != nil {
		return fmt.Errorf("install DLC: failed is DLC verified check, %s", err)
	}

	dlcID := spec.GetId()
	// Skip installing the DLC if already verified.
	if verified {
		log.Printf("Provision DLC %s skipped as already verified", dlcID)
		return nil
	}

	// TODO(crbug.com/1077056): Use CacheForDut from TLW server for images that
	// need to be fetched. (e.g. kernel, root, stateful, DLCs, etc)
	pathPrefix, err := getGsCacheURL(imagePath)
	if err != nil {
		return fmt.Errorf("install DLC: failed to get GS Cache server, %s", err)
	}

	dlcOutputSlotDir := path.Join(dlcOutputDir, string(slot))
	dlcOutputImage := path.Join(dlcOutputSlotDir, dlcImage)
	dlcArtifactURL := path.Join(pathPrefix, "dlc", dlcID, dlcPackage, dlcImage)
	err = runCmd(c, fmt.Sprintf("mkdir -p %s && curl --output %s %s", dlcOutputSlotDir, dlcOutputImage, dlcArtifactURL))
	if err != nil {
		return fmt.Errorf("provision DLC: failed to provision DLC %s, %s", dlcID, err)
	}
	return nil
}

func provisionDLCs(c *ssh.Client, imagePath string, r rootDev, specs []*tls.ProvisionRequest_DLCSpec) error {
	if len(specs) == 0 {
		return nil
	}

	// Stop dlcservice daemon in order to not interfere with provisioning DLCs.
	if err := runCmd(c, "stop dlcservice"); err != nil {
		log.Printf("provision DLCs: failed to stop dlcservice daemon, %s", err)
	}
	defer func() {
		if err := runCmd(c, "start dlcservice"); err != nil {
			log.Printf("provision DLCs: failed to start dlcservice daemon, %s", err)
		}
	}()

	activeSlot := r.getActiveDLCSlot()
	errCh := make(chan error)
	for _, spec := range specs {
		go func(spec *tls.ProvisionRequest_DLCSpec) {
			dlcID := spec.GetId()
			if err := installDLC(c, spec, imagePath, path.Join(dlcCacheDir, dlcID, dlcPackage), activeSlot); err != nil {
				errMsg := fmt.Sprintf("failed to install DLC %s, %s", dlcID, err)
				errCh <- errors.New(errMsg)
				return
			}
			errCh <- nil
		}(spec)
	}

	var err error
	for range specs {
		errTmp := <-errCh
		if errTmp == nil {
			continue
		}
		err = fmt.Errorf("%s, %s", err, errTmp)
	}
	if err != nil {
		return fmt.Errorf("provision DLCs: failed to install the following DLCs (%s)", err)
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

func (s *server) provision(req *tls.ProvisionRequest, opName string) {
	log.Printf("provision: started %v", opName)
	defer func() {
		log.Printf("provision: finished %v", opName)
	}()

	setError := func(opErr *status.Status) {
		if err := s.lroMgr.SetError(opName, opErr); err != nil {
			log.Printf("provision: failed to set Operation error, %s", err)
		}
	}

	// Set a timeout for provisioning.
	// TODO(kimjae): Tie the context with timeout to op.
	ctx, cancel := context.WithTimeout(context.TODO(), time.Hour)
	defer cancel()

	imagePath, err := parseImagePath(req)
	if err != nil {
		setError(newOperationError(
			codes.InvalidArgument,
			fmt.Sprintf("provision: unsupported ProvisionRequest_ChromeOSImage_PathOneof in request, %s", err),
			tls.ProvisionResponse_REASON_INVALID_REQUEST))
		return
	}

	targetBuilderPath, err := parseTargetBuilderPath(imagePath)
	if err != nil {
		setError(newOperationError(
			codes.InvalidArgument,
			fmt.Sprintf("provision: bad ProvisionRequest_ChromeOSImage_GsPathPrefix in request, %s", err),
			tls.ProvisionResponse_REASON_INVALID_REQUEST))
		return
	}

	// Verify that the DUT is reachable.
	addr, err := s.getSSHAddr(ctx, req.GetName())
	if err != nil {
		setError(newOperationError(
			codes.InvalidArgument,
			fmt.Sprintf("provision: DUT SSH address unattainable prior to provisioning, %s", err),
			tls.ProvisionResponse_REASON_INVALID_REQUEST))
		return
	}

	// Connect to the DUT.
	c, err := s.clientPool.Get(addr)
	if err != nil {
		setError(newOperationError(
			codes.FailedPrecondition,
			fmt.Sprintf("provision: DUT unreachable prior to provisioning (SSH client), %s", err),
			tls.ProvisionResponse_REASON_DUT_UNREACHABLE_PRE_PROVISION))
		return
	}
	defer s.clientPool.Put(addr, c)

	// Get the root device.
	r, err := getRootDev(c)
	if err != nil {
		setError(newOperationError(
			codes.Aborted,
			fmt.Sprintf("provision: failed to get root device from DUT, %s", err),
			tls.ProvisionResponse_REASON_PROVISIONING_FAILED))
		return
	}

	// Provision the OS.
	select {
	case <-ctx.Done():
		setError(newOperationError(
			codes.DeadlineExceeded,
			"provision: timed out before provisioning OS",
			tls.ProvisionResponse_REASON_PROVISIONING_TIMEDOUT))
		return
	default:
	}
	// Get the current builder path.
	builderPath, err := getBuilderPath(c)
	if err != nil {
		setError(newOperationError(
			codes.Aborted,
			fmt.Sprintf("provision: failed to get the builder path from DUT, %s", err),
			tls.ProvisionResponse_REASON_PROVISIONING_FAILED))
		return
	}
	// Only provision the OS if the DUT is not on the requested OS.
	if builderPath != targetBuilderPath {
		if err := provisionOS(c, imagePath, r); err != nil {
			setError(newOperationError(
				codes.Aborted,
				fmt.Sprintf("provision: failed to provision OS, %s", err),
				tls.ProvisionResponse_REASON_PROVISIONING_FAILED))
			return
		}
		// After a reboot, need a new client connection so close the old one.

		// Try to reconnect for a least 300 seconds.
		// TODO(kimjae): Make this connection verification into a function.
		for i := 0; i < 300; i++ {
			c, err = s.clientPool.Get(addr)
			if err != nil {
				// Try to reconnect again after a delay.
				time.Sleep(time.Second)
				continue
			}
			defer s.clientPool.Put(addr, c)
			break
		}
		if err := verifyOSProvision(c, targetBuilderPath); err != nil {
			setError(newOperationError(
				codes.Aborted,
				fmt.Sprintf("provision: failed to verify OS provision, %s", err),
				tls.ProvisionResponse_REASON_PROVISIONING_FAILED))
			return
		}
		// Get the new root device after reboot.
		r, err = getRootDev(c)
		if err != nil {
			setError(newOperationError(
				codes.Aborted,
				fmt.Sprintf("provision: failed to get root device from DUT after reboot, %s", err),
				tls.ProvisionResponse_REASON_PROVISIONING_FAILED))
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
			tls.ProvisionResponse_REASON_PROVISIONING_TIMEDOUT))
		return
	default:
	}
	if err := provisionDLCs(c, imagePath, r, req.GetDlcSpecs()); err != nil {
		setError(newOperationError(
			codes.Aborted,
			fmt.Sprintf("provision: failed to provision DLCs, %s", err),
			tls.ProvisionResponse_REASON_PROVISIONING_FAILED))
		return
	}

	if err := s.lroMgr.SetResult(opName, &tls.ProvisionResponse{}); err != nil {
		log.Printf("provision: failed to set Opertion result, %s", err)
	}
}

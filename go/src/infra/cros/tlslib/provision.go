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

type provisionState struct {
	s                 *Server
	c                 *ssh.Client
	dutName           string
	imagePath         string
	targetBuilderPath string
}

var (
	rePartitionNumber = regexp.MustCompile(`.*([0-9]+)`)
	reBuilderPath     = regexp.MustCompile(`CHROMEOS_RELEASE_BUILDER_PATH=(.*)`)
)

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

// partitionsInfo holds active/inactive root + kernel partition information.
type partitionInfo struct {
	// The active + inactive kernel device partitions (e.g. /dev/nvme0n1p2).
	activeKernel   string
	inactiveKernel string
	// The active + inactive root device partitions (e.g. /dev/nvme0n1p3).
	activeRoot   string
	inactiveRoot string
}

func getPartitionInfo(r rootDev) partitionInfo {
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

	inactiveSlot := r.getInactiveDLCSlot()
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
func (p *provisionState) installKernel(ctx context.Context, pi partitionInfo) error {
	url, err := p.s.getCacheURL(ctx, path.Join(p.imagePath, "full_dev_part_KERN.bin.gz"), p.dutName)
	if err != nil {
		return fmt.Errorf("install kernel: failed to get GS Cache URL, %s", err)
	}
	return runCmd(p.c, fmt.Sprintf(fetchUngzipConvertCmd, url, pi.inactiveKernel))
}

// installRoot updates rootPartition on disk.
func (p *provisionState) installRoot(ctx context.Context, pi partitionInfo) error {
	url, err := p.s.getCacheURL(ctx, path.Join(p.imagePath, "full_dev_part_ROOT.bin.gz"), p.dutName)
	if err != nil {
		return fmt.Errorf("install root: failed to get GS Cache URL, %s", err)
	}
	return runCmd(p.c, fmt.Sprintf(fetchUngzipConvertCmd, url, pi.inactiveRoot))
}

const (
	statefulPath           = "/mnt/stateful_partition"
	updateStatefulFilePath = statefulPath + "/.update_available"
)

// installStateful updates the stateful partition on disk (finalized after a reboot).
func (p *provisionState) installStateful(ctx context.Context) error {
	url, err := p.s.getCacheURL(ctx, path.Join(p.imagePath, "stateful.tgz"), p.dutName)
	if err != nil {
		return fmt.Errorf("install stateful: failed to get GS Cache URL, %s", err)
	}
	return runCmd(p.c, strings.Join([]string{
		fmt.Sprintf("rm -rf %[1]s %[2]s/var_new %[2]s/dev_image_new", updateStatefulFilePath, statefulPath),
		fmt.Sprintf("curl %s | tar --ignore-command-error --overwrite --directory=%s -xzf -", url, statefulPath),
		fmt.Sprintf("echo -n clobber > %s", updateStatefulFilePath),
	}, " && "))
}

func (p *provisionState) revertStatefulInstall() {
	const (
		varNew      = "var_new"
		devImageNew = "dev_image_new"
	)
	varNewPath := path.Join(statefulPath, varNew)
	devImageNewPath := path.Join(statefulPath, devImageNew)
	err := runCmd(p.c, fmt.Sprintf("rm -rf %s %s %s", varNewPath, devImageNewPath, updateStatefulFilePath))
	if err != nil {
		log.Printf("revert stateful install: failed to revert stateful installation, %s", err)
	}
}

func (p *provisionState) installPartitions(ctx context.Context, pi partitionInfo) []error {
	kernelErr := make(chan error)
	rootErr := make(chan error)
	statefulErr := make(chan error)
	go func() {
		kernelErr <- p.installKernel(ctx, pi)
	}()
	go func() {
		rootErr <- p.installRoot(ctx, pi)
	}()
	go func() {
		statefulErr <- p.installStateful(ctx)
	}()

	var provisionErrs []error
	if err := <-kernelErr; err != nil {
		provisionErrs = append(provisionErrs, err)
	}
	if err := <-rootErr; err != nil {
		provisionErrs = append(provisionErrs, err)
	}
	if err := <-statefulErr; err != nil {
		p.revertStatefulInstall()
		provisionErrs = append(provisionErrs, err)
	}
	return provisionErrs
}

func (p *provisionState) postInstall(pi partitionInfo) error {
	return runCmd(p.c, strings.Join([]string{
		"tmpmnt=$(mktemp -d)",
		fmt.Sprintf("mount -o ro %s ${tmpmnt}", pi.inactiveRoot),
		fmt.Sprintf("${tmpmnt}/postinst %s", pi.inactiveRoot),
		"{ umount ${tmpmnt} || true; }",
		"{ rmdir ${tmpmnt} || true; }",
	}, " && "))
}

func (p *provisionState) revertPostInstall(pi partitionInfo) {
	if err := runCmd(p.c, fmt.Sprintf("/postinst %s 2>&1", pi.activeRoot)); err != nil {
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

func (p *provisionState) revertProvisionOS(pi partitionInfo) {
	p.revertStatefulInstall()
	p.revertPostInstall(pi)
}

// provisionOS will provision the OS, but on failure it will set op.Result to longrunning.Operation_Error
// and return the same error message
func (p *provisionState) provisionOS(ctx context.Context) error {
	r, err := getRootDev(p.c)
	if err != nil {
		return fmt.Errorf("installPartitions: failed to get root device from DUT, %s", err)
	}

	stopSystemDaemons(p.c)

	// Only clear the inactive verified DLC marks if the DLCs exist.
	dlcsExist, err := pathExists(p.c, dlcLibDir)
	if err != nil {
		return fmt.Errorf("provisionOS: failed to check if DLC is enabled, %s", err)
	}
	if dlcsExist {
		if err := clearInactiveDLCVerifiedMarks(p.c, r); err != nil {
			return fmt.Errorf("provisionOS: failed to clear inactive verified DLC marks, %s", err)
		}
	}

	pi := getPartitionInfo(r)
	if errs := p.installPartitions(ctx, pi); len(errs) > 0 {
		return fmt.Errorf("provisionOS: failed to provision the OS, %s", errs)
	}
	if err := p.postInstall(pi); err != nil {
		p.revertProvisionOS(pi)
		return fmt.Errorf("provisionOS: failed to set next kernel, %s", err)
	}
	if err := clearTPM(p.c); err != nil {
		p.revertProvisionOS(pi)
		return fmt.Errorf("provisionOS: failed to clear TPM owner, %s", err)
	}
	runLabMachineAutoReboot(p.c)
	if err := rebootDUT(p.c); err != nil {
		p.revertProvisionOS(pi)
		return fmt.Errorf("provisionOS: failed reboot DUT, %s", err)
	}
	return nil
}

func (p *provisionState) verifyOSProvision() error {
	actualBuilderPath, err := getBuilderPath(p.c)
	if err != nil {
		return fmt.Errorf("verify OS provision: failed to get builder path, %s", err)
	}
	if actualBuilderPath != p.targetBuilderPath {
		return fmt.Errorf("verify OS provision: DUT is on builder path %s when expected builder path is %s, %s",
			actualBuilderPath, p.targetBuilderPath, err)
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

func (r rootDev) getInactiveDLCSlot() dlcSlot {
	switch slot := r.getActiveDLCSlot(); slot {
	case dlcSlotA:
		return dlcSlotB
	case dlcSlotB:
		return dlcSlotA
	default:
		panic(fmt.Sprintf("Invalid DLC slot %s", slot))
	}
}

func isDLCVerified(c *ssh.Client, spec *tls.ProvisionDutRequest_DLCSpec, slot dlcSlot) (bool, error) {
	dlcID := spec.GetId()
	verified, err := pathExists(c, path.Join(dlcLibDir, dlcID, string(slot), dlcVerified))
	if err != nil {
		return false, fmt.Errorf("is DLC verfied: failed to check if DLC %s is verified, %s", dlcID, err)
	}
	return verified, nil
}

func (p *provisionState) installDLC(ctx context.Context, spec *tls.ProvisionDutRequest_DLCSpec, dlcOutputDir string, slot dlcSlot) error {
	verified, err := isDLCVerified(p.c, spec, slot)
	if err != nil {
		return fmt.Errorf("install DLC: failed is DLC verified check, %s", err)
	}

	dlcID := spec.GetId()
	// Skip installing the DLC if already verified.
	if verified {
		log.Printf("Provision DLC %s skipped as already verified", dlcID)
		return nil
	}

	dlcURL := path.Join(p.imagePath, "dlc", dlcID, dlcPackage, dlcImage)
	url, err := p.s.getCacheURL(ctx, dlcURL, p.dutName)
	if err != nil {
		return fmt.Errorf("install DLC: failed to get GS Cache server, %s", err)
	}

	dlcOutputSlotDir := path.Join(dlcOutputDir, string(slot))
	dlcOutputImage := path.Join(dlcOutputSlotDir, dlcImage)
	dlcCmd := fmt.Sprintf("mkdir -p %s && curl --output %s %s", dlcOutputSlotDir, dlcOutputImage, url)
	if err := runCmd(p.c, dlcCmd); err != nil {
		return fmt.Errorf("provision DLC: failed to provision DLC %s, %s", dlcID, err)
	}
	return nil
}

func (p *provisionState) provisionDLCs(ctx context.Context, specs []*tls.ProvisionDutRequest_DLCSpec) error {
	if len(specs) == 0 {
		return nil
	}

	// Stop dlcservice daemon in order to not interfere with provisioning DLCs.
	if err := runCmd(p.c, "stop dlcservice"); err != nil {
		log.Printf("provision DLCs: failed to stop dlcservice daemon, %s", err)
	}
	defer func() {
		if err := runCmd(p.c, "start dlcservice"); err != nil {
			log.Printf("provision DLCs: failed to start dlcservice daemon, %s", err)
		}
	}()

	var err error
	r, err := getRootDev(p.c)
	if err != nil {
		return fmt.Errorf("provision DLCs: failed to get root device from DUT, %s", err)
	}

	errCh := make(chan error)
	for _, spec := range specs {
		go func(spec *tls.ProvisionDutRequest_DLCSpec) {
			dlcID := spec.GetId()
			dlcOutputDir := path.Join(dlcCacheDir, dlcID, dlcPackage)
			if err := p.installDLC(ctx, spec, dlcOutputDir, r.getActiveDLCSlot()); err != nil {
				errMsg := fmt.Sprintf("failed to install DLC %s, %s", dlcID, err)
				errCh <- errors.New(errMsg)
				return
			}
			errCh <- nil
		}(spec)
	}

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

func (p *provisionState) connect(ctx context.Context, addr string) (func(), error) {
	c, err := p.s.clientPool.GetContext(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("connect: DUT unreachable, %s", err)
	}

	p.c = c
	disconnect := func(c *ssh.Client) func() {
		return func() {
			p.s.clientPool.Put(addr, c)
		}
	}(p.c)
	return disconnect, nil
}

func newProvisionState(s *Server, req *tls.ProvisionDutRequest) (*provisionState, error) {
	p := &provisionState{
		s:       s,
		dutName: req.Name,
	}

	// Verify the incoming request path oneof is valid.
	switch t := req.GetImage().GetPathOneof().(type) {
	// Requests with gs_path_prefix should be in the format:
	// gs://chromeos-image-archive/eve-release/R86-13388.0.0
	case *tls.ProvisionDutRequest_ChromeOSImage_GsPathPrefix:
		p.imagePath = req.GetImage().GetGsPathPrefix()
	default:
		return nil, fmt.Errorf("newProvisionState: unsupported ImagePathOneof in ProvisionDutRequest, %T", t)
	}

	u, uErr := url.Parse(p.imagePath)
	if uErr != nil {
		return nil, fmt.Errorf("setPaths: failed to parse path %s, %s", p.imagePath, uErr)
	}

	d, version := path.Split(u.Path)
	p.targetBuilderPath = path.Join(path.Base(d), version)

	return p, nil
}

func (s *Server) provision(req *tls.ProvisionDutRequest, opName string) {
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

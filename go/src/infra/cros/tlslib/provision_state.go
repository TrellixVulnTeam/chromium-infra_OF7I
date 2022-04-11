// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package tlslib provides the canonical implementation of a common TLS server.
package tlslib

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"path"
	"strings"
	"time"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"golang.org/x/crypto/ssh"
)

type provisionState struct {
	s                 *Server
	c                 *ssh.Client
	dutName           string
	imagePath         string
	targetBuilderPath string
	targetLsbHash     string
	forceProvisionOs  bool
	preventReboot     bool
}

func newProvisionState(s *Server, req *tls.ProvisionDutRequest) (*provisionState, error) {
	p := &provisionState{
		s:                s,
		dutName:          req.Name,
		forceProvisionOs: req.ForceProvisionOs,
		preventReboot:    req.PreventReboot,
	}

	// Verify the incoming request path oneof is valid.
	switch t := req.GetTargetBuild().GetPathOneof().(type) {
	// Requests with gs_path_prefix should be in the format:
	// gs://chromeos-image-archive/eve-release/R86-13388.0.0
	case *tls.ChromeOsImage_GsPathPrefix:
		p.imagePath = req.GetTargetBuild().GetGsPathPrefix()
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

func (p *provisionState) shouldProvisionOS() bool {
	// Get the current builder path.
	// If the builder path is missing or fails to be retrieved, continue to provision.
	builderPath, err := getBuilderPath(p.c)
	if err != nil {
		log.Printf("provision: failed to get pre-provision builder path, %s", err)
		return true
	}
	// Only provision the OS if any of the following are true:
	//  - the DUT is not on the requested OS.
	//  - the force marker exists on the DUT.
	//  - the force flag was used to provision.
	shouldProvision := false
	if builderPath != p.targetBuilderPath {
		log.Printf("Going to provision DUT from %s to %s", builderPath, p.targetBuilderPath)
		shouldProvision = true
	}
	if shouldForceProvision(p.c) || p.forceProvisionOs {
		if !shouldProvision {
			log.Printf("Going to force provision to %s", p.targetBuilderPath)
			shouldProvision = true
		} else {
			log.Printf("Ignoring force provision as already provisioning")
		}
	}
	return shouldProvision
}

// provisionOS will provision the OS, but on failure it will set op.Result to longrunning.Operation_Error
// and return the same error message
func (p *provisionState) provisionOS(ctx context.Context) error {
	r, err := getRootDev(p.c)
	if err != nil {
		return fmt.Errorf("provisionOS: failed to get root device from DUT, %s", err)
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
	if err := p.installPartitions(ctx, pi); err != nil {
		return fmt.Errorf("provisionOS: failed to provision the OS, %s", err)
	}
	if err := p.postInstall(pi); err != nil {
		p.revertPostInstall(pi)
		return fmt.Errorf("provisionOS: failed to set next kernel, %s", err)
	}

	if board, err := getBoard(p.c); err == nil && strings.HasPrefix(board, "reven") {
		log.Printf("provisionOS: skip clearing TPM owner for board=%s, %s", board, err)
	} else if err := clearTPM(p.c); err != nil {
		return fmt.Errorf("provisionOS: failed to clear TPM owner, %s", err)
	}

	if p.preventReboot {
		log.Printf("provisionOS: reboot prevented by request")
	} else if err := rebootDUT(ctx, p.c); err != nil {
		return fmt.Errorf("provisionOS: failed to reboot DUT, %s", err)
	}
	return nil
}

func (p *provisionState) wipeStateful(ctx context.Context) error {
	if p.preventReboot {
		log.Printf("wipeStateful: wipe skipped as reboot will be prevented by request")
		return nil
	}

	if err := runCmd(p.c, "echo 'fast keepimg' > /mnt/stateful_partition/factory_install_reset"); err != nil {
		return fmt.Errorf("wipeStateful: Failed to to write to factory reset file, %s", err)
	}

	runLabMachineAutoReboot(p.c)
	if err := rebootDUT(ctx, p.c); err != nil {
		return fmt.Errorf("wipeStateful: failed to reboot DUT, %s", err)
	}
	return nil
}

func (p *provisionState) provisionStateful(ctx context.Context) error {
	stopSystemDaemons(p.c)

	if err := p.installStateful(ctx); err != nil {
		p.revertStatefulInstall()
		return fmt.Errorf("provisionStateful: failed to install stateful partition, %s", err)
	}
	if p.preventReboot {
		log.Printf("provisionStateful: reboot prevented by request")
	} else {
		runLabMachineAutoReboot(p.c)
		if err := rebootDUT(ctx, p.c); err != nil {
			return fmt.Errorf("provisionStateful: failed to reboot DUT, %s", err)
		}
	}
	return nil
}

func (p *provisionState) verifyOSProvision(ctx context.Context) error {
	sourceLsbHash, err := runCmdOutput(p.c, "sha256sum /etc/lsb-release")
	if err != nil {
		return fmt.Errorf("verify OS provision: failed to get /etc/lsb-release hash, %s", err)
	}

	shaFields := strings.Fields(sourceLsbHash)
	if len(shaFields) < 1 {
		return fmt.Errorf("verify OS provision: invalid output from sha256sum /etc/lsb-release call, %s", sourceLsbHash)
	}
	sourceLsbHash = shaFields[0]

	if sourceLsbHash != p.targetLsbHash {
		return fmt.Errorf("verify OS provision: /etc/lsb-release hashes differ, found %s, %s was expected", sourceLsbHash, p.targetLsbHash)
	}

	if err := p.verifyKernelState(ctx); err != nil {
		return err
	}
	return nil
}

func (p *provisionState) verifyKernelState(ctx context.Context) error {
	r, err := getRootDev(p.c)
	if err != nil {
		return fmt.Errorf("verifyKernelState: failed to get root device from DUT, %s", err)
	}
	pi := getPartitionInfo(r)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("verifyKernelState: timeout reached, %w", err)
		default:
			if kernelSuccess, err := runCmdOutput(p.c, fmt.Sprintf("cgpt show -S -i %s %s", pi.activeKernelNum, r.disk)); err != nil {
				log.Printf("verifyKernelState: retrying, failed to read active kernel success attribute, %s", err)
			} else {
				kernelSuccess = strings.TrimSpace(kernelSuccess)
				if kernelSuccess == "1" {
					return nil
				}
				// Otherwise retry after a delay until timeout.
				log.Printf("verifyKernelState: waiting for active kernel to be marked successful")
			}
			time.Sleep(2 * time.Second)
		}
	}
}

const (
	fetchUngzipConvertCmd = `curl --keepalive-time 20 -S -s -v -# -C - --retry 3 --retry-delay 60 %[1]s | gzip -d | dd of=%[2]s obs=2M
pipestatus=("${PIPESTATUS[@]}")
if [[ "${pipestatus[0]}" -ne 0 ]]; then
  echo "$(date --rfc-3339=seconds) ERROR: Fetching %[1]s failed." >&2
  exit 1
elif [[ "${pipestatus[1]}" -ne 0 ]]; then
  echo "$(date --rfc-3339=seconds) ERROR: Decompressing %[1]s failed." >&2
  exit 1
elif [[ "${pipestatus[2]}" -ne 0 ]]; then
  echo "$(date --rfc-3339=seconds) ERROR: Writing to %[2]s failed." >&2
  exit 1
fi`
)

// installKernel updates kernelPartition on disk.
func (p *provisionState) installKernel(ctx context.Context, pi partitionInfo) error {
	url, err := p.s.cacheForDut(ctx, path.Join(p.imagePath, "full_dev_part_KERN.bin.gz"), p.dutName)
	if err != nil {
		return fmt.Errorf("install kernel: failed to get GS Cache URL, %s", err)
	}
	return runCmdRetry(ctx, p.c, 5, fmt.Sprintf(fetchUngzipConvertCmd, url, pi.inactiveKernel))
}

// installRoot updates rootPartition on disk.
func (p *provisionState) installRoot(ctx context.Context, pi partitionInfo) error {
	url, err := p.s.cacheForDut(ctx, path.Join(p.imagePath, "full_dev_part_ROOT.bin.gz"), p.dutName)
	if err != nil {
		return fmt.Errorf("install root: failed to get GS Cache URL, %s", err)
	}
	err = runCmdRetry(ctx, p.c, 5, fmt.Sprintf(fetchUngzipConvertCmd, url, pi.inactiveRoot))
	if err != nil {
		return fmt.Errorf("install root: download and copy to DUT failed, %s", err)
	}
	return nil
}

// installMiniOS updates miniOS Partitions on disk.
func (p *provisionState) installMiniOS(ctx context.Context, pi partitionInfo) error {
	url, err := p.s.cacheForDut(ctx, path.Join(p.imagePath, "full_dev_part_MINIOS.bin.gz"), p.dutName)
	if err != nil {
		return fmt.Errorf("install miniOS: failed to get GS Cache URL, %s", err)
	}
	// Write to both A + B miniOS partitions.
	if err := runCmdRetry(ctx, p.c, 5, fmt.Sprintf(fetchUngzipConvertCmd, url, pi.miniOSA)); err != nil {
		return fmt.Errorf("install miniOS: failed to write to A partition, %s", err)
	}
	if err := runCmdRetry(ctx, p.c, 5, fmt.Sprintf(fetchUngzipConvertCmd, url, pi.miniOSB)); err != nil {
		return fmt.Errorf("install miniOS: failed to write to B partition, %s", err)
	}
	return nil
}

const (
	statefulPath           = "/mnt/stateful_partition"
	updateStatefulFilePath = statefulPath + "/.update_available"
)

// installStateful updates the stateful partition on disk (finalized after a reboot).
func (p *provisionState) installStateful(ctx context.Context) error {
	url, err := p.s.cacheForDut(ctx, path.Join(p.imagePath, "stateful.tgz"), p.dutName)
	if err != nil {
		return fmt.Errorf("install stateful: failed to get GS Cache URL, %s", err)
	}
	return runCmdRetry(ctx, p.c, 5, strings.Join([]string{
		fmt.Sprintf("rm -rf %[1]s %[2]s/var_new %[2]s/dev_image_new", updateStatefulFilePath, statefulPath),
		fmt.Sprintf("curl -S -s -v -# -C - --retry 3 --retry-delay 60 %s | tar --ignore-command-error --overwrite --directory=%s -xzf -", url, statefulPath),
		fmt.Sprintf("echo -n clobber > %s", updateStatefulFilePath),
	}, " && "))
}

func (p *provisionState) installPartitions(ctx context.Context, pi partitionInfo) error {
	if err := p.installKernel(ctx, pi); err != nil {
		log.Printf("installPartitions: failed to install kernel.")
		return err
	}
	if err := p.installRoot(ctx, pi); err != nil {
		log.Printf("installPartitions: failed to install rootfs.")
		return err
	}
	return nil
}

func (p *provisionState) postInstall(pi partitionInfo) error {
	tmpMnt, err := runCmdOutput(p.c, "mktemp -d")
	if err != nil {
		return fmt.Errorf("postInstall: failed to create temporary directory, %s", err)
	}
	tmpMnt = strings.TrimSpace(tmpMnt)

	// Mount, get hash, unmount.
	err = runCmd(p.c, fmt.Sprintf("mount -o ro %s %s", pi.inactiveRoot, tmpMnt))
	if err != nil {
		return fmt.Errorf("postInstall: failed to mount inactive root, %s", err)
	}

	targetLsbHash, err := runCmdOutput(p.c, fmt.Sprintf("sha256sum %s/etc/lsb-release", tmpMnt))
	if err != nil {
		return fmt.Errorf("postInstall: getting sha256 hash of /etc/lsb-release within tmp rootfs mount failed, %s", err)
	}
	shaFields := strings.Fields(targetLsbHash)
	if len(shaFields) < 1 {
		return fmt.Errorf("postInstall: invalid output from sha256sum /etc/lsb-release call, %s", targetLsbHash)
	}
	// Since 'sha256sum` includes the filename path we need to split the output.
	p.targetLsbHash = shaFields[0]

	err = runCmd(p.c, fmt.Sprintf("%s/postinst %s", tmpMnt, pi.inactiveRoot))
	if err != nil {
		return fmt.Errorf("postInstall: failed to postinst from inactive root, %s", err)
	}

	if err := runCmd(p.c, fmt.Sprintf("umount %s", tmpMnt)); err != nil {
		return fmt.Errorf("postInstall: failed to umount temporary directory, %s", err)
	}
	if err := runCmd(p.c, fmt.Sprintf("rmdir %s", tmpMnt)); err != nil {
		return fmt.Errorf("postInstall: failed to remove temporary directory, %s", err)
	}
	return nil
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

func (p *provisionState) revertPostInstall(pi partitionInfo) {
	if err := runCmd(p.c, fmt.Sprintf("/postinst %s 2>&1", pi.activeRoot)); err != nil {
		log.Printf("revert post install: failed to revert postinst, %s", err)
	}
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

	// TODO(kimjae): Can parallelize, once outputs can be sorted.
	for _, spec := range specs {
		dlcID := spec.GetId()
		dlcOutputDir := path.Join(dlcCacheDir, dlcID, dlcPackage)
		if err := p.installDLC(ctx, spec, dlcOutputDir, getActiveDLCSlot(r)); err != nil {
			return fmt.Errorf("provision DLCs: failed to install the following DLC %s (%s)", dlcID, err)
		}
	}

	return nil
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
	url, err := p.s.cacheForDut(ctx, dlcURL, p.dutName)
	if err != nil {
		return fmt.Errorf("install DLC: failed to get GS Cache server, %s", err)
	}

	dlcOutputSlotDir := path.Join(dlcOutputDir, string(slot))
	dlcOutputImage := path.Join(dlcOutputSlotDir, dlcImage)
	dlcCmd := fmt.Sprintf(`mkdir -p %[1]s && curl -S -s -v -# -C - --retry 3 --retry-delay 60 --output %[2]s %[3]s`,
		dlcOutputSlotDir, dlcOutputImage, url)
	if err := runCmd(p.c, dlcCmd); err != nil {
		return fmt.Errorf("provision DLC: failed to provision DLC %s, %s", dlcID, err)
	}
	return nil
}

func (p *provisionState) provisionMiniOS(ctx context.Context) error {
	log.Printf("provision MiniOS: started")
	defer log.Printf("provision MiniOS: finished")

	r, err := getRootDev(p.c)
	if err != nil {
		return fmt.Errorf("provision MiniOS: failed to get root device from DUT, %s", err)
	}

	// Check if the device has miniOS partitions.
	for _, part := range []string{"9", "10"} {
		out, err := runCmdOutput(p.c, fmt.Sprintf("cgpt show -t %s -i %s", r.disk, part))
		if err != nil {
			return fmt.Errorf("provision MiniOS: failed to get partition type, %s", err)
		}
		out = strings.TrimSpace(out)
		// Check against miniOS GUID type.
		if out != "09845860-705F-4BB5-B16C-8A8A099CAF52" {
			log.Printf("Skipping miniOS provision as device doesn't support miniOS")
			return nil
		}
	}

	log.Printf("provision MiniOS: continuing to provision miniOS partitions")
	if err := p.installMiniOS(ctx, getPartitionInfo(r)); err != nil {
		return fmt.Errorf("provision MiniOS: failed to install miniOS, %s", err)
	}
	return nil
}

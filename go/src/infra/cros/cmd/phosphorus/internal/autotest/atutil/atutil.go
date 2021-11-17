// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/*
Package atutil provides a higher level Autotest interface than the autotest package.
*/
package atutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/pkg/errors"
	"go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/luci/common/logging"

	"infra/cros/cmd/phosphorus/internal/autotest"
	"infra/cros/cmd/phosphorus/internal/osutil"
	"infra/cros/internal/cmd"
	"infra/cros/internal/docker"
	"infra/cros/internal/osutils"
)

const (
	keyvalFile                = "keyval"
	autoservPidFile           = ".autoserv_execute"
	tkoPidFile                = ".parser_execute"
	sspDeployShadowConfigFile = "ssp_deploy_shadow_config.json"
)

// RunAutoserv runs an autoserv task.
//
// This function always returns a non-nil Result, but some fields may
// not be meaningful.  For example, Result.Exit will be 0 even if
// autoserv could not be run.  In this case, Result.Started will be
// false and an error will also returned.
//
// Output is written to the Writer.
//
// If containerImageInfo is set, the autoserv task will run inside a
// Docker container.
//
// The location of autoserv in the container is specified by m (the same as if
// autoserv runs on the host). Similarly, results are written to the results dir
// specified by j.
//
// Result.TestsFailed may not be set, depending on AutoservJob.  An
// error is not returned for test failures.
func RunAutoserv(
	ctx context.Context,
	m *MainJob,
	j AutoservJob,
	w io.Writer,
	dockerCmdRunner cmd.CommandRunner,
	containerImageInfo *api.ContainerImageInfo,
) (r *Result, err error) {
	if err2 := prepareHostInfo(m.ResultsDir, j); err2 != nil {
		return nil, err2
	}
	defer func() {
		if err2 := retrieveHostInfo(m.ResultsDir, j); err2 != nil {
			log.Printf("Failed to retrieve host info for autoserv test: %s", err2)
			if err == nil {
				err = err2
			}
		}
	}()
	a := j.AutoservArgs()
	if j, ok := j.(keyvalsJob); ok {
		if err := writeKeyvals(a.ResultsDir, j.JobKeyvals()); err != nil {
			return &Result{}, err
		}
	}
	switch {
	case isTest(a):
		return runTest(ctx, m.AutotestConfig, a, w, dockerCmdRunner, containerImageInfo)
	default:
		return runTask(ctx, m.AutotestConfig, a, w, dockerCmdRunner, containerImageInfo)
	}
}

// TKOParse runs tko/parse on the results directory.  The level is
// used by tko/parse to determine how many parts of the results dir
// absolute path to take for the unique job tag.
//
// Parse output is written to the Writer.
//
// This function returns the number of tests failed and an error if
// any.
func TKOParse(c autotest.Config, resultsDir string, w io.Writer) (failed int, err error) {
	cmd := autotest.ParseCommand(c, resultsDir)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return 0, errors.Wrap(err, "run tko/parse")
	}
	p := filepath.Join(resultsDir, tkoPidFile)
	n, err := readTestsFailed(p)
	if err != nil {
		return 0, errors.Wrap(err, "parse tests failed")
	}
	return n, nil
}

type SSPDeployConfig struct {
	Source      string `json:"source,omitempty"`
	Target      string `json:"target,omitempty"`
	Append      bool   `json:"append,omitempty"`
	Mount       bool   `json:"mount,omitempty"`
	Readonly    bool   `json:"readonly,omitempty"`
	ForceCreate bool   `json:"force_create,omitempty"`
}

// ParseSSPDeployShadowConfig parses a JSON file containing ssp deploy shadow
// config into Mounts for use in a ContainerConfig.
func ParseSSPDeployShadowConfig(
	ctx context.Context,
	autotestConfig autotest.Config,
	configFile string,
) ([]mount.Mount, error) {
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading SSP deploy shadow config file")
	}

	var deployConfigs []SSPDeployConfig
	err = json.Unmarshal(bytes, &deployConfigs)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing SSP deploy shadow config file")
	}

	var mounts []mount.Mount

	for _, config := range deployConfigs {
		if !path.IsAbs(config.Target) {
			return nil, fmt.Errorf("target path must be absolute (%s)", config.Target)
		}

		if !filepath.IsAbs(config.Source) {
			if strings.HasPrefix(config.Source, "~") {
				return nil, fmt.Errorf("source path may not have '~' (%s)", config.Source)
			}

			absPath := filepath.Join(autotestConfig.AutotestDir, config.Source)
			logging.Infof(
				ctx,
				"joining autotest dir (%q) to relative path %q, to form absolute path %q",
				autotestConfig.AutotestDir, config.Source, absPath,
			)

			config.Source = absPath
		}

		if _, err := os.Stat(config.Source); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if config.ForceCreate {
					logging.Infof(ctx, "source %s does not exist, creating because forceCreate is set", config.Source)

					if err := os.MkdirAll(config.Source, os.ModePerm); err != nil {
						return nil, fmt.Errorf("failed creating source dir (%s): %w", config.Source, err)
					}
				} else {
					return nil, fmt.Errorf("source %s does not exist, and forceCreate is not set", config.Source)
				}
			} else {
				return nil, fmt.Errorf("failed calling stat on file (%s): %w", config.Source, err)
			}
		}

		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Target:   config.Target,
			Source:   config.Source,
			ReadOnly: config.Readonly,
		})
	}

	return mounts, nil
}

// getHostInfoStoreMounts returns Mount objects to mount each host info store
// path into a Docker container.
func getHostInfoStoreMounts(ctx context.Context, autotestArgs *autotest.AutoservArgs) []mount.Mount {
	var mounts []mount.Mount

	rootDir := autotestArgs.ResultsDir

	for _, dutName := range append(autotestArgs.Hosts, autotestArgs.PeerDuts...) {
		hostInfoFilePath := HostInfoFilePath(rootDir, dutName)

		fileInfo, err := os.Stat(hostInfoFilePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				logging.Warningf(ctx, "host info file path %q does not exist: %s", hostInfoFilePath, err)
			} else {
				logging.Warningf(ctx, "got error calling Stat on host info file path %q: %s", hostInfoFilePath, err)
			}
		} else {
			logging.Infof(ctx, "result of stat on host info file path: %+v", fileInfo)
		}

		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Target:   filepath.ToSlash(hostInfoFilePath),
			Source:   hostInfoFilePath,
			ReadOnly: false,
		})
	}

	return mounts
}

// runTask runs an autoserv task.
//
// Result.TestsFailed is always zero.
func runTask(ctx context.Context,
	c autotest.Config,
	a *autotest.AutoservArgs,
	w io.Writer,
	dockerCmdRunner cmd.CommandRunner,
	containerImageInfo *api.ContainerImageInfo,
) (*Result, error) {
	r := &Result{}

	if containerImageInfo.GetName() != "" {
		// SSP args should not be specified when autoserv is run in a Docker
		// container.
		argsNoSSP := *a
		argsNoSSP.SSPBaseImageName = ""
		argsNoSSP.RequireSSP = false
		argsNoSSP.Verbose = true
		cmd := autotest.AutoservCommand(c, &argsNoSSP)

		imageName := fmt.Sprintf(
			"%s/%s/%s@%s",
			containerImageInfo.GetRepository().GetHostname(),
			containerImageInfo.GetRepository().GetProject(),
			containerImageInfo.GetName(),
			containerImageInfo.GetDigest(),
		)

		containerConfig := &container.Config{
			Image: imageName,
			Cmd:   cmd.Args,
			User:  "chromeos-test",
		}

		// The container runs as a different user from the host. Just make the
		// results dir writable by all, to allow the container to write.
		if err := osutils.RecursiveChmod(a.ResultsDir, os.ModePerm); err != nil {
			return r, fmt.Errorf("failed calling chmod on results dir (%q): %w", a.ResultsDir, err)
		}

		mounts := []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   a.ResultsDir,
				Target:   a.ResultsDir,
				ReadOnly: false,
			},
		}

		sspDeployShadowConfigMounts, err := ParseSSPDeployShadowConfig(
			ctx, c, filepath.Join(c.AutotestDir, sspDeployShadowConfigFile),
		)
		if err != nil {
			return r, errors.Wrap(err, "failed parsing SSP deploy shadow config")
		}

		mounts = append(mounts, sspDeployShadowConfigMounts...)
		// TODO(b/201431966): host info store files are inside the results dir,
		// which is already mounted to the container. Remove this redundant
		// mount if possible.
		mounts = append(mounts, getHostInfoStoreMounts(ctx, a)...)

		// The results dir on the host is bound to the same path in the
		// container. Thus, autoserv will write to a.ResultsDir in the
		// container, which is bound to a.ResultsDir in the host, so results
		// are available in the expected location on the host.
		hostConfig := &container.HostConfig{
			NetworkMode: "host",
			Mounts:      mounts,
		}

		logging.Infof(ctx, "creating Docker container with command %q", containerConfig.Cmd)

		err = docker.RunContainer(ctx, dockerCmdRunner, containerConfig, hostConfig, containerImageInfo)
		if err != nil {
			var exErr *exec.ExitError
			if errors.As(err, &exErr) {
				r.Exit = exErr.ProcessState.ExitCode()
			} else {
				return r, err
			}
		}
	} else {
		cmd := autotest.AutoservCommand(c, a)
		cmd.Stdout = w
		cmd.Stderr = w

		var err error
		r.RunResult, err = osutil.RunWithAbort(ctx, cmd)
		if err != nil {
			return r, err
		}
		if es, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			r.Exit = es.ExitStatus()
		} else {
			return r, errors.New("RunAutoserv: failed to get exit status: unknown process state")
		}
	}

	logging.Infof(ctx, "RunAutoserv: exited %d", r.Exit)
	return r, nil
}

// runTest runs an autoserv test.
//
// Unlike runTask, this function performs some things only needed for
// tests, like parsing the number of test failed and writing a job
// finished timestamp.
func runTest(
	ctx context.Context,
	c autotest.Config,
	a *autotest.AutoservArgs,
	w io.Writer,
	dockerCmdRunner cmd.CommandRunner,
	containerImageInfo *api.ContainerImageInfo,
) (*Result, error) {
	r, err := runTask(ctx, c, a, w, dockerCmdRunner, containerImageInfo)
	if !r.Started || r.Exit != 0 {
		// autoserv did not exit cleanly so artifacts may not be present.
		return r, err
	}
	p := filepath.Join(a.ResultsDir, autoservPidFile)
	if i, err2 := readTestsFailed(p); err2 != nil {
		if err == nil {
			err = err2
		}
	} else {
		r.TestsFailed = i
	}
	if err2 := appendJobFinished(a.ResultsDir); err == nil {
		err = err2
	}
	return r, err
}

// isTest returns true if the given AutoservArgs represents a test
// job.
func isTest(a *autotest.AutoservArgs) bool {
	switch {
	case a.Verify, a.Cleanup, a.Reset, a.Repair, a.Provision:
		return false
	default:
		return true
	}
}

// readTestsFailed reads the number of tests failed from the given
// pid file.
func readTestsFailed(pidFile string) (int, error) {
	b, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	s := string(b)
	lines := strings.Split(s, "\n")
	if len(lines) < 3 {
		return 0, fmt.Errorf("Not enough lines in pidfile %s", pidFile)
	}
	i, err := strconv.Atoi(lines[2])
	if err != nil {
		return 0, err
	}
	return i, nil
}

func writeKeyvals(resultsDir string, m map[string]string) error {
	p := keyvalPath(resultsDir)
	if err := os.MkdirAll(filepath.Dir(p), 0777); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	err = autotest.WriteKeyvals(f, m)
	if err2 := f.Close(); err == nil {
		err = err2
	}
	return err
}

// appendJobFinished appends a job_finished value to the test’s keyval file.
func appendJobFinished(resultsDir string) error {
	p := keyvalPath(resultsDir)
	msg := fmt.Sprintf("job_finished=%d\n", time.Now().Unix())
	return appendToFile(p, msg)
}

func keyvalPath(resultsDir string) string {
	return filepath.Join(resultsDir, keyvalFile)
}

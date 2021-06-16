// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ssh

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/ssh"

	"infra/cros/recovery/tlw"
	"infra/libs/sshpool"
)

const defaultSSHUser = "root"

// getSSHConfig provides default config for SSH.
func SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            defaultSSHUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         60 * time.Second,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(SSHSigner)},
	}
}

// Run executes command on the target address by SSH.
func Run(pool *sshpool.Pool, addr string, cmd string) (result *tlw.RunResult) {
	result = &tlw.RunResult{
		Command:  cmd,
		ExitCode: -1,
	}
	if pool == nil {
		result.Stderr = "run SSH: pool is not initialized"
		return
	} else if addr == "" {
		result.Stderr = "run SSH: addr is empty"
		return
	} else if cmd == "" {
		result.Stderr = fmt.Sprintf("run SSH %q: cmd is empty", addr)
		return
	}
	sc, err := pool.Get(addr)
	if err != nil {
		result.Stderr = fmt.Sprintf("run SSH %q: fail to get client from pool; %s", addr, err)
		return
	}
	defer pool.Put(addr, sc)
	result = internalRunSSH(cmd, sc)
	log.Printf("run SSH %q: Cmd: %q; ExitCode: %d; Stdout: %q;  Stderr: %q", addr, result.Command, result.ExitCode, result.Stdout, result.Stderr)
	return
}

func internalRunSSH(cmd string, client *ssh.Client) (result *tlw.RunResult) {
	result = &tlw.RunResult{
		Command:  cmd,
		ExitCode: -1,
	}
	session, err := client.NewSession()
	if err != nil {
		result.Stderr = fmt.Sprintf("internal run SSH: %s", err)
		return
	}
	defer session.Close()
	var stdOut, stdErr bytes.Buffer
	session.Stdout = &stdOut
	session.Stderr = &stdErr

	err = session.Run(cmd)

	result.Stdout = stdOut.String()
	result.Stderr = stdErr.String()
	if err == nil {
		result.ExitCode = 0
	} else if exitErr, ok := err.(*ssh.ExitError); ok {
		result.ExitCode = exitErr.ExitStatus()
	}
	return
}

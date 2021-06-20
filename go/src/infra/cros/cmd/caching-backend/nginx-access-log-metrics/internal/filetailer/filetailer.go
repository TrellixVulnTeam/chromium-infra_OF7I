// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package filetailer

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"time"
)

// Tailer scans a file for newly appended lines and returns them.
type Tailer struct {
	cmd        *exec.Cmd
	terminated chan struct{}
	scanner    *bufio.Scanner
}

// New creates a new Tailer object.
// It's the caller's responsibility to ensure the filename is correct. We don't
// check the file existence here because we have to tolerate log file rotation.
func New(filename string) (*Tailer, error) {
	// We tail a file by its name instead of descriptor in order to handle
	// the case of file rotation. Thus we use `tail -F`.
	cmd := exec.Command("tail", "-n", "0", "-F", filename)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create new tailer: %s", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("create new tailer: %s", err)
	}
	log.Printf("filetailer: ('tail' pid: %d) tailing %q", cmd.Process.Pid, filename)

	terminated := make(chan struct{})
	go func() {
		cmd.Wait()
		close(terminated)
	}()

	return &Tailer{cmd: cmd, terminated: terminated, scanner: bufio.NewScanner(stdout)}, nil
}

// Scan scans the file for new lines.
// See bufio.Scanner.Scan() for more details.
func (t *Tailer) Scan() bool {
	return t.scanner.Scan()
}

// Text returns the most recent line by a call of Scan from the file.
// See bufio.Scanner.Text() for more details.
func (t *Tailer) Text() string {
	return t.scanner.Text()
}

// Close closes the Tailer object and release all resources.
func (t *Tailer) Close() {
	// Clean up the 'tail' process using the method of SIGTERM, timeout,
	// SIGKILL.
	if err := t.closeTailing(); err != nil {
		log.Printf("filetailer: closing 'tail' with SIGTERM: %s", err)
		return
	}
	select {
	case <-t.terminated:
		log.Printf("filetailer: 'tail' was exited")
		return
	case <-time.After(2 * time.Second):
		if err := t.cmd.Process.Kill(); err != nil {
			log.Printf("filetailer: closing 'tail' with SIGKILL: %s", err)
			return
		}
		log.Printf("filetailer: 'tail' was killed")
	}
	<-t.terminated
	if err := t.scanner.Err(); err != nil {
		log.Printf("filetailer: scanner exited with: %s", err)
	}
}

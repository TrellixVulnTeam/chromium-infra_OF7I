// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package atutil

import (
	"infra/cros/cmd/phosphorus/internal/autotest"
	"infra/cros/cmd/phosphorus/internal/osutil"
)

// MainJob describes the overall job, which dictates certain job
// global settings for running autoserv.
type MainJob struct {
	AutotestConfig autotest.Config
	ResultsDir     string
}

// AutoservJob describes the interface a job object needs to be passed
// to RunAutoserv.
type AutoservJob interface {
	AutoservArgs() *autotest.AutoservArgs
}

type keyvalsJob interface {
	JobKeyvals() map[string]string
}

// AdminTaskType is an enum used in AdminTask to determine what type
// of admin task to run.
type AdminTaskType int

const (
	// NoTask can be used as a null AdminTaskType value.
	NoTask AdminTaskType = iota
	// Verify represents `autoserv -v`.
	Verify
	// Cleanup represents `autoserv --cleanup`.
	Cleanup
	// Reset represents `autoserv --reset`.
	Reset
	// Repair represents `autoserv -R`.
	Repair
)

//go:generate stringer -type=AdminTaskType

const hostInfoSubDir = "host_info_store"

var _ AutoservJob = &AdminTask{}

// AdminTask represents an admin task to run.  AdminTask implements AutoservJob.
type AdminTask struct {
	Type       AdminTaskType
	Host       string
	ResultsDir string
}

// AutoservArgs represents the CLI args for `autoserv`.
func (t *AdminTask) AutoservArgs() *autotest.AutoservArgs {
	a := &autotest.AutoservArgs{
		HostInfoSubDir: hostInfoSubDir,
		Hosts:          []string{t.Host},
		Lab:            true,
		ResultsDir:     t.ResultsDir,
		WritePidfile:   true,
	}
	switch t.Type {
	case Verify:
		a.Verify = true
	case Cleanup:
		a.Cleanup = true
	case Reset:
		a.Reset = true
	case Repair:
		a.Repair = true
	}
	return a
}

var _ AutoservJob = &Provision{}

// Provision represents a provision task to run.  Provision implements
// AutoservJob.
type Provision struct {
	Host       string
	Labels     []string
	ResultsDir string
}

// AutoservArgs represents the CLI args for `autoserv`.
func (p *Provision) AutoservArgs() *autotest.AutoservArgs {
	return &autotest.AutoservArgs{
		HostInfoSubDir: hostInfoSubDir,
		Hosts:          []string{p.Host},
		JobLabels:      p.Labels,
		Lab:            true,
		Provision:      true,
		ResultsDir:     p.ResultsDir,
		WritePidfile:   true,
	}
}

var _ AutoservJob = &Test{}
var _ keyvalsJob = &Test{}

// Test represents a test to run.  Test implements AutoservJob.
type Test struct {
	Args             string
	ClientTest       bool
	ControlFile      string
	ControlName      string
	ExecutionTag     string
	Hosts            []string
	Keyvals          map[string]string
	Name             string
	Owner            string
	ParentJobID      int
	PeerDuts         []string
	RequireSSP       bool
	ResultsDir       string
	SSPBaseImageName string
	TestSourceBuild  string
}

// AutoservArgs represents the CLI args for `autoserv`.
func (t *Test) AutoservArgs() *autotest.AutoservArgs {
	return &autotest.AutoservArgs{
		Args:             t.Args,
		ClientTest:       t.ClientTest,
		ControlFile:      t.ControlFile,
		ControlName:      t.ControlName,
		ExecutionTag:     t.ExecutionTag,
		Hosts:            t.Hosts,
		HostInfoSubDir:   hostInfoSubDir,
		JobName:          t.Name,
		JobOwner:         t.Owner,
		Lab:              true,
		NoTee:            true,
		ParentJobID:      t.ParentJobID,
		PeerDuts:         t.PeerDuts,
		RequireSSP:       t.RequireSSP,
		ResultsDir:       t.ResultsDir,
		SSPBaseImageName: t.SSPBaseImageName,
		TestSourceBuild:  t.TestSourceBuild,
		VerifyJobRepoURL: true,
		WritePidfile:     true,
	}
}

// JobKeyvals returns the autotest keyvals.
func (t *Test) JobKeyvals() map[string]string {
	return t.Keyvals
}

// Result contains information about RunAutoserv results.
type Result struct {
	osutil.RunResult
	// Exit is the exit status for the autoserv command, if
	// autoserv was run.
	Exit        int
	TestsFailed int
}

// Success returns true if autoserv exited with 0 and no tests failed.
func (r *Result) Success() bool {
	return r.Exit == 0 && r.TestsFailed == 0
}

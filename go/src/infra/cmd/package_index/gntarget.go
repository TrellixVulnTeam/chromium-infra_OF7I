// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"sync"

	kpb "infra/cmd/package_index/kythe/proto"
)

// Error return by a target processor when a given target cannot be processed.
var errNotSupported = errors.New("not supported")

// Used to process GN targets.
type processor func(context.Context, string, string, string, string,
	*FileHashMap, *gnTarget) (GnTargetInterface, error)

// GnTargetInterface the return value of the GN target processors that contains
// methods for processing a target's data files and compilation unit.
type GnTargetInterface interface {
	getUnit() (*kpb.CompilationUnit, error)
	getFiles() ([]string, error)
}

// Maps all GN target names to their respective JSON information.
// Used for parsing a GN targets file.
var gnTargetsMap map[string]gnTargetInfo

// gnTargetInfo stores the JSON information for a given target.
type gnTargetInfo struct {
	Args    []string `json:"args"`
	Sources []string `json:"sources"`
	Script  string   `json:"script"`
}

// gnTarget stores a singular target's name and JSON information parsed from a GN targets file.
type gnTarget struct {
	targetName string
	targetInfo gnTargetInfo
}

// GnTargets contains info for GN target processing.
type GnTargets struct {
	once       sync.Once
	filePath   string
	processors []processor
	targetChan chan *gnTarget
	targetsLen int

	// WaitGroup for closing targetDataOut.
	dataWg sync.WaitGroup

	// WaitGroup for closing targetUnitOut.
	unitWg sync.WaitGroup

	// WaitGroup for deferring processing of compilation units until after data files
	// have been processed into kzip entries.
	kzipDataWg sync.WaitGroup
}

// NewGnTargets returns a GnTargets struct based on JSON file gnTargetsPath.
func NewGnTargets(gnTargetsPath string) *GnTargets {
	gn := &GnTargets{
		filePath:   gnTargetsPath,
		processors: []processor{protoTargetProcessor, mojomTargetProcessor},
	}
	return gn
}

// populateChannel parses the GN targets JSON file given in the command line arguments and
// sends the formatted data contained in a gnTarget struct to the targets channel in gnTargets.
func (gnTargets *GnTargets) populateChannel() {
	if gnTargetsMap == nil {
		// Unmarshal JSON.
		dat, err := ioutil.ReadFile(gnTargets.filePath)
		if err != nil {
			panic(err)
		}
		json.Unmarshal(dat, &gnTargetsMap)
	}
	gnTargets.targetsLen = len(gnTargetsMap)
	gnTargets.targetChan = make(chan *gnTarget, gnTargets.targetsLen)
	for targetName, targetInfo := range gnTargetsMap {
		gnTargets.targetChan <- &gnTarget{targetName, targetInfo}
	}
	close(gnTargets.targetChan)
}

// ProcessGnTargets takes a target from either targetFilesChan or targetUnitsChan
// and runs through several language-specific processors. It sends the results of
// the processors to channels for unit files and data files.
//
// If files is true, then process and send target data files. Otherwise, process
// and send the compilation unit.
func (gnTargets *GnTargets) ProcessGnTargets(ctx context.Context, rootPath, outDir, corpus, buildConfig string,
	hashMaps *FileHashMap, targetDataOut chan<- string, targetUnitOut chan<- *kpb.CompilationUnit) error {
	// Parse GN targets once.
	gnTargets.once.Do(gnTargets.populateChannel)

	// Channel for deferred compilation unit processing.
	gnInterfaceChan := make(chan GnTargetInterface, gnTargets.targetsLen)
	for target := range gnTargets.targetChan {
		// Iterate through processors to find appropriate language.
		for _, proc := range gnTargets.processors {
			gnInterface, err := proc(ctx, rootPath, outDir, corpus, buildConfig, hashMaps, target)
			if err != nil {
				continue
			}

			// Get and process data files.
			files, err := gnInterface.getFiles()
			if err != nil {
				return err
			}
			if files == nil {
				continue
			}
			for _, f := range files {
				targetDataOut <- f
			}

			// Send gnInterface to channel for deferred unit processing.
			gnInterfaceChan <- gnInterface
			break
		}
	}
	close(gnInterfaceChan)
	gnTargets.dataWg.Done()

	// Wait for data files to be processed for writing to kzip.
	gnTargets.kzipDataWg.Wait()

	// Process compilation units.
	for gnInterface := range gnInterfaceChan {
		// Get and process the compilation unit.
		unit, err := gnInterface.getUnit()
		if err != nil {
			return err
		}
		if unit == nil {
			continue
		}
		targetUnitOut <- unit
	}
	gnTargets.unitWg.Done()
	return nil
}

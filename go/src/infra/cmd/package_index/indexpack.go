package main

import (
	"context"
)

// indexPack contains the information necessary to assemble the kzip.
type indexPack struct {
	// Path to which the index pack will be written.
	outputFile string
	// Path to the root of the checkout (i.e. the path containing src/). outDir is relative to this.
	rootPath string
	// The output directory from which compilation was run.
	outDir string
	// Path to the compilation database.
	compDbPath string
	// Path to a json file contains gn target information, as produced by 'gn desc --format=json'.
	// See 'gn help desc' for more info.
	gnTargetsPath string
	// Path to java kzips produced by javac_extractor. Units are in json format.
	existingJavaKzipsPath string
	// The corpus to use for the generated Kythe VNames, e.g. 'chromium'.
	// A VName identifies a node in the Kythe index.
	// For more details, see: https://kythe.io/docs/kythe-storage.html
	corpus string
	// The build config to specify in the unit file, e.g. 'android' or 'windows' (optional)
	buildConfig string
	verbose     bool

	// Mapping to and from a filename and its content hash.
	hashMaps *FileHashMap

	// Used for logging.
	ctx context.Context
}

// newIndexPack initializes a new indexPack struct.
func newIndexPack(ctx context.Context, outputFile, rootPath, outDir, compDbPath,
	gnTargetsPath, existingJavaKzipsPath, corpus, buildConfig string, verbose bool) *indexPack {
	// Initialize indexPack.
	ip := &indexPack{
		outputFile:            outputFile,
		rootPath:              rootPath,
		outDir:                outDir,
		compDbPath:            compDbPath,
		gnTargetsPath:         gnTargetsPath,
		existingJavaKzipsPath: existingJavaKzipsPath,
		corpus:                corpus,
		buildConfig:           buildConfig,
		verbose:               verbose,
		ctx:                   ctx,
	}
	ip.hashMaps = NewFileHashMap()
	return ip
}

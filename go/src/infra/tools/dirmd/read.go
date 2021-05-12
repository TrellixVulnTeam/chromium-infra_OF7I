// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"bufio"
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/encoding/prototext"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"

	dirmdpb "infra/tools/dirmd/proto"
)

// Filename is the standard name of the metadata file.
const Filename = "DIR_METADATA"

var gitBinary string

func init() {
	gitBinary = "git"
	if runtime.GOOS == "windows" {
		gitBinary = "git.exe"
	}
}

// ReadMetadata reads metadata from a single directory.
// See also ReadMapping.
//
// Returns (nil, nil) if the metadata is not defined.
func ReadMetadata(dir string) (*dirmdpb.Metadata, error) {
	fullPath := filepath.Join(dir, Filename)
	contents, err := ioutil.ReadFile(fullPath)
	switch {
	case os.IsNotExist(err):
		// Try the legacy file.
		md, _, err := ReadOwners(dir)
		return md, err

	case err != nil:
		return nil, err
	}

	var ret dirmdpb.Metadata
	if err := prototext.Unmarshal(contents, &ret); err != nil {
		return nil, errors.Annotate(err, "failed to parse %q", fullPath).Err()
	}
	return &ret, nil
}

// ReadMapping reads all metadata from files in git in the given directories.
//
// Each directory must reside in a git checkout. ReadMapping reads
// visible-to-git metadata files under the directory.
// In particular, files outside of the repo are not read, as well as files
// matched by .gitignore files. The set of considered files is equivalent to
// `git ls-files <dir>`, see its documentation.
// Note that a directory does not have to be the root of the git repo, and
// multiple directories in the same repo are allowed.
//
// One of the repos must be the root repo, while other repos must be its
// sub-repos. In other words, all git repos referred to by the directories must
// be subdirectories of one of the repos.
// The root dir of the root repo becomes the metadata root.
func ReadMapping(ctx context.Context, form dirmdpb.MappingForm, dirs ...string) (*Mapping, error) {
	if len(dirs) == 0 {
		return nil, nil
	}

	// Ensure all dir paths are absolute, for simplicity down the road.
	for i, d := range dirs {
		var err error
		if dirs[i], err = filepath.Abs(d); err != nil {
			return nil, errors.Annotate(err, "%q", d).Err()
		}
	}

	// Group all dirs by the repo root.
	byRepoRoot, err := dirsByRepoRoot(ctx, dirs)
	if err != nil {
		return nil, err
	}

	r := &mappingReader{
		Mapping:         *NewMapping(0),
		semReadMetadata: semaphore.NewWeighted(int64(runtime.NumCPU())),
	}
	// Find the metadata root, i.e. the root dir of the root repo.
	if r.Root, err = findMetadataRoot(byRepoRoot); err != nil {
		return nil, err
	}

	// Read the metadata files concurrently.
	eg, ctx := errgroup.WithContext(ctx)
	defer eg.Wait()
	for repoRoot, repoDirs := range byRepoRoot {
		repoRoot := repoRoot
		for _, dir := range removeRedundantDirs(repoDirs...) {
			dir := dir
			eg.Go(func() error {
				err := r.ReadGitFiles(ctx, repoRoot, dir, form == dirmdpb.MappingForm_FULL)
				return errors.Annotate(err, "failed to process %q", dir).Err()
			})
		}
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// Finally, bring the mapping to the desired form.
	switch form {
	case dirmdpb.MappingForm_REDUCED:
		r.Mapping.Reduce()
	case dirmdpb.MappingForm_COMPUTED, dirmdpb.MappingForm_FULL:
		r.Mapping.ComputeAll()
	}

	return &r.Mapping, nil
}

// findMetadataRoot returns the root directory of the root repo.
func findMetadataRoot(byRepoRoot map[string][]string) (string, error) {
	rootSlice := make([]string, 0, len(byRepoRoot))
	for rr := range byRepoRoot {
		rootSlice = append(rootSlice, rr)
	}
	sort.Strings(rootSlice)

	// The shortest must be the root.
	// Verify that all others have it as the prefix.
	rootNormalized := normalizeDir(rootSlice[0])

	for _, rr := range rootSlice[1:] {
		if !strings.HasPrefix(normalizeDir(rr), rootNormalized) {
			return "", errors.Reason("failed to determine the metadata root: expected %q to be a subdir of %q", rr, rootSlice[0]).Err()
		}
	}
	return rootSlice[0], nil
}

// dirsByRepoRoot groups directories by the root of the git repo they reside in.
func dirsByRepoRoot(ctx context.Context, dirs []string) (map[string][]string, error) {
	var mu sync.Mutex
	// Most likely, dirs are in different repos, so allocate len(dirs) entries.
	ret := make(map[string][]string, len(dirs))
	eg, ctx := errgroup.WithContext(ctx)
	for _, dir := range dirs {
		dir := dir
		eg.Go(func() error {
			cmd := exec.CommandContext(ctx, gitBinary, "-C", dir, "rev-parse", "--show-toplevel")
			stdout, err := cmd.Output()
			if err != nil {
				return err
			}
			repoRoot := string(bytes.TrimSpace(stdout))

			mu.Lock()
			ret[repoRoot] = append(ret[repoRoot], dir)
			mu.Unlock()
			return nil
		})
	}
	return ret, eg.Wait()
}

// removeRedundantDirs removes directories already included in other directories.
// Mutates dirs in place.
func removeRedundantDirs(dirs ...string) []string {
	// Sort directories from shorest-to-longest.
	// Note that this sorts by byte-length (not rune-length) and there is a small
	// chance that a directory path contains a 2+ byte rune, but this is OK
	// because such a rune is very unlikely to be equivalent to another shorter
	// rune.
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) < len(dirs[j])
	})

	ret := dirs[:0] // https://github.com/golang/go/wiki/SliceTricks#filter-in-place
	acceptedNormalized := make([]string, 0, len(dirs))
	for _, d := range dirs {
		dirNormalized := normalizeDir(d)
		redundant := false
		for _, shorter := range acceptedNormalized {
			if strings.HasPrefix(dirNormalized, shorter) {
				redundant = true
				break
			}
		}
		if !redundant {
			acceptedNormalized = append(acceptedNormalized, dirNormalized)
			ret = append(ret, d)
		}
	}
	return ret
}

// ReadComputed returns computed metadata for the target directories.
// All returned metadata includes inherited metadata, starting from root.
// The returned mapping includes entries only for targets.
//
// Reads metadata files only for directories that are on the node path from
// root to each of the targets. In particular, does not necessarily read
// the entire tree under root, so this is more efficient than ReadMapping.
func ReadComputed(root string, targets ...string) (*Mapping, error) {
	r := &mappingReader{Root: root}
	for _, target := range targets {
		if err := r.readUpMissing(target); err != nil {
			return nil, errors.Annotate(err, "failed to read metadata for %q", target).Err()
		}
	}

	r.ComputeAll()

	// Filter by targets.
	ret := NewMapping(len(targets))
	for _, target := range targets {
		key, err := r.DirKey(target)
		if err != nil {
			panic(err) // Impossible: we have just used these paths above.
		}
		ret.Dirs[key] = r.Mapping.Dirs[key]
	}
	return ret, nil
}

// mappingReader reads Mapping from the file system.
type mappingReader struct {
	// Root is an absolute path to the metadata root directory.
	// In the case of multiple repos, it is the root dir of the root repo.
	Root string

	// Mapping is the result of reading.
	Mapping

	mu              sync.Mutex
	semReadMetadata *semaphore.Weighted
}

// ReadGitFiles reads metadata files-in-git under dir and adds them to r.Mapping.
//
// It uses "git-ls-files <dir>" to discover the files, so for example it ignores
// files outside of the repo. See more in `git ls-files -help`.
func (r *mappingReader) ReadGitFiles(ctx context.Context, absRepoRoot, absTreeRoot string, preserveFileStructure bool) error {
	// First, determine the key prefix.
	keyPrefixNative, err := filepath.Rel(r.Root, absRepoRoot)
	if err != nil {
		return err
	}
	keyPrefix := filepath.ToSlash(keyPrefixNative)

	eg, ctx := errgroup.WithContext(ctx)
	defer eg.Wait()

	// Concurrently start `git ls-files`, read its output and read the discovered
	// metadata files.

	eg.Go(func() error {
		cmd := exec.CommandContext(ctx, gitBinary, "-C", absRepoRoot, "ls-files", "--full-name", absTreeRoot)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return errors.Annotate(err, "failed to start `git ls-files`").Err()
		}
		defer cmd.Wait() // do not exit the func until the subprocess exits.

		seen := stringset.New(0)
		scan := bufio.NewScanner(stdout)
		for scan.Scan() {
			relFileName := scan.Text()      // slash-separated, relative to repo root
			relDir := path.Dir(relFileName) // slash-separated, relative to repo root
			key := path.Join(keyPrefix, relDir)

			if preserveFileStructure {
				// Ensure the existence of the directory is recorded even if there is no metadata.
				r.mu.Lock()
				if _, ok := r.Dirs[key]; !ok {
					r.Dirs[key] = nil
				}
				r.mu.Unlock()
			}

			if base := path.Base(relFileName); base != Filename && base != OwnersFilename {
				// Not a metadata file.
				continue
			}
			if !seen.Add(relDir) {
				// Already seen this dir.
				continue
			}

			// Schedule a read.
			eg.Go(func() error {
				if err := r.semReadMetadata.Acquire(ctx, 1); err != nil {
					return err
				}
				defer r.semReadMetadata.Release(1)

				absDir := filepath.Join(absRepoRoot, filepath.FromSlash(relDir))
				switch md, err := ReadMetadata(absDir); {
				case err != nil:
					return errors.Annotate(err, "failed to read metadata from %q", absDir).Err()
				case md != nil:
					r.mu.Lock()
					r.Dirs[key] = md
					r.mu.Unlock()
				}

				return nil
			})
		}
		return scan.Err()
	})

	return eg.Wait()
}

// readUpMissing reads metadata of directories on the node path from target to
// root, and stops as soon as it finds a directory with metadata.
func (r *mappingReader) readUpMissing(target string) error {
	root := filepath.Clean(r.Root)
	target = filepath.Clean(target)

	for {
		key, err := r.DirKey(target)
		switch {
		case err != nil:
			return err
		case r.Dirs[key] != nil:
			// Exit early.
			return nil
		}

		switch meta, err := ReadMetadata(target); {
		case err != nil:
			return errors.Annotate(err, "failed to read metadata of %q", target).Err()

		case meta != nil:
			if r.Dirs == nil {
				r.Dirs = map[string]*dirmdpb.Metadata{}
			}
			r.Dirs[key] = meta
		}

		if target == root {
			return nil
		}

		// Go up.
		parent := filepath.Dir(target)
		if parent == target {
			// We have reached the root of the file system, but not `root`.
			// This is impossible because DirKey would have failed.
			panic("impossible")
		}
		target = parent
	}
}

// DirKey returns a r.Dirs key for the given dir on the file system.
// The path must be a part of the tree under r.Root.
func (r *mappingReader) DirKey(dir string) (string, error) {
	key, err := filepath.Rel(r.Root, dir)
	if err != nil {
		return "", err
	}

	// Dir keys use forward slashes.
	key = filepath.ToSlash(key)

	if strings.HasPrefix(key, "../") {
		return "", errors.Reason("the path %q must be under the root %q", dir, r.Root).Err()
	}

	return key, nil
}

var pathSepString = string(os.PathSeparator)

// normalizeDir returns version of the dir suitable for prefix checks.
// On Windows, returns the path in the lower case.
// The returned path ends with the path separator.
func normalizeDir(dir string) string {
	if runtime.GOOS == "windows" {
		// Windows is not the only OS with case-insensitive file systems, but that's
		// the only one we support.
		dir = strings.ToLower(dir)
	}

	if !strings.HasSuffix(dir, pathSepString) {
		dir += pathSepString
	}
	return dir
}

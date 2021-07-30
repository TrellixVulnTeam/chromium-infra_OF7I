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
	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"

	dirmdpb "infra/tools/dirmd/proto"
)

// Filename is the standard name of the metadata file.
const Filename = "DIR_METADATA"

var gitBinary string

func init() {
	if runtime.GOOS != "windows" {
		gitBinary = "git"
		return
	}

	gitBinary = "git.exe"
	if _, err := exec.LookPath("git.bat"); err == nil {
		// git.bat is available. Prefer git.bat instead.
		gitBinary = "git.bat"
	}
	// Note that this function does not raise errors (by panicking).
	// Instead, if code execution needs git indeed, then it will fail with a nice
	// error message (as opposed to a stack trace from panic).
}

// ReadFile reads metadata from a file.
func ReadFile(fileName string) (*dirmdpb.Metadata, error) {
	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	ret := &dirmdpb.Metadata{}
	return ret, prototext.Unmarshal(contents, ret)
}

// ReadMetadata reads metadata from a single directory.
// See also ReadMapping.
//
// Returns (nil, nil) if the metadata is not defined.
func ReadMetadata(dir string) (*dirmdpb.Metadata, error) {
	md, err := ReadFile(filepath.Join(dir, Filename))
	if os.IsNotExist(err) {
		md, _, err = ReadOwners(dir)
	}
	return md, err
}

// ReadMapping reads all metadata from files in git in the given directories.
//
// Each directory must reside in a git checkout.
// One of the repos must be the root repo, while other repos must be its
// sub-repos. In other words, all git repos referred to by the directories
// must be subdirectories of one of the repos.
// The root dir of the root repo becomes the metadata root.
//
// Unless the form is sparse, the returned mapping includes metadata of
// ancestors and descendants of the specified directories.
// In the sparse form, metadata of only the specified directories is
// returned, which is usually much faster.
//
// Descendants of the specified directories are discovered using
// "git ls-files <dir>" and not FS walk.
// This means files outside of the repo are ignored, as well as files
// matched by .gitignore files.
// Note that when reading ancestors of the specified directories,
// the .gitignore files are not respected.
// This inconsistency should not make a difference in
// the vast majority of cases because it is confusing to have
// git-ignored DIR_METADATA in the middle of the ancestry chain.
// Such a case might indicate that DIR_METADATA files are used incorrectly.
// This behavior can be changed, but it would come with a performance penalty.
func ReadMapping(ctx context.Context, form dirmdpb.MappingForm, dirs ...string) (*Mapping, error) {
	if len(dirs) == 0 {
		return nil, nil
	}

	// Ensure all dir paths are clean and absolute, for simplicity down the road.
	for i, d := range dirs {
		var err error
		if dirs[i], err = filepath.Abs(d); err != nil {
			return nil, errors.Annotate(err, "%q", d).Err()
		}
	}

	// Group all dirs by the repo root.
	// Unlike Mapping.Repos, a mapping key is an absolute os-native file path.
	repos, err := dirsByRepoRoot(ctx, dirs)
	if err != nil {
		return nil, err
	}

	r := &mappingReader{
		Mapping:         *NewMapping(0),
		semReadMetadata: semaphore.NewWeighted(int64(runtime.NumCPU())),
	}
	r.eg, ctx = errgroup.WithContext(ctx)
	defer r.eg.Wait()

	// Find the metadata root, i.e. the root dir of the root repo.
	if r.Root, err = findMetadataRoot(repos); err != nil {
		return nil, err
	}

	// Read the metadata from the specified directories, their ancestors (for inheritance)
	// and mixins they import.
	var wgReadUpMissing sync.WaitGroup
	for _, repo := range repos {
		repo := repo

		relRepoPath, err := filepath.Rel(r.Root, repo.absRoot)
		if err != nil {
			return nil, err
		}
		r.Repos[filepath.ToSlash(relRepoPath)] = repo.Repo

		for _, dir := range repo.dirs {
			dir := dir
			wgReadUpMissing.Add(1)
			r.eg.Go(func() error {
				defer wgReadUpMissing.Done()
				err := r.readUpMissing(ctx, repo, dir)
				return errors.Annotate(err, "failed to process %q", dir).Err()
			})
		}
	}

	// Wait for readUpMissing calls to finish before proceeding because
	// readUpMissing assumes it is the only one populating r.Dirs.
	wgReadUpMissing.Wait()

	// If the form isn't sparse, then also read the descendants.
	if form != dirmdpb.MappingForm_SPARSE {
		for _, repo := range repos {
			repo := repo
			// Remove redundant dirs to avoid reading the same files multiple times.
			repo.dirs = removeRedundantDirs(repo.dirs...)
			for _, dir := range repo.dirs {
				dir := dir
				r.eg.Go(func() error {
					err := r.ReadGitFiles(ctx, repo, dir, form == dirmdpb.MappingForm_FULL)
					return errors.Annotate(err, "failed to process %q", dir).Err()
				})
			}
		}
	}
	if err := r.eg.Wait(); err != nil {
		return nil, err
	}

	// Finally, bring the mapping to the desired form.
	switch form {
	case dirmdpb.MappingForm_SPARSE:
		if err := r.Mapping.ComputeAll(); err != nil {
			return nil, err
		}
		// Trim down the mapping.
		ret := NewMapping(len(dirs))
		ret.Repos = r.Repos
		for _, dir := range dirs {
			key, err := r.DirKey(dir)
			if err != nil {
				panic(err) // Impossible: we have just used these paths above.
			}
			ret.Dirs[key] = r.Mapping.Dirs[key]
		}
		return ret, nil

	case dirmdpb.MappingForm_REDUCED:
		if err := r.Mapping.Reduce(); err != nil {
			return nil, err
		}
	case dirmdpb.MappingForm_COMPUTED, dirmdpb.MappingForm_FULL:
		if err := r.Mapping.ComputeAll(); err != nil {
			return nil, err
		}
	}

	// Clean up nils and empty entries, left mostly by readUpMissing.
	for key, md := range r.Dirs {
		if md == nil || proto.Equal(md, emptyMD) {
			delete(r.Dirs, key)
		}
	}

	return &r.Mapping, nil
}

type repoInfo struct {
	*dirmdpb.Repo
	absRoot string
	dirs    []string
}

// findMetadataRoot returns the root directory of the root repo.
func findMetadataRoot(repos map[string]*repoInfo) (string, error) {
	rootSlice := make([]string, 0, len(repos))
	for rr := range repos {
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
func dirsByRepoRoot(ctx context.Context, dirs []string) (map[string]*repoInfo, error) {
	var mu sync.Mutex
	// Most likely, dirs are in different repos, so allocate len(dirs) entries.
	ret := make(map[string]*repoInfo, len(dirs))
	eg, ctx := errgroup.WithContext(ctx)
	for _, dir := range dirs {
		dir := dir

		// Check if dir is a symlink.
		p, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return nil, err
		}
		dir = p

		eg.Go(func() error {
			cmd := exec.CommandContext(ctx, gitBinary, "-C", dir, "rev-parse", "--show-toplevel")
			stdout, err := cmd.Output()
			if err != nil {
				return errors.Annotate(err, "failed to call %q", cmd.Args).Err()
			}
			repoRoot := string(bytes.TrimSpace(stdout))

			mu.Lock()
			defer mu.Unlock()
			repo := ret[repoRoot]
			if repo == nil {
				repo = &repoInfo{
					Repo:    &dirmdpb.Repo{Mixins: map[string]*dirmdpb.Metadata{}},
					absRoot: repoRoot,
				}
				ret[repoRoot] = repo
			}
			repo.dirs = append(repo.dirs, dir)
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

// mappingReader reads Mapping from the file system.
type mappingReader struct {
	// Root is an absolute path to the metadata root directory.
	// In the case of multiple repos, it is the root dir of the root repo.
	Root string

	// Mapping is the result of reading.
	Mapping

	mu              sync.Mutex
	semReadMetadata *semaphore.Weighted
	eg              *errgroup.Group
}

// ReadGitFiles reads metadata files-in-git under dir and adds them to r.Mapping.
//
// It uses "git-ls-files <dir>" to discover the files, so for example it ignores
// files outside of the repo. See more in `git ls-files -help`.
func (r *mappingReader) ReadGitFiles(ctx context.Context, repo *repoInfo, absTreeRoot string, preserveFileStructure bool) error {
	// First, determine the key prefix.
	keyPrefixNative, err := filepath.Rel(r.Root, repo.absRoot)
	if err != nil {
		return err
	}
	keyPrefix := filepath.ToSlash(keyPrefixNative)

	// Concurrently start `git ls-files`, read its output and read the discovered
	// metadata files.

	cmd := exec.CommandContext(ctx, gitBinary, "-C", repo.absRoot, "ls-files", "--full-name", absTreeRoot)
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

		absDir := filepath.Join(repo.absRoot, filepath.FromSlash(relDir))
		r.goReadDir(ctx, repo, absDir, key)
	}
	return scan.Err()
}

// goReadDir starts a goroutine with r.eg to read the metadata of the directory.
func (r *mappingReader) goReadDir(ctx context.Context, repo *repoInfo, absDir, key string) {
	r.eg.Go(func() error {
		if err := r.semReadMetadata.Acquire(ctx, 1); err != nil {
			return err
		}
		defer r.semReadMetadata.Release(1)

		md, err := ReadMetadata(absDir)
		switch {
		case err != nil:
			return errors.Annotate(err, "failed to read metadata from %q", absDir).Err()
		case md == nil:
			return nil
		}

		r.mu.Lock()
		defer r.mu.Unlock()

		r.Dirs[key] = md

		// If the file imports mixins, read them too.
		for _, mx := range md.Mixins {
			mx := mx
			if strings.Contains(mx, "\\") {
				return errors.Reason(
					"%s: mixin path %s contains back slashes; only forward slashes are allowed",
					absDir, mx,
				).Err()
			}
			if path.Base(mx) == "DIR_METADATA" {
				return errors.Reason(
					"%s imports a file with base name 'DIR_METADATA'; this is prohibited "+
						"to avoid a wrong expectation that the imported file implicitly includes metadata from its ancesors",
					absDir,
				).Err()
			}
			if _, ok := repo.Mixins[mx]; !ok {
				repo.Mixins[mx] = nil // mark as seen
				r.eg.Go(func() error {
					mxFileName := filepath.Join(repo.absRoot, filepath.FromSlash(strings.TrimPrefix(mx, "//")))
					switch mxMd, err := ReadFile(mxFileName); {
					case err != nil:
						return errors.Annotate(err, "failed to read %q", mxFileName).Err()
					case len(mxMd.Mixins) != 0:
						return errors.Reason("%s: importing a mixin in a mixin is not supported", mxFileName).Err()
					default:
						r.mu.Lock()
						repo.Mixins[mx] = mxMd
						r.mu.Unlock()
						return nil
					}
				})
			}
		}

		return nil
	})
}

// readUpMissing reads metadata of the specified directory and its ancestors,
// or until it reaches a directory already present in r.Dirs.
func (r *mappingReader) readUpMissing(ctx context.Context, repo *repoInfo, dir string) error {
	key, err := r.DirKey(dir)
	if err != nil {
		return err
	}

	// Compute the furthest ancestor and in a format suitable for exact-match
	// checking.
	// Strictly speaking it is incorrect to use r.Root, repo root should be
	// used instead, but this would be a breaking change in Chromium.
	// For example, src/v8/DIR_METADATA doesn't specify monorail project,
	// so if this "bug" is fixed, then v8 would lose its monorail project.
	// Also note that this is consistent with nearestAncestor() function behavior,
	// which goes up the dir tree and does not stop at the git repo root.
	// TODO(nodir): fix this.
	upTo := filepath.Clean(normalizeDir(r.Root))

	dirNormalized := filepath.Clean(normalizeDir(dir))
	for {
		r.mu.Lock()
		_, seen := r.Dirs[key]
		if !seen {
			r.Dirs[key] = nil // mark as seen
		}
		r.mu.Unlock()
		if seen {
			return nil
		}

		r.goReadDir(ctx, repo, dir, key)

		if dirNormalized == upTo {
			return nil
		}

		// Go up.
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			panic(errors.Reason("reached the root of the file system, but not %q", upTo).Err())
		}
		dir = parentDir
		dirNormalized = filepath.Dir(dirNormalized)
		// Do not call r.DirKey() again - it makes a syscall.
		key = path.Dir(key)
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

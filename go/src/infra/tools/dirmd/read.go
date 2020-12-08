// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/encoding/prototext"

	"go.chromium.org/luci/common/errors"

	dirmdpb "infra/tools/dirmd/proto"
)

// Filename is the standard name of the metadata file.
const Filename = "DIR_METADATA"

// ReadMetadata reads metadata from a single directory.
// See also MappingReader.
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

// ReadMapping reads all metadata from the directory tree at root.
func ReadMapping(root string, form dirmdpb.MappingForm) (*Mapping, error) {
	r := &mappingReader{Root: root}
	if err := r.ReadAll(form); err != nil {
		return nil, err
	}
	return &r.Mapping, nil
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
	// Root is a path to the root directory.
	Root string
	// Mapping is the result of reading.
	Mapping
}

// ReadAll reads metadata from the entire directory tree, overwriting
// r.Mapping.
func (r *mappingReader) ReadAll(form dirmdpb.MappingForm) error {
	r.Mapping = *NewMapping(0)

	ctx := context.Background()
	eg, ctx := errgroup.WithContext(ctx)
	sem := semaphore.NewWeighted(100) // read up to 100 files concurrently
	var mu sync.Mutex
	eg.Go(func() error {
		return filepath.Walk(r.Root, func(dir string, info os.FileInfo, err error) error {
			switch {
			case ctx.Err() != nil:
				return ctx.Err()
			case err != nil:
				return err
			case !info.IsDir():
				return nil
			}

			// Read the metadata concurrently.
			eg.Go(func() error {
				if err := sem.Acquire(ctx, 1); err != nil {
					return err
				}
				defer sem.Release(1)

				key := r.mustDirKey(dir)
				switch meta, err := ReadMetadata(dir); {
				case err != nil:
					return errors.Annotate(err, "failed to read metadata of %q", dir).Err()

				case meta != nil:
					mu.Lock()
					r.Dirs[key] = meta
					mu.Unlock()

				case form == dirmdpb.MappingForm_FULL:
					// Put an empty metadata, so that ComputeAll() populates it below.
					mu.Lock()
					r.Dirs[key] = &dirmdpb.Metadata{}
					mu.Unlock()
				}
				return nil
			})
			return nil
		})
	})
	if err := eg.Wait(); err != nil {
		return err
	}

	switch form {
	case dirmdpb.MappingForm_REDUCED:
		r.Mapping.Reduce()
	case dirmdpb.MappingForm_COMPUTED, dirmdpb.MappingForm_FULL:
		r.Mapping.ComputeAll()
	}

	return nil
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

// mustDirKey is like DirKey, but panics on failure.
func (r *mappingReader) mustDirKey(dir string) string {
	key, err := r.DirKey(dir)
	if err != nil {
		panic(err)
	}
	return key
}

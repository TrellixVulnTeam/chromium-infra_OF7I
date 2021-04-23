// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
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
		if err := r.readInheritedMissing(target); err != nil {
			return nil, errors.Annotate(err, "failed to read metadata for %q", target).Err()
		}
	}

	if err := r.ComputeAll(); err != nil {
		return nil, err
	}

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
	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	var mu sync.Mutex

	var visit func(dir, key string) error
	visit = func(dir, key string) error {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		defer sem.Release(1)

		f, err := os.Open(dir)
		if err != nil {
			return err
		}
		defer f.Close()

		hasMeta := false
		for {
			// TODO(nodir): switch to f.ReadDir after switching to Go 1.16
			names, err := f.Readdirnames(128)
			if err == io.EOF {
				break // We have exhausted all entries in the directory.
			}
			if err != nil {
				return errors.Annotate(err, "failed to read %q", dir).Err()
			}

			for _, name := range names {
				name := name
				fullName := filepath.Join(dir, name)
				switch fileInfo, err := os.Lstat(fullName); {
				case err != nil:
					return err
				case fileInfo.IsDir():
					eg.Go(func() error {
						return visit(fullName, path.Join(key, name))
					})
				case name == Filename || name == OwnersFilename:
					hasMeta = true
				}
			}
		}

		switch {
		case hasMeta:
			switch meta, err := ReadMetadata(dir); {
			case err != nil:
				return errors.Annotate(err, "failed to read %q", dir).Err()

			case meta != nil:
				mu.Lock()
				r.Dirs[key] = meta
				mu.Unlock()
			}

		case form == dirmdpb.MappingForm_FULL:
			// Ensure the key is registered in the mapping, so that ComputeAll()
			// populates it below.
			// Must not be nil because ComputeAll() doesn't support it.
			mu.Lock()
			r.Dirs[key] = &dirmdpb.Metadata{}
			mu.Unlock()
		}

		return nil
	}

	eg.Go(func() error {
		return visit(r.Root, ".")
	})
	if err := eg.Wait(); err != nil {
		return err
	}

	switch form {
	case dirmdpb.MappingForm_REDUCED:
		return r.Mapping.Reduce()
	case dirmdpb.MappingForm_COMPUTED, dirmdpb.MappingForm_FULL:
		return r.Mapping.ComputeAll()
	default:
		return nil
	}
}

// readInheritedMissing reads metadata of the target dir and its inheritance chain,
// and stops as soon as it finds a directory with metadata.
func (r *mappingReader) readInheritedMissing(target string) error {
	root := filepath.Clean(r.Root)
	target = filepath.Clean(target)

	chain := stringset.New(1) // to detect cycles
	for {
		chain.Add(target)
		key, err := r.DirKey(target)
		switch {
		case err != nil:
			return err
		case r.Dirs[key] != nil:
			// Exit early.
			return nil
		}

		meta, err := ReadMetadata(target)
		switch {
		case err != nil:
			return errors.Annotate(err, "failed to read metadata of %q", target).Err()

		case meta != nil:
			if r.Dirs == nil {
				r.Dirs = map[string]*dirmdpb.Metadata{}
			}
			r.Dirs[key] = meta
		}

		// Transition to the next node in the inheritance chain.
		var next string
		switch {
		case meta.GetInheritFrom() == "" && target == root:
			return nil

		case meta.GetInheritFrom() == "":
			next = filepath.Dir(target)

		case meta.InheritFrom == NoInheritance:
			return nil

		case strings.HasPrefix(meta.InheritFrom, "//"):
			next = filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(meta.InheritFrom, "//")))

		default:
			return errors.Reason("unexpected inherit_from value %q in dir %q", meta.InheritFrom, target).Err()
		}

		if chain.Has(next) {
			return errors.Reason("inheritance cycle with dir %q is detected", target).Err()
		}
		target = next
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

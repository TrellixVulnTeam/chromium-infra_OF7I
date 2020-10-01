package main

import (
	"sync"

	"go.chromium.org/luci/common/data/stringset"
)

type fileHashMap struct {
	sync.Mutex

	// Maps from source file name to the SHA256 hash of its content.
	filehashes map[string]string

	// Maps back from SHA256 hash to source file name.
	// Used to debug cases where duplicate files are added to the zip.
	filenamesByHash map[string]string
}

type concurrentSet struct {
	sync.Mutex
	set stringset.Set
}

// Add attempts to add string s to the set.
// Returns true if added and false if s is already in set.
func (cs *concurrentSet) Add(s string) bool {
	cs.Lock()
	defer cs.Unlock()

	if cs.set.Has(s) {
		return false
	}

	cs.set.Add(s)
	return true
}

// Add stores the filename and hash in the relevant dicts.
// Returns true if the filehash hasn't been seen before, otherwise false.
func (m *fileHashMap) Add(fname, contentHash string) bool {
	m.Lock()
	defer m.Unlock()

	m.filehashes[fname] = contentHash
	if _, ok := m.filenamesByHash[contentHash]; ok {
		return false
	}
	m.filenamesByHash[contentHash] = fname
	return true
}

// Filename returns the filename associated with the contentHash.
// Returns ("", false) if not found.
func (m *fileHashMap) Filename(contentHash string) (fname string, ok bool) {
	m.Lock()
	defer m.Unlock()

	fname, ok = m.filenamesByHash[contentHash]
	return
}

// Filehash returns the hash associated with the filename.
// Returns ("", false) if not found.
func (m *fileHashMap) Filehash(fname string) (contentHash string, ok bool) {
	m.Lock()
	defer m.Unlock()

	contentHash, ok = m.filehashes[fname]
	return
}

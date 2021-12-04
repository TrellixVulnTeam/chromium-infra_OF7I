// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package image provides functions to operate on container images efficiently.
package image

import (
	"fmt"
	"log"
	"sort"

	"github.com/google/go-containerregistry/pkg/v1/google"
)

// List represents the container images of the application we check.
// It provides conveniences to access/operate images by the image tag.
type List struct {
	// TagToDigest is a map from a tag name to the digest of the image tagged.
	TagToDigest map[string]string

	// name is the name of the image list.
	name string
	// byCreated is a list of 'google.ManifestInfo' ordered by the created time.
	// Note: we cannot use updated time to order because tagging/untagging
	// changes the updated time.
	byCreated []*google.ManifestInfo
	// tagToIndex is a map from a tag to the index of the ManifestInfo in the
	// list of 'byCreated'. It helps us to find the neighbour manifests quickly
	// by tag name.
	tagToIndex map[string]int
}

// NewList creates a new List instance.
func NewList(name string, manifests map[string]google.ManifestInfo) *List {
	q := make([]*google.ManifestInfo, 0, len(manifests))
	for digest := range manifests {
		m := manifests[digest]
		// It makes no sense to us if a manifest has no tags. So filter them
		// out.
		if len(m.Tags) == 0 {
			continue
		}
		q = append(q, &m)
	}
	// Using heap sort may be better since we usually only need to check the top
	// N images. But because we don't have too much data (e.g. ~1K images for
	// drone), using basic sort method is easier to understand.
	sort.Slice(q, func(i, j int) bool { return q[i].Created.After(q[j].Created) })

	// Construct the tag to index map.
	ti := map[string]int{}
	for i, mi := range q {
		for _, t := range mi.Tags {
			ti[t] = i
		}
	}

	// Construct the tag to digest map.
	td := map[string]string{}
	for digest, m := range manifests {
		for _, t := range m.Tags {
			td[t] = digest
		}
	}
	return &List{
		TagToDigest: td,

		name:       name,
		byCreated:  q,
		tagToIndex: ti,
	}
}

func (i *List) String() string {
	return i.name
}

// Manifest gets the manifest info of a given tag name.
func (i *List) Manifest(tagName string) (*google.ManifestInfo, bool) {
	idx, ok := i.tagToIndex[tagName]
	if !ok {
		return nil, false
	}
	return i.byCreated[idx], true
}

// NewestTag returns a tag of the newest/latest image.
// It doesn't matter to return which tag when the image has multiple tags since
// any of them can identify the image.
func (i *List) NewestTag() (string, bool) {
	if len(i.byCreated) == 0 {
		return "", false
	}
	for _, t := range i.byCreated {
		// All manifests without tags have been filtered out.
		return t.Tags[0], true
	}
	return "", false
}

// TraverseToNewer traverses the image list to the newer direction.
func (i *List) TraverseToNewer(startingTag string, f func(*google.ManifestInfo) (bool, error)) (bool, error) {
	nextNewer := func(idx int) (*google.ManifestInfo, bool) {
		if idx == 0 {
			return nil, false // No more newer image.
		}
		return i.byCreated[idx-1], true
	}
	ok, err := i.traverse(startingTag, nextNewer, f)
	if err != nil {
		return false, fmt.Errorf("traverse to newer: %s", err)
	}
	return ok, nil
}

// TraverseToOlder traverses the image list to the older direction.
func (i *List) TraverseToOlder(startingTag string, f func(*google.ManifestInfo) (bool, error)) (bool, error) {
	nextOlder := func(idx int) (*google.ManifestInfo, bool) {
		if idx == len(i.byCreated)-1 {
			return nil, false // No more older image.
		}
		return i.byCreated[idx+1], true
	}
	ok, err := i.traverse(startingTag, nextOlder, f)
	if err != nil {
		return false, fmt.Errorf("traverse to older: %s", err)
	}
	return ok, nil
}

// traverse traverses the image list from the image identified by 'tag'.
// The callback function will be called for all the images (the starting image
// included) and it uses the boolean return value to indicate whether stop
// (true) or not (false).
func (i *List) traverse(startingTag string, next func(int) (*google.ManifestInfo, bool), f func(*google.ManifestInfo) (bool, error)) (bool, error) {
	m, ok := i.Manifest(startingTag)
	if !ok {
		return false, fmt.Errorf("traverse %q: starting tag %q not found", i, startingTag)
	}
	for {
		stop, err := f(m)
		if err != nil {
			return false, fmt.Errorf("traverse %q: %s", i, err)
		}
		if stop {
			return true, nil
		}

		// All manifests without tags have been filtered out.
		t := m.Tags[0]
		idx, ok := i.tagToIndex[t]
		if !ok {
			return false, fmt.Errorf("traverse %q: data inconsistency: %q not found", i, t)
		}
		m, ok = next(idx)
		if !ok {
			return false, nil // No more image available.
		}
	}
}

// NewerThan indicates whether the image identified by tagName1 is newer (in
// term of building time) than the one identified by tagName2.
func (i *List) NewerThan(tagName1, tagName2 string) (bool, error) {
	idx1, ok1 := i.tagToIndex[tagName1]
	if !ok1 {
		return false, fmt.Errorf("cannot find tag %q in images of %q", tagName1, i)
	}
	idx2, ok2 := i.tagToIndex[tagName2]
	if !ok2 {
		return false, fmt.Errorf("cannot find tag %q in images of %q", tagName2, i)
	}
	return idx1 < idx2, nil
}

// PutTag puts 'tag' to the images identified by 'target'.
// The function is not transaction safe so the caller MUST ensure 'target' is
// existing in the image list.
// It's OK that 'tag' is non-existing. In this case, it's equivalent to add a
// new tag to the 'target'.
func (i *List) PutTag(tag, target string) error {
	// Moving a tag equivalent to delete tag and then add the tag.
	if err := i.DeleteTag(tag); err != nil {
		return fmt.Errorf("put tag: %s", err)
	}
	if err := i.addTag(tag, target); err != nil {
		return fmt.Errorf("put tag: %s", err)
	}
	return nil
}

// addTag adds a new tag to the image identified by the 'target' tag.
func (i *List) addTag(tag, target string) error {
	idx, ok := i.tagToIndex[target]
	if !ok {
		return fmt.Errorf("add tag to images %q: cannot find target tag %q", i, target)
	}
	// We have ensured all manifests has at least one tag.
	i.TagToDigest[tag] = i.TagToDigest[i.byCreated[idx].Tags[0]]
	i.byCreated[idx].Tags = append(i.byCreated[idx].Tags, tag)
	i.tagToIndex[tag] = idx
	return nil
}

// DeleteTag deletes a tag from a manifest of the images.
// We cannot delete the only tag of a manifest.
// We don't return errors when try to delete a non-existing tag.
func (i *List) DeleteTag(tag string) error {
	idx, ok := i.tagToIndex[tag]
	if !ok {
		log.Printf("%q: Skip deleting non-existing tag %q", i, tag)
		return nil
	}
	tags := i.byCreated[idx].Tags
	tagCount := len(tags)
	if tagCount <= 1 {
		return fmt.Errorf("delete tag of images %q: it is the only tag %q", i, tag)
	}
	i.byCreated[idx].Tags = removeFrom(tag, tags)
	delete(i.tagToIndex, tag)
	delete(i.TagToDigest, tag)

	return nil
}

// removeFrom removes a string from a string slice.
func removeFrom(s string, slice []string) []string {
	for i, e := range slice {
		if e == s {
			slice[i] = slice[len(slice)-1]
			return slice[:len(slice)-1]
		}
	}
	return slice
}

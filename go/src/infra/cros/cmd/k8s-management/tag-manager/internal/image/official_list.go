// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package image

import (
	"fmt"
	"log"
	"regexp"

	"github.com/google/go-containerregistry/pkg/v1/google"
)

// OfficialList is a view of image list with official images only.
type OfficialList struct {
	OfficialTagRegex *regexp.Regexp
	RawImages        *List
}

func (o *OfficialList) String() string {
	return o.RawImages.String()
}

// officialTagForManifest returns an official tag of the given manifest if it
// has.
// There's no guarantee which one will be returned when the manifest has
// multiple tags match the official tag pattern.
func (o *OfficialList) officialTagForManifest(m *google.ManifestInfo) (string, bool) {
	for _, t := range m.Tags {
		if o.OfficialTagRegex.Match([]byte(t)) {
			return t, true
		}
	}
	return "", false
}

// GetOfficialTag returns an official tag of the image identified by the tag.
// There's no guarantee which one will be returned when the image has multiple
// tags match the official tag pattern.
func (o *OfficialList) GetOfficialTag(tag string) (string, bool) {
	m, ok := o.RawImages.Manifest(tag)
	if !ok {
		return "", false
	}
	return o.officialTagForManifest(m)
}

// NewestTag returns the tag of the newest/latest official image.
func (o *OfficialList) NewestTag() (string, bool) {
	tag, ok := o.RawImages.NewestTag()
	if !ok {
		return "", false
	}
	found, _ := o.RawImages.TraverseToOlder(tag, func(m *google.ManifestInfo) (bool, error) {
		tag, ok = o.officialTagForManifest(m)
		if ok {
			return true, nil
		}
		return false, nil
	})
	if !found {
		return "", false
	}
	return tag, true
}

// PutTag puts the tag to the image identified by 'target'.
func (o *OfficialList) PutTag(tag, target string) error {
	if err := o.RawImages.PutTag(tag, target); err != nil {
		return fmt.Errorf("official list put tag: %s", err)
	}
	return nil
}

// MoveTag moves the tag along with the official images 'steps' images away on
// the specified direction.
func (o *OfficialList) MoveTag(tag string, steps int) error {
	traverse := o.RawImages.TraverseToNewer
	if steps < 0 {
		traverse = o.RawImages.TraverseToOlder
		steps = -steps
	}
	ok, err := traverse(tag, func(m *google.ManifestInfo) (bool, error) {
		lastTag, ok := o.officialTagForManifest(m)
		if !ok {
			return false, nil
		}
		steps--
		// The first element we traverse is the current image (instead of the
		// one next to it), so when we meet the target image, the 'steps'
		// is -1.
		if steps < 0 {
			if err := o.PutTag(tag, lastTag); err != nil {
				return false, fmt.Errorf("move tag %q of %q: %s", tag, o, err)
			}
			log.Printf("%q: Tag moved locally: %q->%q", o, tag, lastTag)
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("move tag %q of %q: %s", tag, o, err)
	}
	if !ok {
		if err := o.RawImages.DeleteTag(tag); err != nil {
			return fmt.Errorf("move tag %q of %q: %s", tag, o, err)
		}
		log.Printf("%q: Tag moved locally %q", o, tag)
		return nil
	}
	return nil
}

// Distance returns the official image count and the direction from 'tag1' to
// 'tag2'.
// The distance returned is a signed number. The absolute value means the images
// count between the two tags. The sign indicates the if tag1 is newer
// (positive) or older (negative) than tag2.
func (o *OfficialList) Distance(tag1, tag2 string) (int, error) {
	// If tag1 is newer than tag2, then we can start from tag2 along the
	// newer direction to reach tag1.
	newer, err := o.RawImages.NewerThan(tag1, tag2)
	if err != nil {
		return 0, fmt.Errorf("distance %q (%s<->%s): %s", o, tag1, tag2, err)
	}
	traverse := o.RawImages.TraverseToOlder
	if newer {
		traverse = o.RawImages.TraverseToNewer
	}

	distance := 0
	ok, err := traverse(tag2, func(m *google.ManifestInfo) (bool, error) {
		for _, t := range m.Tags {
			if t == tag1 {
				return true, nil
			}
		}
		if _, ok := o.officialTagForManifest(m); ok {
			distance++
		}
		return false, nil
	})
	if err != nil {
		return 0, fmt.Errorf("distance %q (%s<->%s): %s", o, tag1, tag2, err)
	}
	if !ok {
		return 0, fmt.Errorf("this shouldn't happen")
	}
	if !newer {
		distance = -distance
	}
	return distance, nil
}

// Align attempts to align a tag on an official image.
//
// If the tag is on an official image, nothing to do.
// If the tag is non-existing, put the tag on the newest official image if
// there is.
// If the tag is on a non-official image, move the tag to the oldest newer
// official image. If there's no oldest newer official image, remove the tag.
//
// The boolean return value indicates whether the tag has been aligned. It can
// be false when the tag is removed.
func (o *OfficialList) Align(tag string) (bool, error) {
	if _, ok := o.RawImages.Manifest(tag); !ok {
		t, ok := o.NewestTag()
		if !ok {
			log.Printf("%q: Cannot align %q: no newest official tag found", o, tag)
			if err := o.RawImages.DeleteTag(tag); err != nil {
				return false, fmt.Errorf("align %q on official: %s", tag, err)
			}
			return false, nil
		}
		if err := o.PutTag(tag, t); err != nil {
			return false, fmt.Errorf("align %q on official: %s", tag, err)
		}
		return true, nil
	}
	ok, err := o.RawImages.TraverseToOlder(tag, func(m *google.ManifestInfo) (bool, error) {
		if t, ok := o.officialTagForManifest(m); ok {
			if err := o.PutTag(tag, t); err != nil {
				return false, fmt.Errorf("align %q on official: %s", tag, err)
			}
			log.Printf("%q: Tag moved locally: %q->%q", o, tag, t)
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return false, fmt.Errorf("align %q on official: %s", tag, err)
	}
	if !ok {
		log.Printf("%q: No more official images available. Will remove %q", o, tag)
		if err := o.RawImages.DeleteTag(tag); err != nil {
			return false, fmt.Errorf("align %q on official: %s", tag, err)
		}
		return false, nil
	}
	return true, nil
}

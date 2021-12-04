// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"log"

	"infra/cros/cmd/k8s-management/tag-manager/internal/image"
)

// tagPolicy is the interface for all tag policies.
// A policy is the actions apply to a tag. One example is the max distance
// between a tag and another.
type tagPolicy interface {
	// apply applies the policy of a tag to an image list.
	apply(*image.OfficialList) error
	// controlledTag returns the name of the tag controlled by the policy.
	// One tag policy has one and only one tag to control.
	controlledTag() string
}

// maxDistancePolicy defines a policy of the max distance of two tags.
// If the distance is further than the requirement, the policy will move the tag
// to control towards the tag to follow until it meets the max distance
// requirement.
type maxDistancePolicy struct {
	tagToControl    string
	tagToFollow     string
	maxVersionNewer uint
	maxVersionOlder uint
}

func (p *maxDistancePolicy) String() string {
	return fmt.Sprintf("max distance policy (%q, %q)", p.tagToControl, p.tagToFollow)
}

func (p *maxDistancePolicy) apply(img *image.OfficialList) error {
	log.Printf("Applying %q", p)
	// The positive (negative) distance means the tag to control is newer
	// (older) than the tag to follow.
	d, err := img.Distance(p.tagToControl, p.tagToFollow)
	if err != nil {
		return fmt.Errorf("apply %q: %s", p, err)
	}
	if n := int(p.maxVersionNewer); d > n {
		if err := img.MoveTag(p.tagToControl, n-d); err != nil {
			return fmt.Errorf("apply %q: %s", p, err)
		}
	}
	if o := -int(p.maxVersionOlder); d < o {
		if err := img.MoveTag(p.tagToControl, o-d); err != nil {
			return fmt.Errorf("apply %q: %s", p, err)
		}
	}
	return nil
}

func (p *maxDistancePolicy) controlledTag() string { return p.tagToControl }

// latestPolicy is a tag policy that always pin the controlled tag to the
// newest image of the list to be applied.
type latestPolicy struct {
	tag string
}

func (p *latestPolicy) String() string {
	return fmt.Sprintf("latest policy (%q)", p.tag)
}

func (p *latestPolicy) apply(img *image.OfficialList) error {
	target, ok := img.NewestTag()
	if !ok {
		log.Printf("%q: apply %q: No newest tag found", img, p)
		return nil
	}
	if err := img.PutTag(p.controlledTag(), target); err != nil {
		return fmt.Errorf("apply %q: %s", p, err)
	}
	return nil
}

func (p *latestPolicy) controlledTag() string { return p.tag }

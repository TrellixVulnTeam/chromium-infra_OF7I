// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"infra/cros/cmd/k8s-management/tag-manager/internal/image"
)

// appConfig is the image config for a K8s application.
type appConfig struct {
	// officialTagRegex is a regex to check whether a tag is official or not.
	officialTagRegex *regexp.Regexp
	// policies is the tag policies apply to the application images.
	policies []tagPolicy
}

// newAppConfig creates a new appConfig.
func newAppConfig(officialTagPattern string, policies ...tagPolicy) *appConfig {
	if !strings.HasPrefix(officialTagPattern, "^") || !strings.HasSuffix(officialTagPattern, "$") {
		panic(fmt.Sprintf("Official tag pattern %q must start with '^' and end with '$'", officialTagPattern))
	}

	// One tag can and only can be controlled by one policy.
	set := map[string]bool{}
	for _, p := range policies {
		t := p.controlledTag()
		if _, ok := set[t]; ok {
			panic(fmt.Sprintf("duplicated reserved tag found: %q", t))
		}
		set[t] = true
	}

	return &appConfig{
		officialTagRegex: regexp.MustCompile(officialTagPattern),
		policies:         policies,
	}
}

// apply applies tag policies to the repo.
func (a *appConfig) apply(repo ImageRepo) error {
	log.Printf("%q: Applying tag policies", repo.Name())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	t, err := repo.List(ctx)
	if err != nil {
		return fmt.Errorf("apply %q: %s", repo.Name(), err)
	}
	img := image.NewList(repo.Name(), t.Manifests)
	oImg := &image.OfficialList{
		OfficialTagRegex: a.officialTagRegex,
		RawImages:        img,
	}
	for _, p := range a.policies {
		t := p.controlledTag()
		// Get current digests in order to know if the tags changed later.
		oldDigest := img.TagToDigest[t]

		aligned, err := oImg.Align(t)
		if err != nil {
			return fmt.Errorf("apply %q: %s", p, err)
		}
		if aligned {
			if err := p.apply(oImg); err != nil {
				return fmt.Errorf("apply policy %q to %q: %s", p, repo.Name(), err)
			}
		} else {
			log.Printf("%q: applying %q: remove the tag %q due to no official images to align", repo.Name(), p, t)
		}

		// Update remote if applies.
		if newDigest := img.TagToDigest[t]; newDigest != oldDigest {
			if err := updateRemoteRepo(ctx, repo, t, oImg); err != nil {
				return fmt.Errorf("apply policy %q to %q: %s", p, repo.Name(), err)
			}
		} else {
			log.Printf("%q: Skip updating %q (no changes)", repo.Name(), t)
		}
	}
	return nil
}

// updateRemoteRepo updates the tag on the remote side.
func updateRemoteRepo(ctx context.Context, repo ImageRepo, tag string, oImg *image.OfficialList) error {
	if newTag, ok := oImg.GetOfficialTag(tag); ok {
		if err := repo.Tag(ctx, tag, newTag); err != nil {
			return fmt.Errorf("update remote repo %q:%q: %s", repo.Name(), tag, err)
		}
	} else {
		if err := repo.Untag(ctx, tag); err != nil {
			return fmt.Errorf("update remote repo %q:%q: %s", repo.Name(), tag, err)
		}
	}
	return nil
}

// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// ImageRepo is an interface for a container image repo.
type ImageRepo interface {
	// List lists all tags of the repo.
	List(context.Context) (*google.Tags, error)
	// Tag updates (adds or moves) the tag on the remote repo.
	Tag(context.Context, string, string) error
	// Untag remove the tag from the remote repo.
	Untag(context.Context, string) error
	// Name returns the name of the repo.
	Name() string
}

// gcrRepo is an implementation of ImageRepo interface for the repository hosted
// on 'gcr.io'.
type gcrRepo struct {
	name string
	auth authn.Authenticator
}

// List implements the List of ImageRepo.
func (g *gcrRepo) List(ctx context.Context) (*google.Tags, error) {
	repo, err := name.NewRepository(g.name)
	if err != nil {
		return nil, fmt.Errorf("list %q: %s", g.name, err)
	}
	t, err := google.List(repo, google.WithContext(ctx), google.WithAuth(g.auth))
	if err != nil {
		return nil, fmt.Errorf("list %q: %s", g.name, err)
	}
	return t, nil
}

// Tag implements the Tag of ImageRepo.
func (g *gcrRepo) Tag(ctx context.Context, newTag, existingTag string) error {
	targetImg := fmt.Sprintf("%s:%s", g.name, existingTag)
	ref, err := name.ParseReference(targetImg)
	if err != nil {
		return fmt.Errorf("tag remote %q with %q: %s", targetImg, newTag, err)
	}
	desc, err := remote.Get(ref, remote.WithContext(ctx), remote.WithAuth(g.auth))
	if err != nil {
		return fmt.Errorf("tag remote %q with %q: %s", targetImg, newTag, err)
	}
	dst := desc.Ref.Context().Tag(newTag)
	if err := remote.Tag(dst, desc, remote.WithContext(ctx), remote.WithAuth(g.auth)); err != nil {
		return fmt.Errorf("tag remote %q with %q: %s", targetImg, newTag, err)
	}
	log.Printf("%q: Remote tagged %q->%q", g.name, existingTag, newTag)
	return nil
}

// Untag implements the Untag of ImageRepo.
func (g *gcrRepo) Untag(ctx context.Context, tag string) error {
	ref, err := name.ParseReference(fmt.Sprintf("%s:%s", g.name, tag))
	if err != nil {
		return fmt.Errorf("untag remote %q of %q: %s", g, tag, err)
	}
	if err := remote.Delete(ref, remote.WithContext(ctx), remote.WithAuth(g.auth)); err != nil {
		return fmt.Errorf("untag remote %q of %q: %s", g, tag, err)
	}
	log.Printf("%q: Remote untagged %q", g.name, tag)
	return nil
}

func (g *gcrRepo) Name() string { return g.name }

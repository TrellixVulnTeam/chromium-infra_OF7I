// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package cloudbuild wraps interaction with Google Cloud Build.
package cloudbuild

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/option"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/cloudbuildhelper/docker"
	"infra/cmd/cloudbuildhelper/manifest"
	"infra/cmd/cloudbuildhelper/storage"
)

// Builder knows how to trigger Cloud Build builds and check their status.
type Builder struct {
	builds   *cloudbuild.ProjectsLocationsBuildsService
	location string // "projects/.../locations/..."
	pool     *cloudbuild.PoolOption
	cfg      manifest.CloudBuildBuilder
}

// Request specifies what we want to build and push.
//
// It is passed to Trigger.
type Request struct {
	// Source is a reference to the uploaded tarball with the context directory.
	Source *storage.Object

	// Image is a name of the image (without ":<tag>") to build and push.
	//
	// Should include a docker registry part, e.g. have form "gcr.io/.../...".
	//
	// The builder will attach some tag to it and return the final full image name
	// as part of Trigger reply. The caller should still prefer Build.OutputDigest
	// (if it is available) over resolving the returned tag.
	Image string

	// Labels is a labels to put into the produced docker image (if any).
	//
	// May be ignored if the builder doesn't support setting them.
	Labels docker.Labels
}

// Status is possible status of a Cloud Build.
type Status string

// See https://cloud.google.com/cloud-build/docs/api/reference/rest/Shared.Types/Status
const (
	StatusUnknown       Status = "STATUS_UNKNOWN"
	StatusQueued        Status = "QUEUED"
	StatusWorking       Status = "WORKING"
	StatusSuccess       Status = "SUCCESS"
	StatusFailure       Status = "FAILURE"
	StatusInternalError Status = "INTERNAL_ERROR"
	StatusTimeout       Status = "TIMEOUT"
	StatusCancelled     Status = "CANCELLED"
)

// IsTerminal is true if the build is done (successfully or not).
func (s Status) IsTerminal() bool {
	switch s {
	case StatusSuccess, StatusFailure, StatusInternalError, StatusTimeout, StatusCancelled:
		return true
	default:
		return false
	}
}

// Build represents a pending, in-flight or completed build.
type Build struct {
	// Theses fields are always available.
	ID            string // UUID string with the build ID
	LogURL        string // URL to a UI page (for humans) with build logs
	Status        Status // see the enum
	StatusDetails string // human readable string with more details (if any)

	// These fields are available only for successful builds.
	InputHashes  map[string]string // SHA256 hashes of build inputs ("gs://..." => SHA256)
	OutputImage  string            // uploaded full image name (with ":tag"), if known
	OutputDigest string            // digest (in "sha256:..." form) of the image, if known
}

// New prepares a Builder instance.
func New(ctx context.Context, ts oauth2.TokenSource, cfg manifest.CloudBuildBuilder) (*Builder, error) {
	svc, err := cloudbuild.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, errors.Annotate(err, "failed to instantiate cloudbuild.Service").Err()
	}

	// When not using private pools we can hit "global" location in the API.
	// When using a private pool we need to hit the location that hosts it,
	// otherwise Cloud Build replies with "404 no such workerpool".
	location := fmt.Sprintf("projects/%s/locations/global", cfg.Project)

	var pool *cloudbuild.PoolOption
	if cfg.Pool != nil && cfg.Pool.ID != "" {
		location = fmt.Sprintf("projects/%s/locations/%s", cfg.Project, cfg.Pool.Region)
		pool = &cloudbuild.PoolOption{
			Name: fmt.Sprintf("%s/workerPools/%s", location, cfg.Pool.ID),
		}
	}

	return &Builder{
		builds:   svc.Projects.Locations.Builds,
		location: location,
		pool:     pool,
		cfg:      cfg,
	}, nil
}

// Trigger launches a new build, returning its details.
//
// ID of the returned build can be used to query its status later in Check(...).
func (b *Builder) Trigger(ctx context.Context, r Request) (build *Build, image string, err error) {
	// Cloud Build always pushes the tagged image to the registry. The default
	// tag is "latest", and we don't want to use it in case someone decides to
	// rely on it. So pick something more cryptic.
	//
	// Note that when building with !b.cfg.PushesExplicitly (i.e. relying on Cloud
	// Build to do the push) we don't really care if this tag is moved
	// concurrently by someone else. We never read it, we consume only the image
	// digest returned directly by Cloud Build API.
	//
	// When building with b.cfg.PushesExplicitly we should be careful the tag we
	// push as is not being moved by someone else, since we'll need it to get the
	// final SHA256 of the image, since Cloud Build would not return it in its API
	// response.
	image = r.Image
	if !b.cfg.PushesExplicitly {
		image += ":cbh"
	} else {
		image += ":cbh-" + randomTag()
	}

	// Render variables in `args`.
	stepArgs := make([]string, 0, len(b.cfg.Args))
	for _, arg := range b.cfg.Args {
		switch arg {
		case "${CBH_DOCKER_IMAGE}":
			stepArgs = append(stepArgs, image)
		case "${CBH_DOCKER_LABELS}":
			stepArgs = append(stepArgs, r.Labels.AsBuildArgs()...)
		default:
			stepArgs = append(stepArgs, arg)
		}
	}

	// Ask Cloud Build to do the final image push if possible.
	var images []string
	if !b.cfg.PushesExplicitly {
		images = []string{image}
	}

	// Log details to help debugging the configuration.
	logging.Infof(ctx, "Enqueuing a Cloud Build job:")
	logging.Infof(ctx, "  Location: %s", b.location)
	logging.Infof(ctx, "  Executable: %s", b.cfg.Executable)
	logging.Infof(ctx, "  Args: %v", stepArgs)

	// See https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds#Build
	call := b.builds.Create(b.location, &cloudbuild.Build{
		Options: &cloudbuild.BuildOptions{
			LogStreamingOption:    "STREAM_ON",
			Logging:               "GCS_ONLY",
			RequestedVerifyOption: "VERIFIED",
			SourceProvenanceHash:  []string{"SHA256"},
			Pool:                  b.pool,
		},
		Timeout: b.cfg.Timeout,
		Source: &cloudbuild.Source{
			StorageSource: &cloudbuild.StorageSource{
				Bucket:     r.Source.Bucket,
				Object:     r.Source.Name,
				Generation: r.Source.Generation,
			},
		},
		Steps: []*cloudbuild.BuildStep{
			{
				Name: b.cfg.Executable,
				Args: stepArgs,
			},
		},
		Images: images,
	})

	op, err := call.Context(ctx).Do()
	if err != nil {
		return nil, "", errors.Annotate(err, "API call to Cloud Build failed").Err()
	}

	// Cloud Build returns triggered build details with operation's metadata.
	var metadata struct {
		Build *cloudbuild.Build `json:"build"`
	}
	if err := json.Unmarshal(op.Metadata, &metadata); err != nil {
		return nil, "", errors.Annotate(err, "failed to unmarshal operations metadata %s", op.Metadata).Err()
	}
	if metadata.Build == nil {
		return nil, "", errors.Reason("`build` field unexpectedly missing in the metadata %s", op.Metadata).Err()
	}

	return makeBuild(metadata.Build), image, nil
}

// Check returns details of a build given its ID.
func (b *Builder) Check(ctx context.Context, bid string) (*Build, error) {
	build, err := b.builds.Get(fmt.Sprintf("%s/builds/%s", b.location, bid)).Context(ctx).Do()
	if err != nil {
		return nil, errors.Annotate(err, "API call to Cloud Build failed").Err()
	}
	return makeBuild(build), nil
}

func makeBuild(b *cloudbuild.Build) *Build {
	// Parse SourceProvenance into more digestible "file => SHA256" map.
	var prov map[string]string
	if b.SourceProvenance != nil {
		prov = make(map[string]string, len(b.SourceProvenance.FileHashes))
		for name, hashes := range b.SourceProvenance.FileHashes {
			digest := "<unknown>"
			for _, h := range hashes.FileHash {
				if h.Type == "SHA256" {
					digest = b64ToHex(h.Value)
					break
				}
			}
			prov[name] = digest
		}
	}

	// Grab the image from the result. There should be at most one.
	var outImg, outDigest string
	if b.Results != nil {
		for _, img := range b.Results.Images {
			outImg = img.Name
			outDigest = img.Digest
			break
		}
	}

	return &Build{
		ID:            b.Id,
		LogURL:        b.LogUrl,
		Status:        Status(b.Status),
		StatusDetails: b.StatusDetail,
		InputHashes:   prov,
		OutputImage:   outImg,
		OutputDigest:  outDigest,
	}
}

func b64ToHex(b string) string {
	blob, err := base64.StdEncoding.DecodeString(b)
	if err != nil {
		return fmt.Sprintf("<bad hash %s>", err) // should not be happening
	}
	return hex.EncodeToString(blob)
}

func randomTag() string {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

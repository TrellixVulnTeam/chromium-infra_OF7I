// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"

	"infra/cmd/cloudbuildhelper/cloudbuild"
	"infra/cmd/cloudbuildhelper/fileset"
	"infra/cmd/cloudbuildhelper/manifest"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

const (
	testTargetName   = "test-name"
	testBucketName   = "test-bucket"
	testRegistryName = "fake.example.com/registry"
	testDigest       = "sha256:totally-legit-hash"
	testTagName      = "canonical-tag"
	testLogURL       = "https://example.com/cloud-build-log"

	testImageName = testRegistryName + "/" + testTargetName
)

var _true = true // for *bool

func TestBuild(t *testing.T) {
	t.Parallel()

	Convey("With mocks", t, func() {
		testTime := time.Date(2016, time.February, 3, 4, 5, 6, 0, time.Local)
		ctx, tc := testclock.UseTime(context.Background(), testTime)
		tc.SetTimerCallback(func(d time.Duration, t clock.Timer) {
			if testclock.HasTags(t, "sleep-timer") {
				tc.Add(d)
			}
		})
		ctx, _ = clock.WithTimeout(ctx, 20*time.Minute) // don't hang forever

		store := newStorageImplMock()
		registry := newRegistryImplMock()
		builder := newBuilderImplMock(registry)
		fs, digest := prepFileSet()

		var (
			// Path relative to the storage root.
			testTarballPath = fmt.Sprintf("%s/%s.tar.gz", testTargetName, digest)
			// Where we drops the tarball, excluding "#<generation>" suffix.
			testTarballURL = fmt.Sprintf("gs://%s/%s/%s.tar.gz", testBucketName, testTargetName, digest)
		)

		builder.outputDigests = func(img string) string {
			So(img, ShouldEqual, testImageName+":cbh")
			return testDigest
		}

		Convey("Never seen before tarball", func() {
			builder.provenance = func(gs string) string {
				So(gs, ShouldEqual, testTarballURL+"#1") // used first gen
				return digest                            // got its digest correctly
			}

			res, err := runBuild(ctx, buildParams{
				Manifest: &manifest.Manifest{
					Name:          testTargetName,
					Deterministic: &_true,
				},
				Image:        testImageName,
				BuildID:      "b1",
				CanonicalTag: testTagName,
				Tags:         []string{"latest"},
				Stage:        stageFileSet(fs),
				Store:        store,
				Builder:      builder,
				Registry:     registry,
			})
			So(err, ShouldBeNil)

			// Uploaded the file.
			obj, err := store.Check(ctx, testTarballPath)
			So(err, ShouldBeNil)
			So(obj.String(), ShouldEqual, testTarballURL+"#1") // uploaded the first gen

			// Used Cloud Build.
			So(res, ShouldResemble, buildResult{
				Image: &imageRef{
					Image:        testImageName,
					Digest:       testDigest,
					CanonicalTag: testTagName,
					BuildID:      "b1",
				},
				ViewBuildURL: testLogURL,
			})

			// Tagged it with canonical tag.
			img, err := registry.GetImage(ctx, fmt.Sprintf("%s:%s", testImageName, testTagName))
			So(err, ShouldBeNil)
			So(img.Digest, ShouldEqual, testDigest)

			// And moved "latest" tag.
			img, err = registry.GetImage(ctx, testImageName+":latest")
			So(err, ShouldBeNil)
			So(img.Digest, ShouldEqual, testDigest)

			// Now we build this exact tarball again using different canonical tag.
			// We should get back the image we've already built.
			Convey("Building existing tarball deterministically: reuses the image", func() {
				builder.provenance = func(gs string) string {
					panic("Cloud Build should not be invoked")
				}

				// To avoid clashing on metadata keys that depend on timestamps.
				tc.Add(time.Minute)

				res, err := runBuild(ctx, buildParams{
					Manifest: &manifest.Manifest{
						Name:          testTargetName,
						Deterministic: &_true,
					},
					Image:        testImageName,
					BuildID:      "b2",
					CanonicalTag: "another-tag",
					Tags:         []string{"pushed"},
					Stage:        stageFileSet(fs),
					Store:        store,
					Builder:      builder,
					Registry:     registry,
				})
				So(err, ShouldBeNil)

				// Reused the existing image.
				So(res, ShouldResemble, buildResult{
					Image: &imageRef{
						Image:        testImageName,
						Digest:       testDigest,
						CanonicalTag: testTagName,
						BuildID:      "b1", // was build there
						Timestamp:    testTime.Add(10 * time.Second),
					},
				})

				// And moved "pushed" tag, even through no new image was built.
				img, err := registry.GetImage(ctx, testImageName+":pushed")
				So(err, ShouldBeNil)
				So(img.Digest, ShouldEqual, testDigest)

				// Both builds are associated with the tarball via its metadata now.
				tarball, err := store.Check(ctx, testTarballPath)
				So(err, ShouldBeNil)
				md := tarball.Metadata.Values(buildRefMetaKey)
				So(md, ShouldHaveLength, 2)
				So(md[0].Value, ShouldEqual, `{"build_id":"b2","tag":"another-tag"}`)
				So(md[1].Value, ShouldEqual, `{"build_id":"b1","tag":"canonical-tag"}`)
			})

			// Now we build this exact tarball again using different canonical tag,
			// but mark the target as non-deterministic. It should ignore the existing
			// image and build a new one.
			Convey("Building existing tarball non-deterministically: creates new image", func() {
				builder.provenance = func(gs string) string {
					So(gs, ShouldEqual, testTarballURL+"#1") // reused first gen
					return digest
				}
				builder.outputDigests = func(img string) string {
					So(img, ShouldEqual, testImageName+":cbh")
					return "sha256:new-totally-legit-hash"
				}

				// To avoid clashing on metadata keys that depend on timestamps.
				tc.Add(time.Minute)

				res, err := runBuild(ctx, buildParams{
					Manifest: &manifest.Manifest{
						Name:          testTargetName,
						Deterministic: nil,
					},
					Image:        testImageName,
					BuildID:      "b2",
					CanonicalTag: "another-tag",
					Tags:         []string{"latest"},
					Stage:        stageFileSet(fs),
					Store:        store,
					Builder:      builder,
					Registry:     registry,
				})
				So(err, ShouldBeNil)

				// Built the new image.
				So(res, ShouldResemble, buildResult{
					Image: &imageRef{
						Image:        testImageName,
						Digest:       "sha256:new-totally-legit-hash",
						CanonicalTag: "another-tag",
						BuildID:      "b2",
					},
					ViewBuildURL: testLogURL,
				})

				// And moved "latest" tag.
				img, err = registry.GetImage(ctx, testImageName+":latest")
				So(err, ShouldBeNil)
				So(img.Digest, ShouldEqual, "sha256:new-totally-legit-hash")

				// Both builds are associated with the tarball via its metadata now.
				tarball, err := store.Check(ctx, testTarballPath)
				So(err, ShouldBeNil)
				md := tarball.Metadata.Values(buildRefMetaKey)
				So(md, ShouldHaveLength, 2)
				So(md[0].Value, ShouldEqual, `{"build_id":"b2","tag":"another-tag"}`)
				So(md[1].Value, ShouldEqual, `{"build_id":"b1","tag":"canonical-tag"}`)
			})
		})

		Convey("Already seen canonical tag", func() {
			registry.put(fmt.Sprintf("%s:%s", testImageName, testTagName), testDigest)

			res, err := runBuild(ctx, buildParams{
				Manifest:     &manifest.Manifest{Name: testTargetName},
				Image:        testImageName,
				CanonicalTag: testTagName,
				Tags:         []string{"latest"},
				Registry:     registry,
			})
			So(err, ShouldBeNil)

			// Reused the existing image.
			So(res, ShouldResemble, buildResult{
				Image: &imageRef{
					Image:        testImageName,
					Digest:       testDigest,
					CanonicalTag: testTagName,
				},
			})

			// And moved "latest" tag.
			img, err := registry.GetImage(ctx, testImageName+":latest")
			So(err, ShouldBeNil)
			So(img.Digest, ShouldEqual, testDigest)
		})

		Convey("Using :inputs-hash as canonical tag", func() {
			expectedTag := "cbh-inputs-" + digest[:24]

			builder.provenance = func(gs string) string {
				So(gs, ShouldEqual, testTarballURL+"#1") // used first gen
				return digest                            // got its digest correctly
			}

			params := buildParams{
				Manifest: &manifest.Manifest{
					Name:          testTargetName,
					Deterministic: &_true,
				},
				Image:        testImageName,
				BuildID:      "b1",
				CanonicalTag: inputsHashCanonicalTag,
				Tags:         []string{"latest"},
				Stage:        stageFileSet(fs),
				Store:        store,
				Builder:      builder,
				Registry:     registry,
			}
			res, err := runBuild(ctx, params)
			So(err, ShouldBeNil)

			// Uploaded the file.
			obj, err := store.Check(ctx, testTarballPath)
			So(err, ShouldBeNil)
			So(obj.String(), ShouldEqual, testTarballURL+"#1") // uploaded the first gen

			// Used Cloud Build.
			So(res, ShouldResemble, buildResult{
				Image: &imageRef{
					Image:        testImageName,
					Digest:       testDigest,
					CanonicalTag: expectedTag,
					BuildID:      "b1",
				},
				ViewBuildURL: testLogURL,
			})

			// Tagged it with canonical tag.
			img, err := registry.GetImage(ctx, fmt.Sprintf("%s:%s", testImageName, expectedTag))
			So(err, ShouldBeNil)
			So(img.Digest, ShouldEqual, testDigest)

			// Repeating the build reuses the existing image since inputs hash didn't
			// change (and thus its canonical tag also didn't change and we already
			// have an image with this canonical tag).
			res, err = runBuild(ctx, params)
			So(err, ShouldBeNil)
			So(res, ShouldResemble, buildResult{
				Image: &imageRef{
					Image:        testImageName,
					Digest:       testDigest,
					CanonicalTag: expectedTag,
				},
				ViewBuildURL: "", // Cloud Build wasn't used
			})
		})

		Convey("No registry is set => nothing is uploaded", func() {
			builder.provenance = func(gs string) string {
				So(gs, ShouldEqual, testTarballURL+"#1") // used first gen
				return digest                            // got its digest correctly
			}

			res, err := runBuild(ctx, buildParams{
				Manifest:     &manifest.Manifest{Name: testTargetName},
				CanonicalTag: testTagName, // ignored
				Stage:        stageFileSet(fs),
				Store:        store,
				Builder:      builder,
				Registry:     registry,
			})
			So(err, ShouldBeNil)

			// Uploaded the file.
			obj, err := store.Check(ctx, testTarballPath)
			So(err, ShouldBeNil)
			So(obj.String(), ShouldEqual, testTarballURL+"#1") // uploaded the first gen

			// Did NOT produce any image, but have a link to the build.
			So(res, ShouldResemble, buildResult{ViewBuildURL: testLogURL})
		})

		Convey("Cloud Build build failure", func() {
			builder.finalStatus = cloudbuild.StatusFailure
			_, err := runBuild(ctx, buildParams{
				Manifest: &manifest.Manifest{Name: testTargetName},
				Image:    testImageName,
				Stage:    stageFileSet(fs),
				Store:    store,
				Builder:  builder,
				Registry: registry,
			})
			So(err, ShouldErrLike, "build failed, see its logs")
		})

		Convey("Cloud Build API errors", func() {
			builder.checkCallback = func(b *runningBuild) error {
				return fmt.Errorf("boom")
			}
			_, err := runBuild(ctx, buildParams{
				Manifest: &manifest.Manifest{Name: testTargetName},
				Image:    testImageName,
				Stage:    stageFileSet(fs),
				Store:    store,
				Builder:  builder,
				Registry: registry,
			})
			So(err, ShouldErrLike, "waiting for the build to finish: too many errors, the last one: boom")
		})
	})
}

////////////////////////////////////////////////////////////////////////////////

func prepFileSet() (fs *fileset.Set, digest string) {
	fs = &fileset.Set{}

	fs.AddFromMemory("Dockerfile", []byte("boo-boo-boo"), nil)
	fs.AddFromMemory("dir/something", []byte("ba-ba-ba"), nil)

	h := sha256.New()
	if err := fs.ToTarGz(h); err != nil {
		panic(err)
	}
	digest = hex.EncodeToString(h.Sum(nil))
	return
}

func stageFileSet(fs *fileset.Set) stageCallback {
	return func(c context.Context, m *manifest.Manifest, cb func(*fileset.Set) error) error {
		return cb(fs)
	}
}

// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gitiles

import (
	"context"
	"infra/chromium/bootstrapper/gitiles"
	"regexp"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	. "go.chromium.org/luci/common/testing/assertions"
)

func strPtr(s string) *string {
	return &s
}

func TestFactory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Factory", t, func() {

		Convey("returns an RPC client by default", func() {
			factory := Factory(nil)

			client, err := factory(ctx, "fake-host")

			So(err, ShouldBeNil)
			So(client, ShouldNotBeNil)
		})

		Convey("fails for a nil host", func() {
			factory := Factory(map[string]*Host{
				"fake-host": nil,
			})

			client, err := factory(ctx, "fake-host")

			So(err, ShouldNotBeNil)
			So(client, ShouldBeNil)
		})

		Convey("returns RPC client for provided host", func() {
			host := &Host{}
			factory := Factory(map[string]*Host{
				"fake-host": host,
			})

			client, err := factory(ctx, "fake-host")

			So(err, ShouldBeNil)
			So(client, ShouldResemble, &Client{
				hostname: "fake-host",
				gitiles:  host,
			})
		})
	})
}

func logRequest(project, ref string) *gitilespb.LogRequest {
	return &gitilespb.LogRequest{
		Project:    project,
		Committish: ref,
		PageSize:   1,
	}
}

func TestLog(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("gitilesClient.Log", t, func() {

		Convey("returns a revision by default", func() {
			client, _ := Factory(nil)(ctx, "fake-host")

			response, err := client.Log(ctx, logRequest("fake/project", "refs/heads/fake-branch"))

			So(err, ShouldBeNil)
			So(response, ShouldNotBeNil)
			So(response.Log, ShouldHaveLength, 1)
			So(response.Log[0].Id, ShouldNotBeEmpty)
		})

		Convey("fails for a nil project", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": nil,
					},
				},
			})(ctx, "fake-host")

			response, err := client.Log(ctx, logRequest("fake/project", "refs/heads/fake-branch"))

			So(err, ShouldErrLike, `unknown project "fake/project" on host "fake-host"`)
			So(response, ShouldBeNil)
		})

		Convey("fails for an empty ref revision", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Refs: map[string]string{
								"refs/heads/fake-branch": "",
							},
						},
					},
				},
			})(ctx, "fake-host")

			response, err := client.Log(ctx, logRequest("fake/project", "refs/heads/fake-branch"))

			So(err, ShouldErrLike, `unknown ref "refs/heads/fake-branch" for project "fake/project" on host "fake-host"`)
			So(response, ShouldBeNil)
		})

		Convey("returns log for provided revision", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Refs: map[string]string{
								"refs/heads/fake-branch": "fake-revision",
							},
						},
					},
				},
			})(ctx, "fake-host")

			response, err := client.Log(ctx, logRequest("fake/project", "refs/heads/fake-branch"))

			So(err, ShouldBeNil)
			So(response, ShouldNotBeNil)
			So(response.Log, ShouldHaveLength, 1)
			So(response.Log[0].Id, ShouldEqual, "fake-revision")
		})

		Convey("returns log for known revision", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Revisions: map[string]*Revision{
								"fake-revision": {},
							},
						},
					},
				},
			})(ctx, "fake-host")

			response, err := client.Log(ctx, logRequest("fake/project", "fake-revision"))

			So(err, ShouldBeNil)
			So(response, ShouldNotBeNil)
			So(response.Log, ShouldHaveLength, 1)
			So(response.Log[0].Id, ShouldEqual, "fake-revision")
		})

	})
}

func downloadFileRequest(project, revision, path string) *gitilespb.DownloadFileRequest {
	return &gitilespb.DownloadFileRequest{
		Project:    project,
		Committish: revision,
		Path:       path,
	}
}

func TestDownloadFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("gitilesClient.DownloadFile", t, func() {

		Convey("fails by default", func() {
			client, _ := Factory(nil)(ctx, "fake-host")

			response, err := client.DownloadFile(ctx, downloadFileRequest("fake/project", "fake-revision", "fake/file"))

			So(err, ShouldErrLike, `unknown file "fake/file"`)
			So(response, ShouldBeNil)
		})

		Convey("fails for a nil project", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": nil,
					},
				},
			})(ctx, "fake-host")

			response, err := client.DownloadFile(ctx, downloadFileRequest("fake/project", "fake-revision", "fake/file"))

			So(err, ShouldErrLike, `unknown project "fake/project" on host "fake-host"`)
			So(response, ShouldBeNil)
		})

		Convey("fails for a nil revision", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Revisions: map[string]*Revision{
								"fake-revision": nil,
							},
						},
					},
				},
			})(ctx, "fake-host")

			response, err := client.DownloadFile(ctx, downloadFileRequest("fake/project", "fake-revision", "fake/file"))

			So(err, ShouldErrLike, `unknown revision "fake-revision" of project "fake/project" on host "fake-host"`)
			So(response, ShouldBeNil)

		})

		Convey("returns contents for provided file at revision", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Revisions: map[string]*Revision{
								"fake-revision": {
									Files: map[string]*string{
										"fake/file": strPtr("fake-contents"),
									},
								},
							},
						},
					},
				},
			})(ctx, "fake-host")

			response, err := client.DownloadFile(ctx, downloadFileRequest("fake/project", "fake-revision", "fake/file"))

			So(err, ShouldBeNil)
			So(response, ShouldNotBeNil)
			So(response.Contents, ShouldEqual, "fake-contents")
		})

		Convey("returns contents for provided file at revision where file is not affected", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Revisions: map[string]*Revision{
								"fake-revision-1": {
									Files: map[string]*string{
										"fake/file": strPtr("fake-contents"),
									},
								},
								"fake-revision-2": {
									Parent: "fake-revision-1",
								},
							},
						},
					},
				},
			})(ctx, "fake-host")

			response, err := client.DownloadFile(ctx, downloadFileRequest("fake/project", "fake-revision-2", "fake/file"))

			So(err, ShouldBeNil)
			So(response, ShouldNotBeNil)
			So(response.Contents, ShouldEqual, "fake-contents")
		})

		Convey("fails for nil contents", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Revisions: map[string]*Revision{
								"fake-revision": {
									Files: map[string]*string{
										"fake/file": nil,
									},
								},
							},
						},
					},
				},
			})(ctx, "fake-host")

			response, err := client.DownloadFile(ctx, downloadFileRequest("fake/project", "fake-revision", "fake/file"))
			So(err, ShouldErrLike, `unknown file "fake/file" at revision "fake-revision" of project "fake/project" on host "fake-host"`)
			So(response, ShouldBeNil)
		})

	})
}

func TestDownloadDiff(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("gitilesClient.DownloadFile", t, func() {

		Convey("returns an empty diff by default", func() {
			client, _ := Factory(nil)(ctx, "fake-host")

			response, err := client.DownloadDiff(ctx, &gitilespb.DownloadDiffRequest{
				Project:    "fake/project",
				Committish: "fake-revision",
			})

			So(err, ShouldBeNil)
			So(response, ShouldNotBeNil)
			So(response.Contents, ShouldBeEmpty)
		})

		Convey("fails for a nil project", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": nil,
					},
				},
			})(ctx, "fake-host")

			response, err := client.DownloadDiff(ctx, &gitilespb.DownloadDiffRequest{
				Project:    "fake/project",
				Committish: "fake-revision",
			})

			So(err, ShouldErrLike, `unknown project "fake/project" on host "fake-host"`)
			So(response, ShouldBeNil)
		})

		Convey("fails for a nil revision", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Revisions: map[string]*Revision{
								"fake-revision": nil,
							},
						},
					},
				},
			})(ctx, "fake-host")

			response, err := client.DownloadDiff(ctx, &gitilespb.DownloadDiffRequest{
				Project:    "fake/project",
				Committish: "fake-revision",
			})

			So(err, ShouldErrLike, `unknown revision "fake-revision" of project "fake/project" on host "fake-host"`)
			So(response, ShouldBeNil)
		})

		Convey("returns difference between revision and parent", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Revisions: map[string]*Revision{
								"fake-revision-1": {
									Files: map[string]*string{
										"to-re-add": strPtr("fake-contents-to-be-re-added\n"),
									},
								},
								"fake-revision-2": {
									Files: map[string]*string{
										"to-modify":         strPtr("fake-contents-1\n"),
										"to-make-non-empty": strPtr(""),
										"to-clear":          strPtr("fake-contents-to-be-removed\n"),
										"to-delete":         strPtr("fake-contents-for-file-to-be-deleted\n"),
										"to-re-add":         nil,
									},
									Parent: "fake-revision-1",
								},
								"fake-revision-3": {
									Files: map[string]*string{
										"to-modify":         strPtr("fake-contents-2\n"),
										"to-make-non-empty": strPtr("fake-contents-added\n"),
										"to-add":            strPtr("fake-contents-for-new-file\n"),
										"to-clear":          strPtr(""),
										"to-delete":         nil,
										"to-re-add":         strPtr("fake-contents-to-be-re-added\n"),
									},
									Parent: "fake-revision-2",
								},
							},
						},
					},
				},
			})(ctx, "fake-host")

			Convey("for all files if no path is specified", func() {
				response, err := client.DownloadDiff(ctx, &gitilespb.DownloadDiffRequest{
					Project:    "fake/project",
					Committish: "fake-revision-3",
				})

				So(err, ShouldBeNil)
				So(response, ShouldNotBeNil)
				// (?m) - multiline regex mode: ^ matches line start
				contents := regexp.MustCompile(`(?m)^index [0-9a-f]+\.\.[0-9a-f]+`).ReplaceAllLiteralString(response.Contents, "index A..B")
				So(contents, ShouldEqual,
					`diff --git a/to-add b/to-add
new file mode 100644
index A..B
--- /dev/null
+++ b/to-add
@@ -0,0 +1 @@
+fake-contents-for-new-file
diff --git a/to-clear b/to-clear
index A..B 100644
--- a/to-clear
+++ b/to-clear
@@ -1 +0,0 @@
-fake-contents-to-be-removed
diff --git a/to-delete b/to-delete
deleted file mode 100644
index A..B
--- a/to-delete
+++ /dev/null
@@ -1 +0,0 @@
-fake-contents-for-file-to-be-deleted
diff --git a/to-make-non-empty b/to-make-non-empty
index A..B 100644
--- a/to-make-non-empty
+++ b/to-make-non-empty
@@ -0,0 +1 @@
+fake-contents-added
diff --git a/to-modify b/to-modify
index A..B 100644
--- a/to-modify
+++ b/to-modify
@@ -1 +1 @@
-fake-contents-1
+fake-contents-2
diff --git a/to-re-add b/to-re-add
new file mode 100644
index A..B
--- /dev/null
+++ b/to-re-add
@@ -0,0 +1 @@
+fake-contents-to-be-re-added
`)

			})

			Convey("for individual file if path is specified", func() {
				response, err := client.DownloadDiff(ctx, &gitilespb.DownloadDiffRequest{
					Project:    "fake/project",
					Committish: "fake-revision-3",
					Path:       "to-modify",
				})

				So(err, ShouldBeNil)
				So(response, ShouldNotBeNil)
				// (?m) - multiline regex mode: ^ matches line start
				contents := regexp.MustCompile(`(?m)^index [0-9a-f]+\.\.[0-9a-f]+`).ReplaceAllLiteralString(response.Contents, "index A..B")
				So(contents, ShouldEqual,
					`diff --git a/to-modify b/to-modify
index A..B 100644
--- a/to-modify
+++ b/to-modify
@@ -1 +1 @@
-fake-contents-1
+fake-contents-2
`)
			})

		})

		Convey("returns difference between revisions if base is specified", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Revisions: map[string]*Revision{
								"fake-revision-1": {
									Files: map[string]*string{
										"to-modify": strPtr("fake-contents-A\n"),
									},
								},
								"fake-revision-2": {
									Files: map[string]*string{
										"to-modify": strPtr("fake-contents-A\n"),
									},
									Parent: "fake-revision-1",
								},
								"fake-revision-3": {
									Files: map[string]*string{
										"to-modify": strPtr("fake-contents-B\n"),
									},
									Parent: "fake-revision-1",
								},
							},
						},
					},
				},
			})(ctx, "fake-host")

			response, err := client.DownloadDiff(ctx, &gitilespb.DownloadDiffRequest{
				Project:    "fake/project",
				Committish: "fake-revision-2",
				Base:       "fake-revision-3",
			})

			So(err, ShouldBeNil)
			So(response, ShouldNotBeNil)
			// (?m) - multiline regex mode: ^ matches line start
			contents := regexp.MustCompile(`(?m)^index [0-9a-f]+\.\.[0-9a-f]+`).ReplaceAllLiteralString(response.Contents, "index A..B")
			So(contents, ShouldEqual,
				`diff --git a/to-modify b/to-modify
index A..B 100644
--- a/to-modify
+++ b/to-modify
@@ -1 +1 @@
-fake-contents-B
+fake-contents-A
`)
		})

	})

}

func TestIntegration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("gitiles using fake factory", t, func() {

		Convey("succeeds when calling FetchLatestRevision", func() {
			ctx := gitiles.UseGitilesClientFactory(ctx, Factory(nil))
			client := gitiles.NewClient(ctx)

			revision, err := client.FetchLatestRevision(ctx, "fake-host", "fake/project", "refs/heads/fake-branch")

			So(err, ShouldBeNil)
			So(revision, ShouldNotBeEmpty)
		})

		Convey("succeeds when calling DownloadFile", func() {
			ctx := gitiles.UseGitilesClientFactory(ctx, Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Revisions: map[string]*Revision{
								"fake-revision": {
									Files: map[string]*string{
										"fake/file": strPtr("fake-contents"),
									},
								},
							},
						},
					},
				},
			}))
			client := gitiles.NewClient(ctx)

			contents, err := client.DownloadFile(ctx, "fake-host", "fake/project", "fake-revision", "fake/file")

			So(err, ShouldBeNil)
			So(contents, ShouldEqual, "fake-contents")
		})

		Convey("succeeds when calling DownloadDiff", func() {

			Convey("with diff against parent when called for modified path", func() {
				ctx := gitiles.UseGitilesClientFactory(ctx, Factory(map[string]*Host{
					"fake-host": {
						Projects: map[string]*Project{
							"fake/project": {
								Revisions: map[string]*Revision{
									"fake-revision-1": {
										Files: map[string]*string{
											"fake/file": strPtr("fake-contents-1"),
										},
									},
									"fake-revision-2": {
										Files: map[string]*string{
											"fake/file": strPtr("fake-contents-2"),
										},
										Parent: "fake-revision-1",
									},
								},
							},
						},
					},
				}))
				client := gitiles.NewClient(ctx)

				contents, err := client.DownloadDiff(ctx, "fake-host", "fake/project", "fake-revision-2", gitiles.PARENT, "fake/file")

				So(err, ShouldBeNil)
				So(contents, ShouldNotBeEmpty)
			})

			Convey("with no diff against parent when called for unmodified path", func() {
				ctx := gitiles.UseGitilesClientFactory(ctx, Factory(map[string]*Host{
					"fake-host": {
						Projects: map[string]*Project{
							"fake/project": {
								Revisions: map[string]*Revision{
									"fake-revision-1": {
										Files: map[string]*string{
											"fake/file": strPtr("fake-contents-1"),
										},
									},
									"fake-revision-2": {
										Files: map[string]*string{
											"fake/file": strPtr("fake-contents-2"),
										},
										Parent: "fake-revision-1",
									},
								},
							},
						},
					},
				}))
				client := gitiles.NewClient(ctx)

				contents, err := client.DownloadDiff(ctx, "fake-host", "fake/project", "fake-revision-2", gitiles.PARENT, "other/fake/file")

				So(err, ShouldBeNil)
				So(contents, ShouldBeEmpty)
			})

			Convey("with diff against revision when called for modified path", func() {
				ctx := gitiles.UseGitilesClientFactory(ctx, Factory(map[string]*Host{
					"fake-host": {
						Projects: map[string]*Project{
							"fake/project": {
								Revisions: map[string]*Revision{
									"fake-revision-1": {
										Files: map[string]*string{
											"fake/file": strPtr("fake-contents-A"),
										},
									},
									"fake-revision-2": {
										Files: map[string]*string{
											"fake/file": strPtr("fake-contents-A"),
										},
										Parent: "fake-revision-1",
									},
									"fake-revision-3": {
										Files: map[string]*string{
											"fake/file": strPtr("fake-contents-B"),
										},
									},
								},
							},
						},
					},
				}))
				client := gitiles.NewClient(ctx)

				contents, err := client.DownloadDiff(ctx, "fake-host", "fake/project", "fake-revision-2", "fake-revision-3", "fake/file")

				So(err, ShouldBeNil)
				So(contents, ShouldNotBeEmpty)
			})

			Convey("with no diff against revision when called for unmodified path", func() {
				ctx := gitiles.UseGitilesClientFactory(ctx, Factory(map[string]*Host{
					"fake-host": {
						Projects: map[string]*Project{
							"fake/project": {
								Revisions: map[string]*Revision{
									"fake-revision-1": {
										Files: map[string]*string{
											"fake/file": strPtr("fake-contents-A"),
										},
									},
									"fake-revision-2": {
										Files: map[string]*string{
											"fake/file": strPtr("fake-contents-B"),
										},
										Parent: "fake-revision-1",
									},
									"fake-revision-3": {
										Files: map[string]*string{
											"fake/file": strPtr("fake-contents-B"),
										},
									},
								},
							},
						},
					},
				}))
				client := gitiles.NewClient(ctx)

				contents, err := client.DownloadDiff(ctx, "fake-host", "fake/project", "fake-revision-2", "fake-revision-3", "fake/file")

				So(err, ShouldBeNil)
				So(contents, ShouldBeEmpty)
			})

		})

	})
}

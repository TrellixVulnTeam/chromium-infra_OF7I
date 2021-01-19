// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package history

import (
	"bytes"
	"context"
	"testing"

	"golang.org/x/sync/errgroup"

	evalpb "infra/rts/presubmit/eval/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestPlayer(t *testing.T) {
	t.Parallel()
	Convey(`ReaderWriter`, t, func() {
		ctx := context.Background()
		buf := bytes.NewBuffer(nil)

		records := []*evalpb.Record{
			parseRecord(`test_duration {
				test_variant { id: "1" }
				duration: { seconds: 1 }
			}`),

			parseRecord(`rejection_fragment {
				rejection {
					patchsets {
						change { host: "gerrit.example.com", project: "project", number: 1}
						patchset: 1,
						changed_files {
							repo: "https://repo.example.com"
							path: "source_file1"
						}
						changed_files {
							repo: "https://repo.example.com"
							path: "source_file2"
						}
					}
					timestamp { seconds: 123 }
				}
			}`),
			parseRecord(`rejection_fragment {
				rejection {
					failed_test_variants {
						id: "test1"
						file_name: "test_file1"
					}
				}
			}`),

			parseRecord(`test_duration {
				test_variant { id: "2" }
				duration: { seconds: 2 }
			}`),

			parseRecord(`rejection_fragment {
				rejection {
					failed_test_variants {
						id: "test2"
						file_name: "test_file2"
					}
				}
				terminal: true
			}`),

			parseRecord(`test_duration {
				test_variant { id: "3" }
				duration: { seconds: 3 }
			}`),

			parseRecord(`rejection_fragment {
				rejection {
					patchsets {
						change { host: "gerrit.example.com", project: "project", number: 2}
						patchset: 2,
						changed_files {
							repo: "https://repo.example.com"
							path: "source_file3"
						}
					}
					timestamp { seconds: 123 }
					failed_test_variants {
						id: "test3"
						file_name: "test_file3"
					}
				}
				terminal: true
			}`),
		}

		// Write the records.
		w := NewWriter(buf)
		for _, r := range records {
			So(w.Write(r), ShouldBeNil)
		}
		So(w.Close(), ShouldBeNil)

		eg, ctx := errgroup.WithContext(ctx)

		p := NewPlayer(NewReader(buf))
		eg.Go(func() error {
			return p.Playback(ctx)
		})

		var rejections []*evalpb.Rejection
		eg.Go(func() error {
			for r := range p.RejectionC {
				rejections = append(rejections, r)
			}
			return nil
		})

		var durations []*evalpb.TestDuration
		eg.Go(func() error {
			for d := range p.DurationC {
				durations = append(durations, d)
			}
			return nil
		})

		So(eg.Wait(), ShouldBeNil)

		So(rejections, ShouldHaveLength, 2)
		So(rejections[0], ShouldResembleProtoText, `
			patchsets {
				change { host: "gerrit.example.com", project: "project", number: 1}
				patchset: 1,
				changed_files {
					repo: "https://repo.example.com"
					path: "source_file1"
				}
				changed_files {
					repo: "https://repo.example.com"
					path: "source_file2"
				}
			}
			timestamp { seconds: 123 }
			failed_test_variants {
				id: "test1"
				file_name: "test_file1"
			}
			failed_test_variants {
				id: "test2"
				file_name: "test_file2"
			}
		`)
		So(rejections[1], ShouldResembleProtoText, `
			patchsets {
				change { host: "gerrit.example.com", project: "project", number: 2}
				patchset: 2,
				changed_files {
					repo: "https://repo.example.com"
					path: "source_file3"
				}
			}
			timestamp { seconds: 123 }
			failed_test_variants {
				id: "test3"
				file_name: "test_file3"
			}
		`)

		So(durations, ShouldHaveLength, 3)
		So(durations[0], ShouldResembleProtoText, `
			test_variant { id: "1" }
			duration: { seconds: 1 }
		`)
		So(durations[1], ShouldResembleProtoText, `
			test_variant { id: "2" }
			duration: { seconds: 2 }
		`)
		So(durations[2], ShouldResembleProtoText, `
			test_variant { id: "3" }
			duration: { seconds: 3 }
		`)
	})
}

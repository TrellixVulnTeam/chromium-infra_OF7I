// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dirmd

import (
	"testing"

	dirmdpb "infra/tools/dirmd/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestFindInheritFrom(t *testing.T) {
	t.Parallel()

	Convey(`findInheritFrom`, t, func() {
		m := &Mapping{
			Dirs: map[string]*dirmdpb.Metadata{
				".": {},
				"no-inheritance": {
					InheritFrom: "-",
				},
				"a": {},

				"missing-leading-double-slash": {
					InheritFrom: "/x",
				},

				"from-a": {
					InheritFrom: "//a",
				},
				"from-a-b-c": {
					InheritFrom: "//a/b/c",
				},
			},
		}

		test := func(dir, expected string) {
			inheritFrom, inheritFromMD, err := m.findInheritFrom(dir, m.Dirs[dir])
			So(err, ShouldBeNil)
			So(inheritFrom, ShouldEqual, expected)
			So(inheritFromMD, ShouldEqual, m.Dirs[inheritFrom])
			return
		}

		Convey(`.`, func() {
			test(".", "")
		})

		Convey(`no-inheritance`, func() {
			test("no-inheritance", "")
		})

		Convey(`missing-leading-double-slash`, func() {
			_, _, err := m.findInheritFrom("missing-leading-double-slash", m.Dirs["missing-leading-double-slash"])
			So(err, ShouldErrLike, "unexpected inherit_from value")
		})

		Convey(`a`, func() {
			test("a", ".")
		})

		Convey(`a/b/c`, func() {
			test("a/b/c", "a")
		})

		Convey(`from-a`, func() {
			test("from-a", "a")
		})

		Convey(`from-a-b-c`, func() {
			test("from-a-b-c", "a")
		})

		Convey(`from-a-b-c/d/e`, func() {
			test("from-a-b-c/d/e", "from-a-b-c")
		})
	})
}

func TestComputeAll(t *testing.T) {
	t.Parallel()
	Convey(`ComputeAll`, t, func() {
		Convey(`Works`, func() {
			m := &Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": {
						TeamEmail: "team@example.com",
						// Will be inherited entirely.
						Wpt: &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},

						// Will be inherited partially.
						Monorail: &dirmdpb.Monorail{
							Project: "chromium",
						},
					},
					"a": {
						TeamEmail: "team-email@chromium.org",
						Monorail: &dirmdpb.Monorail{
							Component: "Component",
						},
					},
				},
			}

			err := m.ComputeAll()
			So(err, ShouldBeNil)

			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": m.Dirs["."], // did not change
					"a": {
						TeamEmail: "team-email@chromium.org",
						Wpt:       &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Component",
						},
					},
				},
			})
		})

		Convey(`Deep nesting`, func() {
			m := &Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".":   {TeamEmail: "team@example.com"},
					"a":   {},
					"a/b": {},
				},
			}
			err := m.ComputeAll()
			So(err, ShouldBeNil)
			So(m.Dirs["a/b"].TeamEmail, ShouldEqual, "team@example.com")
		})

		Convey(`No root`, func() {
			input := &Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a": {TeamEmail: "a"},
					"b": {TeamEmail: "b"},
				},
			}

			actual := input.Clone()
			err := actual.ComputeAll()
			So(err, ShouldBeNil)
			So(input.Proto(), ShouldResembleProto, input.Proto())
		})

		Convey(`InheritFrom`, func() {
			Convey(`Works`, func() {
				m := &Mapping{
					Dirs: map[string]*dirmdpb.Metadata{
						".":   {TeamEmail: "team@example.com"},
						"a":   {Os: dirmdpb.OS_LINUX},
						"x":   {InheritFrom: "//a/b"},
						"x/y": {},
					},
				}
				err := m.ComputeAll()
				So(err, ShouldBeNil)
				So(m.Dirs["x/y"].TeamEmail, ShouldEqual, "team@example.com")
				So(m.Dirs["x/y"].Os, ShouldEqual, dirmdpb.OS_LINUX)
			})

			Convey(`Cycle`, func() {
				m := &Mapping{
					Dirs: map[string]*dirmdpb.Metadata{
						"a": {InheritFrom: "//b"},
						"b": {InheritFrom: "//a"},
					},
				}
				err := m.ComputeAll()
				So(err, ShouldErrLike, "cycle")
			})
		})
	})
}

func TestReduce(t *testing.T) {
	t.Parallel()

	Convey(`Reduce`, t, func() {
		Convey(`Works`, func() {
			input := &Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": {
						TeamEmail: "team@example.com",
						Monorail: &dirmdpb.Monorail{
							Project: "chromium",
						},
					},
					"a": {
						TeamEmail: "team@example.com", // redundant
						Wpt:       &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium", // redundant
							Component: "Component",
						},
					},
				},
			}

			actual := input.Clone()
			err := actual.Reduce()
			So(err, ShouldBeNil)
			So(actual.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": input.Dirs["."], // did not change
					"a": {
						Wpt: &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},
						Monorail: &dirmdpb.Monorail{
							Component: "Component",
						},
					},
				},
			})
		})

		Convey(`Deep nesting`, func() {
			m := &Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".":   {TeamEmail: "team@example.com"},
					"a":   {TeamEmail: "team@example.com"},
					"a/b": {TeamEmail: "team@example.com"},
				},
			}
			err := m.Reduce()
			So(err, ShouldBeNil)
			So(m.Dirs, ShouldNotContainKey, "a")
			So(m.Dirs, ShouldNotContainKey, "a/b")
		})

		Convey(`Nothing to reduce`, func() {
			Convey(`Works`, func() {
				m := &Mapping{
					Dirs: map[string]*dirmdpb.Metadata{
						"a": {TeamEmail: "team@example.com"},
						"b": {TeamEmail: "team@example.com"},
					},
				}
				err := m.Reduce()
				So(err, ShouldBeNil)
				So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
					Dirs: map[string]*dirmdpb.Metadata{
						"a": {TeamEmail: "team@example.com"},
						"b": {TeamEmail: "team@example.com"},
					},
				})
			})

			Convey(`No inheritance`, func() {
				m := &Mapping{
					Dirs: map[string]*dirmdpb.Metadata{
						".": {TeamEmail: "team@example.com"},
						"a": {
							TeamEmail:   "team@example.com",
							InheritFrom: "-",
						},
					},
				}
				err := m.Reduce()
				So(err, ShouldBeNil)
				So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
					Dirs: map[string]*dirmdpb.Metadata{
						".": {TeamEmail: "team@example.com"},
						"a": {
							TeamEmail:   "team@example.com",
							InheritFrom: "-",
						},
					},
				})
			})
		})
	})
}

func TestMerge(t *testing.T) {
	Convey(`Merge`, t, func() {
		Convey(`wpt.notify false value is not overwritten/ignored`, func() {
			inherited := &dirmdpb.Metadata{
				Wpt: &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},
			}
			own := &dirmdpb.Metadata{
				Wpt: &dirmdpb.WPT{Notify: dirmdpb.Trinary_NO},
			}
			merge(inherited, own)
			So(inherited.Wpt.Notify, ShouldEqual, dirmdpb.Trinary_NO)
		})
	})
}

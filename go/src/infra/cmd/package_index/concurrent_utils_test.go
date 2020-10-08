package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSet(t *testing.T) {
	t.Parallel()
	Convey("Concurrent set", t, func() {
		cs := NewConcurrentSet(0)

		Convey("Adding to set", func() {
			Convey("Adding a new value should return true", func() {
				So(cs.Add("hello"), ShouldEqual, true)
				So(cs.Add("hello"), ShouldEqual, false)
			})
		})
	})
}

func TestMap(t *testing.T) {
	t.Parallel()
	Convey("FileHashMap", t, func() {
		m := NewFileHashMap()

		Convey("Getting nonexistent values", func() {
			hash, ok := m.Filehash("nonexistent")
			Convey("Filehash should not be fetched", func() {
				So(hash, ShouldEqual, "")
				So(ok, ShouldEqual, false)
			})

			fname, ok := m.Filename("nonexistent")
			Convey("Filename should not be fetched", func() {
				So(fname, ShouldEqual, "")
				So(ok, ShouldEqual, false)
			})
		})

		Convey("Adding and retrieving data", func() {
			Convey("Adding should return true if new and false if not", func() {
				So(m.Add("hello", "hash"), ShouldEqual, true)
				So(m.Add("bye", "hash"), ShouldEqual, false)

				fname, ok := m.Filename("hash")
				So(fname, ShouldEqual, "hello")
				So(ok, ShouldEqual, true)

				hash, ok := m.Filehash("hello")
				So(hash, ShouldEqual, "hash")
				So(ok, ShouldEqual, true)
			})
		})
	})
}

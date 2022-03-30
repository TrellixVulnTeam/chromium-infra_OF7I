// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPatchFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("patchFile", t, func() {

		Convey("returns patched contents for modified file", func() {
			contents := `0
1
2
3
5
5
6
7
8
`
			diff := `commit 16ad87ebb64dd08be38df602d8614b1fc3918ece (HEAD -> main)
Author: Author
Date:   Fri Jan 1 00:00:00 2021 -0700

	Test commit

diff --git a/test/bar b/test/bar
index 2550c5c..73b065a 100644
--- a/test/bar
+++ b/test/bar
@@ -1,4 +1,4 @@
 red
 blue
-green
+yellow
 
diff --git a/test/baz b/test/baz
index 8c159cc..78603df 100644
--- a/test/baz
+++ b/test/baz
@@ -2,7 +2,7 @@
 1
 2
 3
-5
+4
 5
 6
 7
diff --git a/test/foo b/test/foo
index 7b09d64..4dced17 100644
--- a/test/foo
+++ b/test/foo
@@ -1,3 +1,5 @@
 cat
+lion
 dog
+wolf
 
`

			newContents, err := patchFile(ctx, "test/baz", contents, diff)

			So(err, ShouldBeNil)
			So(newContents, ShouldEqual, `0
1
2
3
4
5
6
7
8
`)
		})

		Convey("fails with patch rejected tag if the patch doesn't apply", func() {
			contents := `0
1
2
3
5
5
6
7
8
`
			diff := `commit 16ad87ebb64dd08be38df602d8614b1fc3918ece (HEAD -> main)
Author: Author
Date:   Fri Jan 1 00:00:00 2021 -0700

	Test commit

diff --git a/test/baz b/test/baz
index 8c159cc..78603df 100644
--- a/test/baz
+++ b/test/baz
@@ -2,7 +2,7 @@
 1
 2
 3
-3
+4
 5
 6
 7
`

			newContents, err := patchFile(ctx, "test/baz", contents, diff)

			So(err, ShouldNotBeNil)
			So(PatchRejected.In(err), ShouldBeTrue)
			So(newContents, ShouldBeEmpty)
		})

	})
}

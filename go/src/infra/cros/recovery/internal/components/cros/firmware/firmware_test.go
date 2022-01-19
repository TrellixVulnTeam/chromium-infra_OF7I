//go:build linux
// +build linux

// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package firmware

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"infra/cros/recovery/logger"
)

func TestExtractECImage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := logger.NewLogger()
	board := "my-board"
	model := "my-model"
	tarballPath := "/some/folder/my_folder/tarbar.tr"
	Convey("Happy path", t, func() {
		runRequest := map[string]string{
			"mkdir -p /some/folder/my_folder/EC": "",
			"tar tf /some/folder/my_folder/tarbar.tr": `ec.bin
my-board/ec.bin`,
			"tar xf /some/folder/my_folder/tarbar.tr -C /some/folder/my_folder/EC ec.bin": "",
		}

		image, err := extractECImage(ctx, tarballPath, mockRunner(runRequest), logger, board, model)
		So(err, ShouldBeNil)
		So(image, ShouldEqual, "/some/folder/my_folder/EC/ec.bin")
	})
	Convey("Happy path with board file", t, func() {
		runRequest := map[string]string{
			"mkdir -p /some/folder/my_folder/EC": "",
			"tar tf /some/folder/my_folder/tarbar.tr": `my-ec.bin
my-board/ec.bin`,
			"tar xf /some/folder/my_folder/tarbar.tr -C /some/folder/my_folder/EC my-board/ec.bin": "",
		}

		image, err := extractECImage(ctx, tarballPath, mockRunner(runRequest), logger, board, model)
		So(err, ShouldBeNil)
		So(image, ShouldEqual, "/some/folder/my_folder/EC/my-board/ec.bin")
	})
	Convey("Happy path with board file with monitor", t, func() {
		runRequest := map[string]string{
			"mkdir -p /some/folder/my_folder/EC": "",
			"tar tf /some/folder/my_folder/tarbar.tr": `my-ec.bin
my-board/ec.bin
npcx_monitor.bin`,
			"tar xf /some/folder/my_folder/tarbar.tr -C /some/folder/my_folder/EC my-board/ec.bin":  "",
			"tar xf /some/folder/my_folder/tarbar.tr -C /some/folder/my_folder/EC npcx_monitor.bin": "",
		}

		image, err := extractECImage(ctx, tarballPath, mockRunner(runRequest), logger, board, model)
		So(err, ShouldBeNil)
		So(image, ShouldEqual, "/some/folder/my_folder/EC/my-board/ec.bin")
	})
}

func TestExtractAPImage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := logger.NewLogger()
	board := "my-board"
	model := "my-model"
	tarballPath := "/some/folder/my_folder/tarbar2.tr"
	Convey("Happy path", t, func() {
		runRequest := map[string]string{
			"mkdir -p /some/folder/my_folder/AP": "",
			"tar tf /some/folder/my_folder/tarbar2.tr": `image.bin
image-my-model.bin`,
			"tar xf /some/folder/my_folder/tarbar2.tr -C /some/folder/my_folder/AP image.bin": "",
		}

		image, err := extractAPImage(ctx, tarballPath, mockRunner(runRequest), logger, board, model)
		So(err, ShouldBeNil)
		So(image, ShouldEqual, "/some/folder/my_folder/AP/image.bin")
	})
	Convey("Happy path with board file", t, func() {
		runRequest := map[string]string{
			"mkdir -p /some/folder/my_folder/AP": "",
			"tar tf /some/folder/my_folder/tarbar2.tr": `image-my.bin
image-my-model.bin`,
			"tar xf /some/folder/my_folder/tarbar2.tr -C /some/folder/my_folder/AP image-my-model.bin": "",
		}

		image, err := extractAPImage(ctx, tarballPath, mockRunner(runRequest), logger, board, model)
		So(err, ShouldBeNil)
		So(image, ShouldEqual, "/some/folder/my_folder/AP/image-my-model.bin")
	})
}

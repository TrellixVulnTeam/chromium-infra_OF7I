// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/memlogger"
	"go.chromium.org/luci/common/logging/teelogger"
)

// TestBasic tests that log level is passing right.
func TestBasic(t *testing.T) {
	Convey(`A new instance`, t, func() {
		ml := logging.Get(memlogger.Use(context.Background())).(*memlogger.MemLogger)
		l := &loggerImpl{
			log: ml,
			// Expected that logger will increase it for +1
			callDepth: 2,
		}
		for _, entry := range []struct {
			L logging.Level
			F func(string, ...interface{})
			T string
		}{
			{logging.Debug, l.Debugf, "DEBU"},
			{logging.Info, l.Infof, "INFO"},
			{logging.Warning, l.Warningf, "WARN"},
			{logging.Error, l.Errorf, "ERRO"},
		} {
			Convey(fmt.Sprintf("Can log to %s", entry.L), func() {
				entry.F("%s", entry.T)
				So(len(ml.Messages()), ShouldEqual, 1)
				msg := ml.Get(entry.L, entry.T, map[string]interface{}(nil))
				So(msg, ShouldNotBeNil)
				So(msg.CallDepth, ShouldEqual, 3)
			})
		}
	})
}

// TestCreateFileLogger tests that file logger created with expected level and set level is matching.
func TestCreateFileLogger(t *testing.T) {
	Convey(`A new file logger`, t, func() {
		l := &loggerImpl{
			// Expected that logger will increase it for +1
			callDepth: 2,
			logDir:    "my_dir",
		}
		buf := new(bytes.Buffer)
		fc := func(ctx context.Context, dir, name string) (io.Writer, closer, error) {
			return buf, func() {}, nil
		}
		for _, entry := range []struct {
			L logging.Level
		}{
			{logging.Debug},
			{logging.Info},
			{logging.Warning},
			{logging.Error},
		} {
			Convey(fmt.Sprintf("Can create logger to %s", entry.L), func() {
				ctx := context.Background()
				var fs []teelogger.Filtered
				So(len(fs), ShouldEqual, 0)

				c, nfs, err := l.createFileLogger(ctx, fc, fs, entry.L)
				So(c, ShouldNotBeNil)
				So(nfs, ShouldNotBeNil)
				So(err, ShouldBeNil)
				So(len(nfs), ShouldEqual, 1)
				So(nfs[0].Factory, ShouldNotBeNil)
				So(logging.GetLevel(c), ShouldEqual, entry.L)
				So(nfs[0].Level, ShouldEqual, entry.L)
			})
		}
	})
}

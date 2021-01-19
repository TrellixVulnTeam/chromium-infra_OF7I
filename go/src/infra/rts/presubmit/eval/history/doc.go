// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package history implements serialization and deserilization of historical
// records used for RTS evaluation.
//
// RTS evaluation uses history files to emulate CQ behavior with a candidate
// selection strategy. Conceptually a history file is a sequence of Record
// protobuf messages,
// see https://source.chromium.org/chromium/infra/infra/+/master:go/src/infra/rts/presubmit/eval/proto/eval.proto.
// More specifically, it is a Zstd-compressed RecordIO-encoded sequence of
// Records.
//
// TODO(nodir): delete this package in favor of .jsonl.gz files.
package history

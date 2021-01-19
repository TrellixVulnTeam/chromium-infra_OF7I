// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package history

import (
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/data/recordio"

	evalpb "infra/rts/presubmit/eval/proto"
)

// Writer serializes historical records to an io.Writer.
type Writer struct {
	buf  []byte
	dst  io.Writer
	rio  recordio.Writer
	zstd *zstd.Encoder
}

// NewWriter creates a Writer.
func NewWriter(w io.Writer) *Writer {
	ret := &Writer{dst: w}

	var err error
	if ret.zstd, err = zstd.NewWriter(w); err != nil {
		panic(err) // we don't pass any options
	}

	ret.rio = recordio.NewWriter(ret.zstd)
	return ret
}

// CreateFile returns Writer that persists data to a new file.
// When done, call Close() on the returned Writer.
func CreateFile(path string) (*Writer, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return NewWriter(f), nil
}

// Write writes a historical record.
func (w *Writer) Write(rec *evalpb.Record) error {
	// Marshal the record reusing the buffer.
	marshalled, err := (&proto.MarshalOptions{}).MarshalAppend(w.buf, rec)
	if err != nil {
		return err
	}
	// If the buffer was too small, remember the new larger one.
	w.buf = marshalled[:0]

	if _, err := w.rio.Write(marshalled); err != nil {
		return err
	}
	return w.rio.Flush()
}

// Close flushes everything and closes the underlying io.Writer.
func (w *Writer) Close() error {
	if err := w.zstd.Close(); err != nil {
		return err
	}

	if closer, ok := w.dst.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

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

// Reader deserializes historical records from an io.Reader.
type Reader struct {
	buf  []byte
	src  io.Reader
	rio  recordio.Reader
	zstd *zstd.Decoder
}

// NewReader creates a Reader.
func NewReader(r io.Reader) *Reader {
	ret := &Reader{src: r}
	ret.zstd, _ = zstd.NewReader(r)               // cannot return error - no options
	ret.rio = recordio.NewReader(ret.zstd, 100e6) // max 100MB proto.
	return ret
}

// OpenFile creates a Reader that reads data from a file.
// When done, call Close() on the returned Reader.
func OpenFile(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return NewReader(f), nil
}

// Read reads the next historical record.
// Returns io.EOF if there is no record.
func (r *Reader) Read() (*evalpb.Record, error) {
	// Read the next frame into the buffer.
	size, frame, err := r.rio.ReadFrame()
	switch {
	case err != nil:
		return nil, err
	case cap(r.buf) < int(size):
		r.buf = make([]byte, size)
	default:
		r.buf = r.buf[:size]
	}
	if _, err := io.ReadFull(frame, r.buf); err != nil {
		return nil, err
	}

	// Unmrashal and return the record.
	rec := &evalpb.Record{}
	if err := proto.Unmarshal(r.buf, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// Close releases all resources and closes the underlying io.Reader.
func (r *Reader) Close() error {
	r.zstd.Close()

	if closer, ok := r.src.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

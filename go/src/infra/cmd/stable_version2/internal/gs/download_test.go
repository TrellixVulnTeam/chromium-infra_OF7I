// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gs

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"cloud.google.com/go/storage"

	"go.chromium.org/luci/common/gcloud/gs"
)

// TestDownloadByteRange tests that the a given byte range is transferred from
// the input file to the local output file.
func TestDownloadByteRange(t *testing.T) {
	t.Parallel()

	source, err := ioutil.TempFile("", "source")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.Write([]byte("1234567890abcdef")); err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	source, err = os.Open(source.Name())
	if err != nil {
		t.Fatal(err)
	}

	dest, err := ioutil.TempFile("", "dest")
	if err != nil {
		t.Fatal(err)
	}
	if err := dest.Close(); err != nil {
		t.Fatal(err)
	}
	os.Remove(dest.Name())
	destPrefix := dest.Name()

	defer os.Remove(source.Name())

	client := &Client{}
	client.C = &FakeClient{wrapped: source}
	if err := client.DownloadByteRange(gs.Path("a"), destPrefix, 0, 10); err != nil {
		t.Errorf("failed to download: %s", err.Error())
	}

	contents, err := ioutil.ReadFile(destPrefix)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff("1234567890", string(contents)); diff != "" {
		t.Errorf("unexpeced diff: %s", diff)
	}
}

type LimitedReadCloser struct {
	closeHandle io.Closer
	readHandle  *io.LimitedReader
}

func (c *LimitedReadCloser) Close() error {
	return c.closeHandle.Close()
}

func (c *LimitedReadCloser) Read(buf []byte) (int, error) {
	return c.readHandle.Read(buf)
}

func LimitReadCloser(readcloser io.ReadCloser, n int64) *LimitedReadCloser {
	out := &LimitedReadCloser{}
	out.closeHandle = readcloser
	out.readHandle = io.LimitReader(readcloser, n).(*io.LimitedReader)
	return out
}

type FakeClient struct {
	wrapped io.ReadCloser
}

func (f *FakeClient) Attrs(p gs.Path) (*storage.ObjectAttrs, error) {
	panic("Attrs")
}

func (f *FakeClient) NewReader(p gs.Path, offset int64, length int64) (io.ReadCloser, error) {
	if offset != 0 {
		panic("nonzero offsets unsupported")
	}
	return LimitReadCloser(f.wrapped, length), nil
}

func (f *FakeClient) NewWriter(p gs.Path) (gs.Writer, error) {
	panic("NewWriter")
}

func (f *FakeClient) Delete(p gs.Path) error {
	panic("Delete")
}

func (f *FakeClient) Rename(src gs.Path, dst gs.Path) error {
	panic("Rename")
}

func (f *FakeClient) Close() error {
	panic("Close")
}

// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"fmt"
	"io"

	"go.chromium.org/luci/common/data/text/indented"

	"infra/rts/presubmit/eval/history"
	evalpb "infra/rts/presubmit/eval/proto"
)

type printer struct {
	indented.Writer
	err error
}

func newPrinter(w io.Writer) *printer {
	return &printer{
		Writer: indented.Writer{
			Writer:    w,
			UseSpaces: true,
			Width:     2,
		},
	}
}

func (p *printer) printf(format string, args ...interface{}) {
	if p.err == nil {
		_, p.err = fmt.Fprintf(&p.Writer, format, args...)
	}
}

// psURL returns the patchset URL.
func psURL(p *evalpb.GerritPatchset) string {
	return fmt.Sprintf("https://%s/c/%d/%d", p.Change.Host, p.Change.Number, p.Patchset)
}

type historyFileInputFlag struct {
	path string
	ptr  **history.Player
}

func (f *historyFileInputFlag) Set(val string) error {
	r, err := history.OpenFile(val)
	if err != nil {
		return err
	}

	f.path = val
	*f.ptr = history.NewPlayer(r)
	return nil
}

func (f *historyFileInputFlag) String() string {
	return f.path
}

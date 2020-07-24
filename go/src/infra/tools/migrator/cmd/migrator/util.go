// Copyright 2020 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"io"
	"os"

	"go.chromium.org/luci/common/errors"
)

func ensureEmptyDirectory(ctx context.Context, path string) error {
	switch fil, err := os.Open(path); {
	case os.IsNotExist(err):
		return errors.Annotate(os.MkdirAll(path, 0777), "creating dir").Err()

	case err == nil:
		switch _, err := fil.Readdirnames(1); err {
		case nil:
			return errors.New("exists but is not empty")
		case io.EOF:
			return nil // exists and is empty
		default:
			return errors.Annotate(err, "reading directory entries").Err()
		}

	default:
		return errors.Annotate(err, "opening %q", path).Err()
	}
}

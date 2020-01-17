// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package querygs

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/luci/common/gcloud/gs"
)

const DONTCARE = "f7e8bdf6-f67c-4d63-aea3-46fa5e980403"

const exampleMetadataJSON = `
{
  "version": {
    "full": "R81-12835.0.0"
  },
  "results": [],
  "unibuild": true,
  "board-metadata": {
    "nami": {
      "models": {
        "sona": {
          "firmware-key-id": "SONA",
          "main-readonly-firmware-version": "Google_Nami.4.7.9",
          "ec-firmware-version": "nami_v1.2.3-4",
          "main-readwrite-firmware-version": "Google_Nami.42.43.44"
        },
        "akali360": {
          "firmware-key-id": "AKALI",
          "main-readonly-firmware-version": "Google_Nami.5.8.13",
          "ec-firmware-version": "nami_v1.2.3-4",
          "main-readwrite-firmware-version": "Google_Nami.52.53.54"
        }
      }
    }
  }
}
`

var testMaybeDownloadFileData = []struct {
	uuid     string
	metadata string
	out      map[string]map[string]string
}{
	{
		"f959c762-214e-4293-b655-032cd791a85f",
		exampleMetadataJSON,
		map[string]map[string]string{
			"nami": {
				"sona":     "Google_Nami.42.43.44",
				"akali360": "Google_Nami.52.53.54",
			},
		},
	},
}

func TestMaybeDownloadFile(t *testing.T) {
	t.Parallel()
	for _, tt := range testMaybeDownloadFileData {
		t.Run(tt.uuid, func(t *testing.T) {
			var r Reader
			r.dld = makeConstantDownloader(tt.metadata)
			e := r.maybeDownloadFile(DONTCARE, DONTCARE)
			if e != nil {
				msg := fmt.Sprintf("uuid (%s): unexpected error (%s)", tt.uuid, e.Error())
				t.Errorf(msg)
			}
			diff := cmp.Diff(tt.out, r.cache)
			if diff != "" {
				msg := fmt.Sprintf("uuid (%s): unexpected diff (%s)", tt.uuid, diff)
				t.Errorf(msg)
			}
		})
	}
}

func makeConstantDownloader(content string) downloader {
	return func(gsPath gs.Path) ([]byte, error) {
		return []byte(content), nil
	}
}

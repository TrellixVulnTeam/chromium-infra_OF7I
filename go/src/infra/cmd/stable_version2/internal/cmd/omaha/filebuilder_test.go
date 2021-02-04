// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package omaha

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"

	"infra/cmd/stable_version2/internal/utils"
)

type errpred func(e error) bool

func noError(e error) bool {
	return e == nil
}

func hasSubstr(msg string) errpred {
	return func(e error) bool {
		if e == nil {
			return false
		}
		return strings.Contains(e.Error(), msg)
	}
}

func TestFileBuilder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		oldSV *sv.StableVersions
		newSV []*sv.StableCrosVersion
		fv    FirmwareVersionFunc
		out   *sv.StableVersions
		f     errpred
	}{
		{
			name: "update single CrOS version",
			oldSV: &sv.StableVersions{
				Cros: []*sv.StableCrosVersion{
					utils.MakeSpecificCrOSSV(
						"nami",
						"vayne",
						"R77-12371.52.22",
					),
				},
				Firmware: []*sv.StableFirmwareVersion{
					utils.MakeSpecificFirmwareVersion(
						"nami",
						"vayne",
						"Google_Hana.8438.184.0",
					),
				},
			},
			newSV: []*sv.StableCrosVersion{
				utils.MakeSpecificCrOSSV(
					"nami",
					"vayne",
					"R77-12371.99.22",
				),
			},
			fv: func(ctx context.Context, board string, version string) ([]*sv.StableFirmwareVersion, error) {
				if board == "nami" && version == "R77-12371.99.22" {
					return []*sv.StableFirmwareVersion{
						utils.MakeSpecificFirmwareVersion(
							"nami",
							"vayne",
							"Google_Hana.8438.200.0",
						),
					}, nil
				}
				return []*sv.StableFirmwareVersion{
					utils.MakeSpecificFirmwareVersion(
						"nami",
						"vayne",
						"Google_Hana.8438.184.0",
					),
				}, nil
			},
			out: &sv.StableVersions{
				Cros: []*sv.StableCrosVersion{
					utils.MakeSpecificCrOSSV(
						"nami",
						"vayne",
						"R77-12371.99.22",
					),
				},
				Firmware: []*sv.StableFirmwareVersion{
					utils.MakeSpecificFirmwareVersion(
						"nami",
						"vayne",
						"Google_Hana.8438.200.0",
					),
				},
			},
			f: noError,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bg := context.Background()

			sv, e := FileBuilder(
				bg,
				tt.oldSV,
				tt.newSV,
				tt.fv,
			)

			diff := cmp.Diff(tt.out, sv, protocmp.Transform())
			if diff != "" {
				msg := fmt.Sprintf("name (%s): unexpected diff (%s)", tt.name, diff)
				t.Errorf(msg)
			}

			if !tt.f(e) {
				t.Errorf("unexpected error: %#v", e)
			}
		})
	}
}

func TestGetCrosFirmwareVersion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		oldSV      *sv.StableVersions
		board      string
		versionMap *boardVersionMap
		fvFunc     FirmwareVersionFunc
		outCros    []*sv.StableCrosVersion
		outFw      []*sv.StableFirmwareVersion
		f          errpred
	}{
		{
			name: "two versions",
			oldSV: &sv.StableVersions{
				Cros: []*sv.StableCrosVersion{
					utils.MakeSpecificCrOSSV(
						"nami",
						"vayne",
						"R77-12371.52.22",
					),
				},
				Firmware: []*sv.StableFirmwareVersion{
					utils.MakeSpecificFirmwareVersion(
						"nami",
						"vayne",
						"Google_Hana.8438.184.0",
					),
				},
			},
			board: "nami",
			versionMap: &boardVersionMap{
				oldBoardVersion: "R77-12371.52.22",
				omahaVersion:    "R77-12371.52.22",
			},
			fvFunc: func(ctx context.Context, board string, version string) ([]*sv.StableFirmwareVersion, error) {
				return []*sv.StableFirmwareVersion{
					utils.MakeSpecificFirmwareVersion(
						"nami",
						"vayne",
						"Google_Hana.8438.184.0",
					),
				}, nil
			},
			outCros: []*sv.StableCrosVersion{
				utils.MakeSpecificCrOSSV(
					"nami",
					"vayne",
					"R77-12371.52.22",
				),
			},
			outFw: []*sv.StableFirmwareVersion{
				utils.MakeSpecificFirmwareVersion(
					"nami",
					"vayne",
					"Google_Hana.8438.184.0",
				),
			},
			f: noError,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bg := context.Background()
			c, f, e := getCrosFirmwareVersion(
				bg,
				tt.oldSV,
				tt.board,
				tt.versionMap,
				tt.fvFunc,
			)

			diff := cmp.Diff(tt.outCros, c, protocmp.Transform())
			if diff != "" {
				msg := fmt.Sprintf("name (%s): unexpected diff (%s)", tt.name, diff)
				t.Errorf(msg)
			}

			diff = cmp.Diff(tt.outFw, f, protocmp.Transform())
			if diff != "" {
				msg := fmt.Sprintf("name (%s): unexpected diff (%s)", tt.name, diff)
				t.Errorf(msg)
			}

			if !tt.f(e) {
				t.Errorf("unexpected error: %#v", e)
			}
		})
	}

}

func TestBestVersion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		in    *boardVersionMap
		model string
		out   string
		f     errpred
	}{
		{
			name: "both versions same",
			in: &boardVersionMap{
				omahaVersion:    "R85-13310.41.0",
				oldBoardVersion: "R85-13310.41.0",
			},
			model: "nami",
			out:   "R85-13310.41.0",
			f:     noError,
		},
		{
			name: "omaha verion newer",
			in: &boardVersionMap{
				omahaVersion:    "R85-13310.48.0",
				oldBoardVersion: "R85-13310.41.0",
			},
			model: "nami",
			out:   "R85-13310.48.0",
			f:     noError,
		},
		{
			name: "so-called old verion newer",
			in: &boardVersionMap{
				omahaVersion:    "R85-13310.41.0",
				oldBoardVersion: "R85-13310.91.0",
			},
			model: "nami",
			out:   "R85-13310.91.0",
			f:     noError,
		},
		{
			name: "new version in map",
			in: &boardVersionMap{
				omahaVersion: "R85-13310.41.0",
				oldModelMap:  map[string]string{"nami": "R85-13310.97.0"},
			},
			model: "nami",
			out:   "R85-13310.97.0",
			f:     noError,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out, e := tt.in.bestVersion(tt.model)

			diff := cmp.Diff(tt.out, out)
			if diff != "" {
				msg := fmt.Sprintf("name (%s): unexpected diff (%s)", tt.name, diff)
				t.Errorf(msg)
			}

			if !tt.f(e) {
				t.Errorf("unexpected error: %#v", e)
			}
		})
	}
}

func TestAllCrosVersions(t *testing.T) {

	cases := []struct {
		name string
		in   *boardVersionMap
		out  map[string]bool
	}{
		{
			"empty",
			&boardVersionMap{},
			map[string]bool{},
		},
		{
			"just omaha version",
			&boardVersionMap{
				omahaVersion: "a",
			},
			map[string]bool{"a": true},
		},
		{
			"just old version",
			&boardVersionMap{
				oldBoardVersion: "a",
			},
			map[string]bool{"a": true},
		},
		{
			"duplicate omaha old board version",
			&boardVersionMap{
				omahaVersion:    "a",
				oldBoardVersion: "a",
			},
			map[string]bool{"a": true},
		},
		{
			"singleton oldBoardVersion",
			&boardVersionMap{
				oldModelMap: map[string]string{
					"k": "v",
				},
			},
			map[string]bool{"v": true},
		},
		{
			"duplicate oldBoardVersion",
			&boardVersionMap{
				oldModelMap: map[string]string{
					"a": "v",
					"b": "v",
				},
			},
			map[string]bool{"v": true},
		},
	}

	t.Parallel()
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := tt.in.allCrosVersions()

			diff := cmp.Diff(tt.out, out)
			if diff != "" {
				msg := fmt.Sprintf("name (%s): unexpected diff (%s)", tt.name, diff)
				t.Errorf(msg)
			}
		})
	}
}

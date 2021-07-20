// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package gerrit

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"infra/cros/internal/assert"

	"github.com/golang/mock/gomock"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/common/proto/gitiles/mock_gitiles"
)

func testDownloadFileFromGitilesSetUp(t *testing.T) *Client {
	// Mock Gitiles controller
	ctl := gomock.NewController(t)
	t.Cleanup(ctl.Finish)
	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)

	// Mock buildspec file request.
	req := &gitilespb.DownloadFileRequest{
		Project:    "chromeos/foo",
		Path:       "bar.xml",
		Committish: "HEAD",
	}
	fileData := "foobar!"
	gitilesMock.EXPECT().DownloadFile(gomock.Any(), DownloadFileRequestEq(req)).Return(
		&gitilespb.DownloadFileResponse{
			Contents: fileData,
		},
		nil,
	)
	mockMap := map[string]gitilespb.GitilesClient{
		"host": gitilesMock,
	}
	return NewTestClient(mockMap)
}

func TestDownloadFileFromGitiles(t *testing.T) {
	gc := testDownloadFileFromGitilesSetUp(t)
	contents, err := gc.DownloadFileFromGitiles(context.Background(), "host", "chromeos/foo", "HEAD", "bar.xml")
	assert.NilError(t, err)
	assert.StringsEqual(t, contents, "foobar!")
}

func TestDownloadFileFromGitilesToPath(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "TestDownlaodFileFromGitilesToPath")
	assert.NilError(t, err)

	outputPath := filepath.Join(tmpDir, "output")

	gc := testDownloadFileFromGitilesSetUp(t)
	err = gc.DownloadFileFromGitilesToPath(context.Background(), "host", "chromeos/foo", "HEAD", "bar.xml", outputPath)
	assert.NilError(t, err)
	got, err := ioutil.ReadFile(outputPath)
	assert.NilError(t, err)
	assert.StringsEqual(t, "foobar!", string(got))
}

func TestFetchFilesFromGitiles_success(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	// This is a base64-encoded .tar.gz file. It contains one file, contents pair:
	// dir/file1, This is a gzipped file!
	base64Enc := `H4sIADj/sFwAA+2VQU7DMBBFs+4phgu0HntmTBfds+yCCxjFtJYIjZxGKpweh6pBhYJYYBCtnyxF
iiP5O877WbrdjXe1j7M6xCoPSinLDJV9BSrNlrWB4f4eRkBDTGhFUIFCEdYV7DLlOaLvti6mKJ13
j+4uurVrTj7XxtC4+LSKm749Nb/fCYzXfwIS9KFeaJkrvp5oC802NH6BzIzKIOqpzA1pPfnroIUs
JOtnudc4+I8KmZT+6H/y5dh/spT859zBBi7c/+H8l+M/4D48ePzxNdL7EKJv9b8loXT+VtiW/v8N
vu7/pOsUtTZkSv+fJ4P/eax/4+D/p/1v1Dv/WUQqUBkzjVy4/7fr0EEaDlbPoW19DcPXcFV0LxQK
hTPnBcGXkjUAEgAA
`
	encodedZip, err := base64.StdEncoding.DecodeString(base64Enc)
	if err != nil {
		t.Error(err)
	}

	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)
	gitilesMock.EXPECT().Archive(gomock.Any(), gomock.Any()).Return(
		&gitilespb.ArchiveResponse{
			Contents: encodedZip,
		},
		nil,
	)
	host := "limited-review.googlesource.com"
	project := "chromiumos/for/the/win"
	ref := "main"
	paths := []string{"dir/file1"}
	mockMap := map[string]gitilespb.GitilesClient{
		host: gitilesMock,
	}
	gc := NewTestClient(mockMap)

	m, err := gc.FetchFilesFromGitiles(context.Background(), host, project, ref, paths)
	if err != nil {
		t.Error(err)
	}

	v, found := (*m)["dir/file1"]
	if !found {
		t.Error("Expected file not found in archive")
	}
	if v != "This is a gzipped file!\n" {
		t.Error("Archive not unzipped correctly")
	}
}

func TestFetchFilesFromGitilesWithLinks_success(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	// This is a base64-encoded .tar.gz file. It contains three file, contents pair:
	// dir/file1, This is a normal file!
	// dir/file2, This is a normal file!
	// dir/file3 -- symlink to dir/file2.
	base64Enc := `H4sIAAAAAAAAA+3TSw6CMBgE4K49RT0BffJfxAs0glp5aIosvL2UIK40YVGMcb6wApJOGKbwIWOJ
iQFZwWRurFSSCWXJKj3enzCpiawxQsXnUguhGbepg0V9d3OBc3Z2+6ot++7de9fgGxfux3Dpr2vk
Wkkx9H/wdakTnjH2T7Sgf5PbnHEVg6mEwSL0n6X+zLHg3HzYv6K5f2Mp7p+MYlwkzDT78/53J9/x
4XK89m1VFjz+DNvNt2PBSp77lwnPWLR/M+1fYP9reO2/vYTG1dg/AAAAAAAAAAAAAAAAAMBvegDt
yZyOACgAAA==
`
	encodedZip, err := base64.StdEncoding.DecodeString(base64Enc)
	if err != nil {
		t.Error(err)
	}

	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)
	gitilesMock.EXPECT().Archive(gomock.Any(), gomock.Any()).Return(
		&gitilespb.ArchiveResponse{
			Contents: encodedZip,
		},
		nil,
	)
	host := "limited-review.googlesource.com"
	project := "chromiumos/for/the/win"
	ref := "main"
	paths := []string{"dir/file1", "dir/file3"}
	mockMap := map[string]gitilespb.GitilesClient{
		host: gitilesMock,
	}
	gc := NewTestClient(mockMap)

	m, err := gc.FetchFilesFromGitiles(context.Background(), host, project, ref, paths)
	if err != nil {
		t.Error(err)
	}

	v, found := (*m)["dir/file1"]
	if !found {
		t.Error("Expected file not found in archive")
	}
	if v != "This is a normal file!\n" {
		t.Error("Archive not unzipped correctly for dir/file1")
	}

	v, found = (*m)["dir/file3"]
	if !found {
		t.Error("Expected file not found in archive")
	}
	if v != "This is a linked file!\n" {
		t.Error("Archive not unzipped correctly for dir/file3")
	}
}

func TestBranches(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	request := &gitilespb.RefsRequest{
		Project:  "my-project",
		RefsPath: "refs/heads",
	}
	response := make(map[string]string)
	response["refs/heads/foo"] = "deadbeef"
	response["refs/heads/bar"] = "beefcafe"

	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)
	gitilesMock.EXPECT().Refs(gomock.Any(), RefsRequestEq(request)).Return(
		&gitilespb.RefsResponse{
			Revisions: response,
		},
		nil,
	)
	host := "limited-review.googlesource.com"
	project := "my-project"

	mockMap := map[string]gitilespb.GitilesClient{
		host: gitilesMock,
	}
	gc := NewTestClient(mockMap)
	m, err := gc.Branches(context.Background(), host, project)

	assert.NilError(t, err)
	assert.StringsEqual(t, m["refs/heads/foo"], "deadbeef")
	assert.StringsEqual(t, m["refs/heads/bar"], "beefcafe")
}
func TestProjects(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	projects := []string{"foo", "bar"}

	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)
	gitilesMock.EXPECT().Projects(gomock.Any(), gomock.Any()).Return(
		&gitilespb.ProjectsResponse{
			Projects: projects,
		},
		nil,
	)
	host := "limited-review.googlesource.com"
	mockMap := map[string]gitilespb.GitilesClient{
		host: gitilesMock,
	}
	gc := NewTestClient(mockMap)
	got, err := gc.Projects(context.Background(), host)

	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(got, projects))
}

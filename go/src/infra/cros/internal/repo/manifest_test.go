// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package repo

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/gerrit"
	"infra/cros/internal/util"

	"github.com/golang/mock/gomock"
	bbproto "go.chromium.org/luci/buildbucket/proto"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/common/proto/gitiles/mock_gitiles"
)

func TestFetchFilesFromGitiles_success(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	// This is a base64-encoded .tar.gz file containing a snapshot of external full.xml and
	// _remotes.xml. I apologize to your eyes. This is a good representation of what comes back from
	// the Gitiles Archive call though.
	base64Enc := `H4sIAAAAAAAA/+1dW5PbuLH2s34FM3nYqolJ3UfSlu3E5XXO2ap47ePL2cqTDIKgBIskGIDUWPn1aYAX3SCyOZ6xU8lwa7wjsfvrbqDR6AZBzNvff3v9/sOTB70Gw8HgZjJxBsV1+v/BcDpzhuPJbDCdzqaTqQP005vZE2fwsGoVV64yIp8MvlnWqXH3oNr3uOiXmAYiIVHwF7qWIuZ57Am56kUkFkn2RSRMHd9IWJCs8h1L/rISYhUxj4q4p8gmYjw5ptySlDN5/N2PNvfxOrnev375y5vXXhw8oIyW8T+aTAYn4384HIwfx//3uK6vr51EZKx3ff3rm3dv3398+dvH6+ufnVdrkqyYcjLhZGuuHMlS4cTQWA6JlHB85sQkYA5P4DZzUsm3JNPfJSFTWc9QP1tnWap+7vdNBGAuTzImIdJ4ReBQIpfUhI+SQKg+8HMNUNO+8Hov80zEAB44apdQkJzdMpY4Z6QOSYL6WwdUBrOcHcscBT95CkBJ8NRoS0vbjDnX2p5rbZCGBynGYm1ZKEmfiiTkKyfkEetVWi61dcrTfuM5v/NsLfLMNFKBbmy/5VHkJGzLpEZOOd0Acp46/s559X8eNPZ1r/dH570m/R8p8lT1zO+gtYbIhIjgF5I5t8zJlf5CW0ZWzIFGcyJBwVy6ZnQDopXn/Jo5KZGKGe5e3QYqZZSHHCSDWI37OWAhyaPM+xpHn41RnvMRvk9IDMihoam5KUlA957YghGSB9DkueLJyvnsxp89520C7SjyKCj8wSjj6IYgjsp93eSneIa7V7fQ55WxG9RgJMul0QTMV4xmXCROIGgesyQrGoTHqZAZSTKn4Coah0jWA7trUaV1TjXjCFVL96C5/+jEPOER2YGaPSPMgFWNrm/GeXygvuln6EUWFG4RCLAuzKFn/ZyD5UDyyvhE7+0H0wmBYCr5CZwvoVEOo4MAaalbgQUgCaNMKSJ5BH2i20XfffsBTNTd64NqGuUIAjwMfGCvB5VMDzboH07iFStsMyppx1E93aWXrUiZDIWM4fuIEcAlpsFBogd9kJJdJEjgrFjCJNE3nvZYRj3nkyrcqOJiyYoDpgSn//juDfyrR9+vesyAVSR2Yhb7cNdzfoPgUoQQ09o9bloInBoaEoZJMXpI7c1a47OhXVgYEb+0T/9rswj4QSlN6JBA96fKtBHbvZER3zDnl08fIUK50LQwkIy3CWmYFJN6xBZDTXuf9x+fsEAwi6GHlA4JDyWjLf+fDU7n/9HwZvA4/3+P69mfoeMd8HoF4+D51dAbXMHYpiKAYP386tPHv7rzqz+/6D2rxuSLnuM8K3zGMfPG8ysqhbqCr+srhIixfn51NP3rAuB03j9ikmzL2e05l1vcOGd2+jZdSq476dM/4iIRJ6q0ziqLCJU2yYGIKAUPupldMjVa/ay/741v7f+DjOA+3Ml6tY3/wdn4H08mkyfOSM+1D6rZk//68f89mri5/6c3k+notP4bjyaP8f97XHeK/1VyWsTBwxSiipRVKq7DWIEsWaj6a0YCXeUpyOtOwqCGqKKtrsTcL8+v5hUclHKcshe9v+vyp4iGylkTSOx8XQtq+p+gPsypzq21S+88zdav+PSHVIovUFxAiptBkFaS9nVmvBZiAxKPZw+oRPf3DtUsqo/nV/tC4mnIZXwLlcjTfQL+tMpU9/NGKlyDBnm9WypyQRy0PvEjFrgRZK/Pr1JIVPNU5+QG7NwQKJRs+hdfF+KPGUxhu9xymeUkYsn2LgZWpbBr0uza3qdqp2IoGQK2LVLpI+wzHc80sepbMGTMYmR96yFNcF4Y9GdUpDtdMjvgOs+vXn76+L9v33+4gqJTd1P9sW+l/tuvr17/9uF1RV1/LOztlwabD39w3Y9mjaIwzYF7pmpkgSkffV1KltUZVD+3a1asMpSLNMVo0yVtscgRAJwNtk9S3od8IgUVl6ZVlvDNshCmqOSpWULJVV35KuEdQsHNlCcwBlUxBPUA3gIxFGJFNS10/fRUF7TFYohRnCcQbTODCf0RFIU3S4KySi67saiuQR+nbJinxk7nlv0kmVnWiclGL0VUCzlKOP/IOd3AsC91tLtQ4XF9YyXCNUvCF4eU2vp9UJuOhjQcTn1KZsFwwILFYjwPpmQ8JVN/SG4WwTichQtCwYkKxQCAJGBBUXYWYn1JErp2YxGAj29JlMN3mcis3mEZyn2fBEsaLQOWwR2BGXNnLJcDRT/I45gz1S98LnIVZKi++IqRcokVIS1mcsXuIuuYsUES5TBvKZij1JJIuuan0aoKI8dxAqOKDblJkWjJvgozoHGdd8zQjKyHW66WupZAReNznqbOghlKxzi1zNOAnE7plzrqjKlBQiRWKhN0gwGuaZvwcgpZDErPirQBTTLKU2bPELq7TYX2UGFCbXYwyS15soXALOQOofY+kUHof4bf0HIqh2i8VHTNgjzC9ccpSwO6XlZdkoREu3/isI8ZGrOQPky0MljqZeddP92xKDxvIFwG0qwYQkxDbu0OvNGogyEKdANtVx6NOPSe1ZrLuVNtXJ1E7S23WLkfX/1CnJvuLhg2nvijwXgwpTOYTJk/X7Ah8YMpG5EFlIvzwF8sFuGALOy1RLUWZG0HXQDoFZw++1osMvcj7heGWXQ2lGlEMr3QbGNBytAN84XwyCLiHN1KXFlnVp+apUIqm7FYK6kLpFarSnIqJKt5UAKKQL4sHglgxVxmqhzuIAW/pAWknBL8rGUsVVR9X0DS6VqZGsbsZfGVTSNLbXJwr40dejxlkuuEmdg8w4J6zNIugSjFMludeEbRDpUHXJf7WSPanqgV0CcZOPwOl0Rc5ELI4frhorvhtqBkFbFnQKDHqR9tmtqkomiHEiKjBGa6RrQ9UTtgpiAxYo09tqdphaNh7MLg5JRBNpJwk4VeBrZRt4tYA17ERNKIvCdCAFb8bunpOB8450PIEr4fNXZeRYGAggEOAVK6q1w/EUVkDjUL/CpBbTsEQjQw6xGcZ7xNJIIHmRPh1NrGjY1bUrRCBSzN1uBEUDFada0nAJzxh3AY4dsmI+C2q1vxLktbiIlzL0fGru6xxrnhgKh9LqP30ZasIfWoiUJCzwqZy5AVNRp3aVarotb6pIEPIcvWLndsNHzC0g+h2mwOrhVFK9RKj3fJsCGiJkcgRz6DkrdJy5qkHUy4MJvrJc9GvAOqdkhJ0vXOujK9v9WKAmM5kyQOG0fgARECMM0zHfWliJoxD+laYb8QX6/WiyhA9vQBQzs6FBxs2xIT9zStcDGHWcC4RkzkBuubp1wYOQlf+Y3TUU3SDiZ8qKAbsUoKBBSUWc1IhqAdKCsn5gaoDDt3J+zWj/LGzKgmaQVL82RXDH5c5+7p27Ez0lzeFPdRMOXEidPxgAGBLlmcRm2Li+fkCGSR03XHeuyYBynDF83NXNMgVa5mwKUpPe9hSrXBIpVJSdCt74+ZEFLSeBnZH4xaaFrhcl1RuuYx7GW8A6JWQEhAebZrzCtjtyJqR9OSl5KFTMIgbs6X0KusF9ER+mxak5OapB0sDRqBUswUesv8NGoeUDVJK9hXIVe6dAyRDbinv4hdPFZunEp06XbX2vF414VdhcM1bxKsWzKZM+pLlh0RFhv4lnEWICpnNOd+TbxafL/c0EegkroxUwSvxhm19TmDxh3OvZE3cFPJhlhloI6UucpY4OICcSs3Tm62ZqebVJtFlQxIdFCKZi0rt1YOHH6emaXUvnls2vZspn4Gg9yEY5OE1AuwXVSScJELJSeEpEimtucXF0RUDCh0KCo4b9kNYqFHYetM9p946HNy69Crn75NvckEqQjMblB2+9+QElnhcMIlIxtIbLALBGbLs1tzlTL+4LrmXaVIqWpbUPHSCAvMe0cw1TIQb3YUVbzFm0bFVqWkfrWrUqPc9OgEjEZgflC9zFZz//L63YfyLSnnk/HY8n04kjnF3q61fgOqwjM7nMIIFAMsiB9Mvy+zhqgO7NZtSZcaqn/mo8oWuyCXz7+6UMZREkWuylP9jtQF9yHzhR/OFoNxOB7eLNiNT2f+nIV0OCCzMZlNZ8OA6b1M2B4VJDjdyd7sMRUHCp9qBzMvRN2Twx4A4hTYr/qXTwvLTan0+RXE7hbJ9fNGCww2rzndQtqir5BMm3hfzWWDa45Fo8HwxhtMu6nbH8ugCn3Ct09td9e+BMVpJM1uBpJmHTKlQyaklF2aiW9ZILeh4USDkA7Qp9T2zgcqdwT/zXE6BISrXSBWeD1qDhS8nyuX/ulPHeANRxrlSv8gjdA8Km156mDlwOFzUIbsILhvuidXNmaMUD0DKnuQuhzdGpjuL7KxYDO6p/FyBtUa0WbeAOnaLB4OBvelp8HCpcbQA+uYpPckuYbDC5cnWcA3CpfYFCFUaZ/GkenUfrE1cvhj/AS0cCsF8Kon0Q90bZCOV3UVbX6gqqumHTOnqnL6I1sVpCNVvYXY3GFgnJFb9TBk7tAbecgsbB3Qy3tXcTt/rXA44XkQRLvuM9wJH0rWhsmERf3t2JvjBRVMHbpfo3dUZzh4UH2GyFnkQKHJwyqEXK44UOhhe2zYscsm3kM2kEbvqM6Ddtikc4dpDjc2x250WA69k2K1mO4Krm4fWjeQ0FWtxcPqtEAqBCxJl5XyigGHvo31FNEBvWTAruSS1Frl6q/r213er0n3j8sb3q+xKEJdxNaECzxYYwPZYQGupMdixz7vBm4YsOj/iHkncE2PxpZZh3SiYsCiK7Po0gm/ZEFL4JCCtbzw1s7SGA866hIwqsdEd40sjPej13YSpVG+4kmHQHXMhpSkF9i71w0nfDhZYsUzBvGnc0Z8xomTl6UdRJwQty6cTAdD7IwTE7llUYc5oWLAoXd/9I3GdQPmc9Ky/ar7k/USFa9GKBkLJEs6PMTEaVL36F4CXised1jnxelT4+K04LHoPpqOuHByRJAu/RtbIn7hrcGaA4cfU7fjTpI9C9YCFheH3kk3ATU7SDpjRUlMQuWuSMLUusPovMRk9ZX/H3ljV2Wk8TWbI/SMd9pkYKG3KzL0xqAKRW8nMSdd6jMMCuUbFarX4huY7m8tvhSy7JbDX+RqmUWg2SbIOUQyEmXsDs9LThlx0oTI9C50vJSSAYWuGPG5uK9noTa01tkbysWhNxgj9TWpnkujbin9BbZ7yA8VV90d4ZAJJ0XkSeCKlCV3SBBtzHeVinklEAeB02Cn7tM9SzSU6CxSuoPw6BUDDh3agK4JT7pOtqeMOGnptzw2O9zejZbnsjiP2s+YuciFlXNfT18MFE4oyFBMdukzG0fzeRpjb4iMiLlrdp+c79ZsVKng6hCotxCp594AOUVuuVxFkMIHTHaJi8dsKEm3zJcZdUnaYQHpgAcnYw3JTLKybCW3UqEgU7LUe+giTknSIQNeC8i5kE/Mj2VAinPTWU6rf1hkIP1Dn8Ln5sp3U2kO3u7kJWfMqAb5GqpLr8jU1dKKZ16x4q3/gkgfDP1a/BhOt0qs7MJuGdma00zMLxYxJwQVkt5t+qo0tjhBXR9cwJnyDMTvzMnkzpzFXh98X5yznuuhog/9qzf/9ctz8/XWUgUynSIJd4hyiAGr5VRnoHuOw7yVp1E8sN65Nefs+8wh2TGy/lAotD+az3lXHNC+YSx1AtApM5l3cVwfL87KNOeumz8lUBy1Vxw9BfA/G7D6oGTlVaJM09dyDVf1PkdfRboJC4wlFIQ5xL36aEZ90KCX7g6VPO6lGtRo2T/QuMH/Woj3A2M+IEMa3tz48zEdLyZ0MRz6w8l4OBrNJv7oJvDJfD4nbHDCfnKAELacajgC9Nw1j3oR9cJSE2VXne9D1T6NSLJa6kyEZKU7NKkfRdu4Ov20T0PWP0BxC5TvY9ORPacPMhoXRrXLKRsZ/tCpi8Lf7bK1SNw3RG4Ccdu0rtiJ9a6amb6Jg6Uvxa1qnA3a6a06VAHrZfHe03GYvbxj/vhALV8FPLS9pHa+7GYh7XCYVp3m932eXCoRWogOXuKi7GZBIRb5IZ3cBGzKhjQIJovROJzPRsMpHbDAn88n367titoWtS7fP3hngN4MR3MGEXRGpqNwMBzOFrNwPCZ0zGbBzWg2Jovxzfx0axJax6NT1L7aTvm0H7h2Qnp3icS3rVlcEnpKfUe5eXLLk2CpYyFW+CWWu2iQ5mFofWX8XLKFtPvhcxu2sxz3fSKykfZMZhU33pRHkiv9h4DMscCQIScBkUH1gtClMFIdZm6pH/a3tKT7/EsLj9fj9Xg9Xo/X4/Xvc/0LEpszawB4AAA=`
	// Override because this is the name of the file in the above base64 gibberish.
	rootXML = "full.xml"
	encodedZip, err := base64.StdEncoding.DecodeString(base64Enc)
	if err != nil {
		t.Error(err)
	}

	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)
	gitilesMock.EXPECT().Archive(gomock.Any(), gomock.Any()).Times(2).Return(
		&gitilespb.ArchiveResponse{
			Contents: encodedZip,
		},
		nil,
	)
	gerrit.MockGitiles = gitilesMock
	gitilesCommit := &bbproto.GitilesCommit{
		Host:    "chrome-internal.googlesource.com",
		Project: "chromeos/manifest-internal",
		Id:      "snapshot",
	}

	m, err := GetRepoToRemoteBranchToSourceRootFromManifests(context.Background(), http.DefaultClient, gitilesCommit)
	if err != nil {
		t.Error(err)
	}
	if len(m) != 176 {
		t.Errorf("expected %d project mappings, found %d", 176, len(m))
	}
	// Make sure that a sample project is present.
	if m["chromiumos/platform/mosys"]["refs/heads/master"] != "src/platform/mosys" {
		t.Errorf("expected to find a mapping for mosys repo. Got mappings: %v", m)
	}
}

func TestGetRepoToRemoteBranchToSourceRootFromManifestFile_success(t *testing.T) {
	m, err := GetRepoToRemoteBranchToSourceRootFromManifestFile("test_data/foo.xml")
	if err != nil {
		t.Error(err)
	}
	if len(m) != 4 {
		t.Errorf("expected %d project mappings, found %d", 4, len(m))
	}
	// Make sure that a sample project is present.
	if m["baz"]["refs/heads/master"] != "baz/" {
		t.Errorf("expected to find a mapping for baz. Got mappings: %v", m)
	}
}

func TestGetRepoToRemoteBranchToSourceRootFromManifestFile_duplicate(t *testing.T) {
	m, err := GetRepoToRemoteBranchToSourceRootFromManifestFile("test_data/duplicate.xml")
	if err != nil {
		t.Error(err)
	}
	if len(m) != 1 {
		t.Errorf("expected %d project mappings, found %d", 1, len(m))
	}
	// The last mapping for a given name and branch should take precedent.
	if m["foo"]["refs/heads/master"] != "buz/" {
		t.Errorf("expected to find a mapping for buz. Got mappings: %v", m)
	}
}

func ManifestEq(a, b *Manifest) bool {
	if len(a.Projects) != len(b.Projects) {
		return false
	}
	for i := range a.Projects {
		if !reflect.DeepEqual(&a.Projects[i], &b.Projects[i]) {
			return false
		}
	}
	if len(a.Includes) != len(b.Includes) {
		return false
	}
	for i := range a.Includes {
		if a.Includes[i] != b.Includes[i] {
			return false
		}
	}
	return true
}

func ManifestMapEq(expected, actual map[string]*Manifest) error {
	for file := range expected {
		if _, ok := actual[file]; !ok {
			return fmt.Errorf("missing manifest %s", file)
		}
		if !ManifestEq(expected[file], actual[file]) {
			return fmt.Errorf("expected %v, found %v", expected[file], actual[file])
		}
	}
	return nil
}

func TestResolveImplicitLinks(t *testing.T) {
	manifest := &Manifest{
		Default: Default{
			RemoteName: "chromeos",
			Revision:   "123",
		},
		Remotes: []Remote{
			{
				Fetch: "https://chromium.org/remote",
				Name:  "chromium",
				Alias: "chromeos",
			},
			{
				Fetch:    "https://google.com/remote",
				Name:     "google",
				Revision: "125",
			},
		},
		Projects: []Project{
			{Path: "baz/", Name: "baz", RemoteName: "chromium"},
			{Path: "fiz/", Name: "fiz", Revision: "124"},
			{Name: "buz", RemoteName: "google",
				Annotations: []Annotation{
					{Name: "branch-mode", Value: "pin"},
				},
			},
		},
		Includes: []Include{
			{"bar.xml"},
		},
	}

	expected := &Manifest{
		Projects: []Project{
			{Path: "baz/", Name: "baz", Revision: "123", RemoteName: "chromium"},
			{Path: "fiz/", Name: "fiz", Revision: "124", RemoteName: "chromeos"},
			{Path: "buz", Name: "buz", Revision: "125", RemoteName: "google",
				Annotations: []Annotation{
					{Name: "branch-mode", Value: "pin"},
				},
			},
		},
		Includes: []Include{
			{"bar.xml"},
		},
	}

	manifest.ResolveImplicitLinks()
	assert.Assert(t, ManifestEq(manifest, expected))
}
func TestLoadManifestTree_success(t *testing.T) {
	expectedResults := make(map[string]*Manifest)
	expectedResults["foo.xml"] = &Manifest{
		Default: Default{
			RemoteName: "chromeos",
			Revision:   "123",
		},
		Remotes: []Remote{
			{
				Fetch: "https://chromium.org/remote",
				Name:  "chromium",
				Alias: "chromeos",
			},
			{
				Fetch:    "https://google.com/remote",
				Name:     "google",
				Revision: "125",
			},
		},
		Projects: []Project{
			{Path: "baz/", Name: "baz", RemoteName: "chromium"},
			{Path: "fiz/", Name: "fiz", Revision: "124"},
			{Name: "buz", RemoteName: "google",
				Annotations: []Annotation{
					{Name: "branch-mode", Value: "pin"},
				},
			},
		},
		Includes: []Include{
			{"bar.xml"},
		},
	}
	expectedResults["bar.xml"] = &Manifest{
		Projects: []Project{
			{Path: "baz/", Name: "baz"},
			{Path: "project/", Name: "project"},
		},
	}

	res, err := LoadManifestTree("test_data/foo.xml")
	assert.NilError(t, err)
	if err = ManifestMapEq(expectedResults, res); err != nil {
		t.Errorf(err.Error())
	}
}

func TestLoadManifestTree_bad_include(t *testing.T) {
	_, err := LoadManifestTree("test_data/bogus.xml")
	assert.ErrorContains(t, err, "bad-include.xml")
}

func TestLoadManifestTree_bad_xml(t *testing.T) {
	_, err := LoadManifestTree("test_data/invalid.xml")
	assert.ErrorContains(t, err, "unmarshal")
}

func TestGetUniqueProject(t *testing.T) {
	manifest := &Manifest{
		Projects: []Project{
			{Path: "foo-a/", Name: "foo"},
			{Path: "foo-b/", Name: "foo"},
			{Path: "bar/", Name: "bar"},
		},
	}

	_, err := manifest.GetUniqueProject("foo")
	assert.ErrorContains(t, err, "multiple projects")

	project, err := manifest.GetUniqueProject("bar")
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(&project, &manifest.Projects[2]))
}

func TestWrite(t *testing.T) {
	tmpDir := "repotest_tmp_dir"
	tmpDir, err := ioutil.TempDir("", tmpDir)
	defer os.RemoveAll(tmpDir)
	assert.NilError(t, err)
	tmpPath := filepath.Join(tmpDir, "foo.xml")

	manifest := &Manifest{
		Projects: []Project{
			{Path: "foo-a/", Name: "foo"},
			{Path: "foo-b/", Name: "foo"},
			{Path: "bar/", Name: "bar"},
		},
	}
	manifest.Write(tmpPath)
	// Make sure file was written successfully.
	_, err = os.Stat(tmpPath)
	assert.NilError(t, err)
	// Make sure manifest was marshalled and written correctly.
	manifestMap, err := LoadManifestTree(tmpPath)
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(manifest, manifestMap["foo.xml"]))
}

func TestGitName(t *testing.T) {
	remote := Remote{
		Alias: "batman",
		Name:  "bruce wayne",
	}
	assert.StringsEqual(t, remote.GitName(), "batman")
	remote = Remote{
		Name: "robin",
	}
	assert.StringsEqual(t, remote.GitName(), "robin")
}

func TestGetProjectByName(t *testing.T) {
	m := Manifest{
		Projects: []Project{
			{Path: "a/", Name: "a"},
			{Path: "b/", Name: "b"},
			{Path: "c/", Name: "c"},
		},
	}

	project, err := m.GetProjectByName("b")
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(*project, m.Projects[1]))
	project, err = m.GetProjectByName("d")
	assert.Assert(t, err != nil)
}
func TestGetProjectByPath(t *testing.T) {
	m := Manifest{
		Projects: []Project{
			{Path: "a/", Name: "a"},
			{Path: "b/", Name: "b"},
		},
	}

	project, err := m.GetProjectByPath("b/")
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(*project, m.Projects[1]))

	// Add a project after the fact to test the internal mapping.
	m.Projects = append(m.Projects, Project{Path: "c/", Name: "c"})
	project, err = m.GetProjectByPath("c/")
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(*project, m.Projects[2]))

	project, err = m.GetProjectByPath("d/")
	assert.Assert(t, err != nil)
}

func deref(projects []*Project) []Project {
	res := []Project{}
	for _, project := range projects {
		res = append(res, *project)
	}
	return res
}

func TestGetProjects(t *testing.T) {
	m := Manifest{
		Projects: []Project{
			{Path: "a1/", Name: "chromiumos/a"},
			{Path: "a2/", Name: "chromiumos/a", Annotations: []Annotation{{Name: "branch-mode", Value: "pin"}}},
			{Path: "b/", Name: "b", Annotations: []Annotation{{Name: "branch-mode", Value: "pin"}}},
			{Path: "c/", Name: "c", Annotations: []Annotation{{Name: "branch-mode", Value: "tot"}}},
			{Path: "d/", Name: "chromiumos/d"},
			{Path: "e/", Name: "chromiumos/e"},
		},
		Remotes: []Remote{
			{Name: "cros"},
		},
		Default: Default{
			RemoteName: "cros",
		},
	}
	singleProjects := deref(m.GetSingleCheckoutProjects())
	assert.Assert(t, reflect.DeepEqual(singleProjects, m.Projects[4:6]))
	multiProjects := deref(m.GetMultiCheckoutProjects())
	assert.Assert(t, reflect.DeepEqual(multiProjects, m.Projects[:2]))
	pinnedProjects := deref(m.GetPinnedProjects())
	assert.Assert(t, reflect.DeepEqual(pinnedProjects, m.Projects[1:3]))
	totProjects := deref(m.GetTotProjects())
	assert.Assert(t, reflect.DeepEqual(totProjects, m.Projects[3:4]))
}

var canBranchTestManifestAnnotation = Manifest{
	Projects: []Project{
		// Projects with annotations labeling branch mode.
		{Path: "foo1/", Name: "foo1",
			Annotations: []Annotation{
				{Name: "branch-mode", Value: "create"},
			},
		},
		{Path: "foo2/", Name: "foo2",
			Annotations: []Annotation{
				{Name: "branch-mode", Value: "pin"},
			},
		},
		{Path: "foo3/", Name: "foo3",
			Annotations: []Annotation{
				{Name: "branch-mode", Value: "tot"},
			},
		},
		{Path: "foo4/", Name: "foo4",
			Annotations: []Annotation{
				{Name: "branch-mode", Value: "bogus"},
			},
		},
	},
}
var canBranchTestManifestRemote = Manifest{
	Projects: []Project{
		// Remote has name but no alias. Project is branchable.
		{Path: "bar/", Name: "chromiumos/bar", RemoteName: "cros"},
		// Remote has alias. Project is branchable.
		{Path: "baz1/", Name: "aosp/baz", RemoteName: "cros1"},
		// Remote has alias. Remote is not a cros remote.
		{Path: "baz2/", Name: "aosp/baz", RemoteName: "cros2"},
		// Remote has alias. Remote is a cros remote, but not a branchable one.
		{Path: "fizz/", Name: "fizz", RemoteName: "cros"},
		// Remote has name but no alias. Remote is a branchable remote, but specific
		// project is not branchable.
		{Path: "buzz/", Name: "buzz", RemoteName: "weave"},
	},
	Remotes: []Remote{
		{Name: "cros"},
		{Name: "cros1", Alias: "cros"},
		{Name: "cros2", Alias: "github"},
		{Name: "weave"},
	},
}

func assertBranchModesEqual(t *testing.T, a, b BranchMode) {
	assert.StringsEqual(t, string(a), string(b))
}

func TestProjectBranchMode_annotation(t *testing.T) {
	manifest := canBranchTestManifestAnnotation
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[0]), Create)
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[1]), Pinned)
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[2]), Tot)
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[3]), UnspecifiedMode)
}

func TestProjectBranchMode_remote(t *testing.T) {
	manifest := canBranchTestManifestRemote
	// Remote has name but no alias. Project is branchable.
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[0]), Create)
	// Remote has alias. Project is branchable.
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[1]), Create)
	// Remote has alias. Remote is not a cros remote.
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[2]), Pinned)
	// Remote has alias. Remote is a cros remote, but not a branchable one.
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[4]), Pinned)
	// Remote has name but no alias. Remote is a branchable remote, but specific
	// project is not branchable.
	assertBranchModesEqual(t, manifest.ProjectBranchMode(manifest.Projects[3]), Pinned)
}

func TestMergeManifests(t *testing.T) {
	// Manifest inheritance is as follows:
	// a --> b --> c
	//  \
	//   \--> d
	a := Manifest{
		Default: Default{
			RemoteName: "cros",
			Revision:   "refs/heads/master",
		},
		Remotes: []Remote{
			{Name: "cros"},
			{Name: "cros-internal"},
		},
		Projects: []Project{
			{Path: "project1/", Name: "project1"},
			{Path: "project2/", Name: "project2"},
			{Path: "project3/", Name: "project3", RemoteName: "cros-internal"},
		},
		Includes: []Include{
			{Name: "b.xml"},
			{Name: "d.xml"},
		},
	}
	b := Manifest{
		Default: Default{
			RemoteName: "cros-internal",
			Revision:   "refs/heads/internal",
		},
		Remotes: []Remote{
			{Name: "cros"},
			{Name: "cros-internal"},
		},
		Projects: []Project{
			{Path: "project3-v2/", Name: "project3"},
			{Path: "project4/", Name: "project4"},
		},
		Includes: []Include{
			{Name: "c.xml"},
		},
	}
	c := Manifest{
		Default: Default{
			RemoteName: "cros-special",
			Revision:   "refs/heads/special",
		},
		Remotes: []Remote{
			{Name: "cros-special", Revision: "refs/heads/unique"},
		},
		Projects: []Project{
			{Path: "project5/", Name: "project5"},
		},
	}
	d := Manifest{
		Default: Default{
			RemoteName: "cros",
			Revision:   "refs/heads/develop",
		},
		Remotes: []Remote{
			{Name: "cros"},
		},
		Projects: []Project{
			{Path: "project6/", Name: "project6"},
			{Path: "project7/", Name: "project7"},
		},
	}
	manifestDict := map[string]*Manifest{
		"a.xml": &a,
		"b.xml": &b,
		"c.xml": &c,
		"d.xml": &d,
	}
	expected := Manifest{
		Default: Default{
			RemoteName: "cros",
			Revision:   "refs/heads/master",
		},
		Remotes: []Remote{
			{Name: "cros"},
			{Name: "cros-internal"},
			{Name: "cros-special", Revision: "refs/heads/unique"},
		},
		Projects: []Project{
			{Path: "project1/", Name: "project1", RemoteName: "cros", Revision: "refs/heads/master"},
			{Path: "project2/", Name: "project2", RemoteName: "cros", Revision: "refs/heads/master"},
			{Path: "project3/", Name: "project3", RemoteName: "cros-internal", Revision: "refs/heads/master"},
			{Path: "project3-v2/", Name: "project3", RemoteName: "cros-internal", Revision: "refs/heads/internal"},
			{Path: "project4/", Name: "project4", RemoteName: "cros-internal", Revision: "refs/heads/internal"},
			{Path: "project5/", Name: "project5", RemoteName: "cros-special", Revision: "refs/heads/unique"},
			{Path: "project6/", Name: "project6", RemoteName: "cros", Revision: "refs/heads/develop"},
			{Path: "project7/", Name: "project7", RemoteName: "cros", Revision: "refs/heads/develop"},
		},
		Includes: []Include{},
	}
	mergedManifest, err := MergeManifests("a.xml", &manifestDict)
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(expected, *mergedManifest))
}

func TestLoadManifestFromFileWithIncludes(t *testing.T) {
	expectedProjectNames := []string{"baz", "fiz", "buz", "project"}

	res, err := LoadManifestFromFileWithIncludes("test_data/foo.xml")
	assert.NilError(t, err)

	projectNames := make([]string, len(res.Projects))
	for i, project := range res.Projects {
		projectNames[i] = project.Name
	}
	assert.Assert(t, util.UnorderedEqual(expectedProjectNames, projectNames))
}

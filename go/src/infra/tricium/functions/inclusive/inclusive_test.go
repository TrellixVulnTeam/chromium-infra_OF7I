package main

import (
	"os"
	"path/filepath"
	"testing"

	tricium "infra/tricium/api/v1"

	. "go.chromium.org/luci/common/testing/assertions"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	okSource          = "test/src/ok.md"
	notOkPath         = "test/src/blacklist.txt"
	okPathNotOkSource = "test/src/list.txt"
)

func TestInclusiveLanguageChecker(t *testing.T) {
	buildDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("error getting current working directory: %v", err)
	}
	Convey("Produces no comment for text containing no blocked terms", t, func() {
		results := &tricium.Data_Results{}
		checkInclusiveLanguage(filepath.Join(buildDir, okSource), okSource, results)
		So(results.Comments, ShouldBeNil)
	})

	Convey("Flags blocked terms in file contents", t, func() {
		results := &tricium.Data_Results{}
		checkInclusiveLanguage(filepath.Join(buildDir, okPathNotOkSource), okPathNotOkSource, results)
		So(results.Comments, ShouldNotBeNil)
		So(results.Comments[0], ShouldResembleProto, &tricium.Data_Comment{
			Category:  "InclusiveLanguageCheck/Warning",
			Message:   commentText["blacklist"],
			Path:      okPathNotOkSource,
			StartLine: 2,
			EndLine:   2,
			StartChar: 8,
			EndChar:   17,
			Suggestions: []*tricium.Data_Suggestion{{
				Description: commentText["blacklist"],
				Replacements: []*tricium.Data_Replacement{{
					Replacement: "blocklist",
					Path:        okPathNotOkSource,
					StartLine:   2,
					EndLine:     2,
					StartChar:   8,
					EndChar:     17,
				}},
			}},
		})
	})

	Convey("Flags blocked terms in file names and file contents", t, func() {
		results := &tricium.Data_Results{}
		checkInclusiveLanguage(filepath.Join(buildDir, notOkPath), notOkPath, results)
		So(results.Comments, ShouldNotBeNil)
		So(results.Comments[0], ShouldResembleProto, &tricium.Data_Comment{
			Category:  "InclusiveLanguageCheck/Warning",
			Message:   commentText["blacklist"],
			Path:      notOkPath,
			StartLine: 0,
			EndLine:   0,
			StartChar: 9,
			EndChar:   18,
			Suggestions: []*tricium.Data_Suggestion{{
				Description: commentText["blacklist"],
				Replacements: []*tricium.Data_Replacement{{
					Replacement: "blocklist",
					Path:        notOkPath,
					StartLine:   0,
					EndLine:     0,
					StartChar:   9,
					EndChar:     18,
				}},
			}},
		})
		So(results.Comments[1], ShouldResembleProto, &tricium.Data_Comment{
			Category:  "InclusiveLanguageCheck/Warning",
			Message:   commentText["blacklist"],
			Path:      notOkPath,
			StartLine: 2,
			EndLine:   2,
			StartChar: 8,
			EndChar:   17,
			Suggestions: []*tricium.Data_Suggestion{{
				Description: commentText["blacklist"],
				Replacements: []*tricium.Data_Replacement{{
					Replacement: "blocklist",
					Path:        notOkPath,
					StartLine:   2,
					EndLine:     2,
					StartChar:   8,
					EndChar:     17,
				}},
			}},
		})
	})
}

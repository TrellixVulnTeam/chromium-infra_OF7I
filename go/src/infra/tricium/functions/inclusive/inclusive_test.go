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
	nocheckSource     = "test/src/nocheck.txt"
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

	Convey("Ingores nocheck lines", t, func() {
		results := &tricium.Data_Results{}
		checkInclusiveLanguage(filepath.Join(buildDir, nocheckSource), nocheckSource, results)
		So(results.Comments, ShouldNotBeNil)
		So(results.Comments, ShouldResembleProto, []*tricium.Data_Comment{
			{
				Category:  "InclusiveLanguageCheck/Warning",
				Message:   commentText["master"],
				Path:      nocheckSource,
				StartLine: 1,
				EndLine:   1,
				StartChar: 0,
				EndChar:   6,
				Suggestions: []*tricium.Data_Suggestion{{
					Description: commentText["master"],
					Replacements: []*tricium.Data_Replacement{{
						Replacement: "main",
						Path:        nocheckSource,
						StartLine:   1,
						EndLine:     1,
						StartChar:   0,
						EndChar:     6,
					}},
				}},
			},
			{
				Category:  "InclusiveLanguageCheck/Warning",
				Message:   commentText["blacklist"],
				Path:      nocheckSource,
				StartLine: 3,
				EndLine:   3,
				StartChar: 0,
				EndChar:   9,
				Suggestions: []*tricium.Data_Suggestion{{
					Description: commentText["blacklist"],
					Replacements: []*tricium.Data_Replacement{{
						Replacement: "blocklist",
						Path:        nocheckSource,
						StartLine:   3,
						EndLine:     3,
						StartChar:   0,
						EndChar:     9,
					}},
				}},
			},
			{
				Category:  "InclusiveLanguageCheck/Warning",
				Message:   commentText["whitelist"],
				Path:      nocheckSource,
				StartLine: 4,
				EndLine:   4,
				StartChar: 0,
				EndChar:   9,
				Suggestions: []*tricium.Data_Suggestion{{
					Description: commentText["whitelist"],
					Replacements: []*tricium.Data_Replacement{{
						Replacement: "allowlist",
						Path:        nocheckSource,
						StartLine:   4,
						EndLine:     4,
						StartChar:   0,
						EndChar:     9,
					}},
				}},
			},
			{
				Category:  "InclusiveLanguageCheck/Warning",
				Message:   commentText["slave"],
				Path:      nocheckSource,
				StartLine: 16,
				EndLine:   16,
				StartChar: 16,
				EndChar:   21,
				Suggestions: []*tricium.Data_Suggestion{{
					Description: commentText["slave"],
					Replacements: []*tricium.Data_Replacement{{
						Replacement: "replica",
						Path:        nocheckSource,
						StartLine:   16,
						EndLine:     16,
						StartChar:   16,
						EndChar:     21,
					}},
				}},
			},
			{
				Category:  "InclusiveLanguageCheck/Warning",
				Message:   commentText["master"],
				Path:      nocheckSource,
				StartLine: 18,
				EndLine:   18,
				StartChar: 16,
				EndChar:   22,
				Suggestions: []*tricium.Data_Suggestion{{
					Description: commentText["master"],
					Replacements: []*tricium.Data_Replacement{{
						Replacement: "main",
						Path:        nocheckSource,
						StartLine:   18,
						EndLine:     18,
						StartChar:   16,
						EndChar:     22,
					}},
				}},
			}, {
				Category:  "InclusiveLanguageCheck/Warning",
				Message:   commentText["master"],
				Path:      nocheckSource,
				StartLine: 21,
				EndLine:   21,
				StartChar: 93,
				EndChar:   99,
				Suggestions: []*tricium.Data_Suggestion{{
					Description: commentText["master"],
					Replacements: []*tricium.Data_Replacement{{
						Replacement: "main",
						Path:        nocheckSource,
						StartLine:   21,
						EndLine:     21,
						StartChar:   93,
						EndChar:     99,
					}},
				}},
			},
		})
	})
}

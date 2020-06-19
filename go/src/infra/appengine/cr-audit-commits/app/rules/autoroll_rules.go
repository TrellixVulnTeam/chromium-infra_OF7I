// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"fmt"
	"strings"
)

// AutoRollRules returns an AccountRules instance for an account
// which should only modify the given set of files and directories.
func AutoRollRules(account string, files, dirs []string) AccountRules {
	return AccountRules{
		Account: account,
		Rules: []Rule{
			OnlyModifiesFilesAndDirsRule{
				Name:  fmt.Sprintf("OnlyModifies_%s", strings.Join(append(files, dirs...), "+")),
				Files: files,
				Dirs:  dirs,
			},
		},
		Notification: CommentOrFileMonorailIssue{
			Components: []string{"Infra>Audit>AutoRoller"},
			Labels:     []string{"CommitLog-Audit-Violation"},
		},
	}
}

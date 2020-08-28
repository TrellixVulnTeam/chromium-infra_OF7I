# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
"""Some constants used by Monorail's v3 API."""

# Max comments per page in the ListComment API.
MAX_COMMENTS_PER_PAGE = 100

# Max issues per page in the SearchIssues API.
MAX_ISSUES_PER_PAGE = 100

# Max issues tp fetch in the BatchGetIssues API.
MAX_BATCH_ISSUES = 100

# Max issues to modify at once in the ModifyIssues API.
MAX_MODIFY_ISSUES = 100

# Max impacted issues allowed in a ModifyIssues API.
MAX_MODIFY_IMPACTED_ISSUES = 50

# Max approval values to modify at once in the ModifyIssueApprovalValues API.
MAX_MODIFY_APPROVAL_VALUES = 100

# Max users to fetch in the BatchGetUsers API.
MAX_BATCH_USERS = 100

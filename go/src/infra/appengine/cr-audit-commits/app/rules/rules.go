package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"context"

	"go.chromium.org/luci/common/proto/git"
)

// DynamicRefFunc is a functype for functions that match a RefConfig with a
// dynamically determined ref.
//
// It is expected to receive the generic RefConfig as hardcoded in RulesMap,
// passed by value to prevent the implementation from modifying it.
// It is expected to return a slice of references to RefConfigs, where each
// matches a ref to audit, and its values BranchName and Metadata have been
// modified accordingly.
//
// Note that for changes to any other field of RefConfig made by functions
// implementing this interface to persist and apply to audits, the Scheduler
// needs to be modified to save them to the RefState, and the SetConcreteRef
// function below needs to be modified to set them in the copy of RefConfig to
// be passed to the scan/audit/notify functions.
type DynamicRefFunc func(context.Context, RefConfig) ([]*RefConfig, error)

// RefConfig represents the hard-coded config for a monitored repo and a
// pointer to the entity representing its datastore-persisted state.
type RefConfig struct { // These are expected to be hard-coded.
	BaseRepoURL     string
	GerritURL       string
	BranchName      string
	Metadata        string
	StartingCommit  string
	MonorailAPIURL  string // Only intended for release branches
	MonorailProject string
	// Do not use "AuditFailure" as a key in this map, it may cause a clash
	// with the notification state for failed audits.
	Rules              map[string]AccountRules
	NotifierEmail      string
	DynamicRefFunction DynamicRefFunc
}

// BranchInfo represents the main branch information of a specific Chrome release
type BranchInfo struct {
	PdfiumBranch   string `json:"pdfium_branch"`
	SkiaBranch     string `json:"skia_branch"`
	WebrtcBranch   string `json:"webrtc_branch"`
	V8Branch       string `json:"v8_branch"`
	ChromiumBranch string `json:"chromium_branch"`
	Milestone      int    `json:"milestone"`
}

// RepoURL composes the url of the repository by appending the branch.
func (rc *RefConfig) RepoURL() string {
	return rc.BaseRepoURL + "/+/" + rc.BranchName
}

// LinkToCommit composes a url to a specific commit
func (rc *RefConfig) LinkToCommit(commit string) string {
	return rc.BaseRepoURL + "/+/" + commit
}

// SetConcreteRef returns a copy of the repoconfig modified to account for
// dynamic refs.
func (rc *RefConfig) SetConcreteRef(ctx context.Context, rs *RefState) *RefConfig {
	// Make a copy.
	result := *rc
	if rs.BranchName != "" {
		result.BranchName = rs.BranchName
	}
	if rs.Metadata != "" {
		result.Metadata = rs.Metadata
	}
	return &result
}

// AccountRules is a rule that applies to a commit if the commit has a given
// account as either its author or its committer.
type AccountRules struct {
	// Account is the account to filter on for account specific rules.
	Account              string
	Rules                []Rule
	NotificationFunction NotificationFunc
}

// MatchesCommit determines whether the AccountRules set it's bound to, applies
// to the given commit.
func (ar AccountRules) MatchesCommit(c *git.Commit) bool {
	return ar.Account == "*" || c.GetCommitter().GetEmail() == ar.Account || c.GetAuthor().GetEmail() == ar.Account
}

// MatchesRelevantCommit determines whether the AccountRules set it's bound to,
// applies to the given commit entity.
func (ar AccountRules) MatchesRelevantCommit(c *RelevantCommit) bool {
	return ar.Account == "*" || c.CommitterAccount == ar.Account || c.AuthorAccount == ar.Account
}

// AuditParams exposes object shared by all rules (and the worker goroutines
// they are run on).
type AuditParams struct {
	TriggeringAccount string
	RepoCfg           *RefConfig
	RefState          *RefState
}

// Rule is an audit rule.
//
// It has a name getter and a rule implementation.
type Rule interface {
	// GetName returns the name of the rule as a string. It is expected to be unique in
	// each repository it applies to.
	GetName() string
	// Run performs a check according to the Rule.
	//
	// This could be called multiple times on the same commit if the rule fails,
	// and needs to be retried or if the rule ran previously and resulted in a
	// non-final state. Rules should self-limit the frequency with which they poll
	// external systems for a given commit.
	//
	// Rules are expected to return an error if they cannot determine whether a policy
	// has been broken or not.
	//
	// Run methods should return a reference to a RuleResult
	Run(context.Context, *AuditParams, *RelevantCommit, *Clients) (*RuleResult, error)
}

// PreviousResult returns the result for a previous application of the rule on the given commit
// or nil.
func PreviousResult(ctx context.Context, rc *RelevantCommit, ruleName string) *RuleResult {
	for _, rr := range rc.Result {
		if rr.RuleName == ruleName {
			return &rr
		}
	}
	return nil
}

// ReleaseConfig is the skeleton of a function to get the ref and milestone
// dynamically.
func ReleaseConfig(ctx context.Context, cfg RefConfig) ([]*RefConfig, error) {

	var branchRefsURLContents []string
	// ------------------
	// OMAHAPROXY||CHROMIUMDASH MAGIC
	// ------------------

	// https://chromiumdash.appspot.com/fetch_milestones is a legacy API that needs some clean up. Here,
	// the platform could be any Chrome platform orther than Android and the result will still be the same.
	contents, err := getURLAsString(ctx, "https://chromiumdash.appspot.com/fetch_milestones?platform=Android")
	if err != nil {
		return nil, err
	}
	branchInfos := []BranchInfo{}
	err = json.Unmarshal([]byte(contents), &branchInfos)
	if err != nil {
		return nil, err
	}
	// We only need Stable and Beta branch details.
	for i := range branchInfos[:2] {
		branchRefsURL := fmt.Sprintf("https://chromium.googlesource.com/chromium/src.git/+log/refs/heads/master..refs/branch-heads/%s/?format=json&n=1000", branchInfos[i].ChromiumBranch)
		// When scanning a branch for the first time, it's unlikely that there'll be more than 1000 commits in it. In subsequent scans, the starting commit will be ignored,
		// but instead the last scanned commit will be used. So even if the commits in the branch exceed 1000 there will be no effect in the auditing.
		branchContents, err := getURLAsString(ctx, branchRefsURL)
		if err != nil {
			return nil, err
		}
		branchRefsURLContents = append(branchRefsURLContents, branchContents)
	}

	return GetReleaseConfig(ctx, cfg, branchRefsURLContents, branchInfos)
}

// GetReleaseConfig is a helper function to get the ref and milestone dynamically.
func GetReleaseConfig(ctx context.Context, cfg RefConfig, branchRefsURLContents []string, branchInfos []BranchInfo) ([]*RefConfig, error) {
	concreteConfigs := []*RefConfig{}
	var err error
	r := regexp.MustCompile(`"commit": "(.+)?"`)
	for i := range branchRefsURLContents {
		temp := strings.Split(branchRefsURLContents[i], "\n")
		lastCommit := ""
		for j := 0; j < len(temp); j++ {
			if r.MatchString(temp[j]) {
				lastCommit = temp[j]
			}
		}
		if lastCommit == "" {
			return nil, errors.New("commit not found or invalid")
		}
		concreteConfig := cfg
		concreteConfig.StartingCommit = r.FindStringSubmatch(lastCommit)[1]
		concreteConfig.BranchName = fmt.Sprintf("refs/branch-heads/%s", branchInfos[i].ChromiumBranch)
		concreteConfig.Metadata, err = SetToken(ctx, "MilestoneNumber", strconv.Itoa(branchInfos[i].Milestone), concreteConfig.Metadata)
		concreteConfigs = append(concreteConfigs, &concreteConfig)
	}
	return concreteConfigs, err
}

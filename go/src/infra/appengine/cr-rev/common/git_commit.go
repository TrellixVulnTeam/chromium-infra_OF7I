package common

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

const gitCommitPositionFooterName = "Cr-Commit-Position"
const svnCommitPositionFooterName = "git-svn-id"

var footerFormat = regexp.MustCompile(`(?m)^([a-zA-Z0-9-_]+)\s*:\s*(.*)$`)
var gitCommitPositionFormat = regexp.MustCompile(`(?P<name>.*)@{#(?P<number>\d+)}`)
var svnCommitPositionFormat = regexp.MustCompile(`(?P<name>.*)@(?P<number>\d+)`)

var errNoPositionFooter = errors.New("No position footer found")
var errInvalidPositionFooter = errors.New("Invalid position footer format")

// GitCommit holds information about single Git commit.
type GitCommit struct {
	Host          string
	Repository    string
	Sha1          string
	CommitMessage string

	footers map[string][]string
}

// CommitPosition is extracted from Git commit message and it uniquely
// identifies a commit.
type CommitPosition struct {
	// Name is either ref name for commits created after SVN-Git
	// transition, or SVN URL/branch name for commits before Git.
	Name string
	// Sequential number.
	Number int
}

// ID uniquely identifies commit.
func (c *GitCommit) ID() string {
	return fmt.Sprintf("%s-%s-%s", c.Host, c.Repository, c.Sha1)
}

// GetFooters parses git commit message and extracts desired footers. A footer
// must contain key and value separated by a colon.
func (c *GitCommit) GetFooters(name string) []string {
	if c.footers != nil {
		return c.footers[name]
	}
	c.footers = make(map[string][]string)
	results := footerFormat.FindAllStringSubmatch(c.CommitMessage, -1)
	for _, result := range results {
		if v, ok := c.footers[result[1]]; ok {
			c.footers[result[1]] = append(v, result[2])
		} else {
			c.footers[result[1]] = []string{result[2]}
		}
	}
	return c.footers[name]
}

// GetPositionNumber looks for Cr-Commit-Position or git-svn-id in commit
// message and returns number in that line.  If there are multiple matching
// lines, the last instance is returned.
func (c *GitCommit) GetPositionNumber() (*CommitPosition, error) {
	// extractPosition is helper function that returns the last matching
	// line of re, or error if there is not match.
	extractPosition := func(footerName string, re *regexp.Regexp) (
		*CommitPosition, error) {
		footers := c.GetFooters(footerName)
		if len(footers) == 0 {
			return nil, errNoPositionFooter
		}
		lastFooter := footers[len(footers)-1]
		match := re.FindStringSubmatch(lastFooter)
		if len(match) == 0 {
			return nil, errInvalidPositionFooter
		}
		cp := CommitPosition{}
		// Extract named parameters from regex, search for name and
		// number.
		for i, name := range re.SubexpNames() {
			switch name {
			case "name":
				cp.Name = match[i]
				break
			case "number":
				positionNumber, err := strconv.Atoi(match[2])
				if err != nil {
					return nil, err
				}
				cp.Number = positionNumber
				break
			}
		}
		return &cp, nil
	}

	pos, err := extractPosition(
		gitCommitPositionFooterName, gitCommitPositionFormat)
	if err == errNoPositionFooter {
		// Git position was not found, try SVN.
		pos, err = extractPosition(
			svnCommitPositionFooterName, svnCommitPositionFormat)
	}
	return pos, err
}

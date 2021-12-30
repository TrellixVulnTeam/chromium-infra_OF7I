// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testname

import (
	"fmt"
	"infra/appengine/weetbix/internal/clustering/rules/lang"
	"regexp"
	"strings"

	"go.chromium.org/luci/common/errors"
)

// likeRewriter escapes usages of '\', '%' and '_', so that
// the original text is interpreted literally in a LIKE
// expression.
var likeRewriter = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// substitutionRE matches use of the '$' operator which
// may be used in templates to substitute in values.
// Captured usages are either:
// - ${name}, which tells the template to insert the value
//   of the capture group with that name.
// - $$, which tells the template insert a literal '$'
//   into the output.
// - $, which indicates an invalid use of the '$' operator
//   ($ not followed by $ or {name}).
var substitutionRE = regexp.MustCompile(`\$\{(\w+?)\}|\$\$?`)

// RuleEvaluator evaluates a test name clustering rule on
// a test name, returning whether the rule matches and
// if so, the LIKE expression that defines the cluster.
type RuleEvaluator func(testName string) (like string, ok bool)

// ClusteringRule is a rule used to cluster a test result by test name.
// TODO(crbug.com/1243174): Make this a message in project_config.proto when
// this is made configurable on a per-project basis.
type ClusteringRule struct {
	// A human-readable name for the rule. This should be unique for each rule.
	// This may be used by Weetbix to explain why it chose to cluster the test
	// name in this way.
	Name string

	// The regular expression describing which test names should be clustered
	// by this rule.
	//
	// Example.
	//   Assume our project uploads google test (gtest) results with the test
	//   name prefix "gtest://".
	//   If want to cluster value-parameterized google tests
	//   together based on the test suite and test case name (ignoring
	//   the value parameter), we may use a pattern like:
	//     "^gtest://(\w+/)?(?P<testcase>\w+\.\w+)/\w+$"
	//
	//   This will allow us to cluster test names like:
	//     "gtest://InstantiationOne/ColorSpaceTest.testNullTransform/0"
	//     "gtest://InstantiationOne/ColorSpaceTest.testNullTransform/1"
	//     "gtest://InstantiationTwo/ColorSpaceTest.testNullTransform/0"
	//   together.
	//
	//   See https://github.com/google/googletest/blob/main/docs/advanced.md#how-to-write-value-parameterized-tests
	//   to further understand this example.
	//
	// Use ?P<name> to name capture groups, so their values can be used in
	// like_template below.
	Pattern string

	// The template used to generate a LIKE expression on test names
	// that defines the test name cluster identified by this rule.
	//
	// This like expression has two purposes:
	// (1) If the test name cluster is large enough to jusify the
	//     creation of a bug cluster, the like expression is used to
	//     generate a failure association rule of the following form:
	//        test LIKE "<evaluated like_template>"
	// (2) A hash of the expression is used as the clustering key for the
	//     test name-based suggested cluster. This generally has the desired
	//     clustering behaviour, i.e. the parts of the test name which
	//     are important enough to included in the LIKE expression for (1)
	//     are also those on which clustering should occur.
	//
	// As is usual for LIKE expressions, the template can contain
	// the following operators to do wildcard matching:
	// * '%' for wildcard match of an arbitrary number of characters, and
	// * '_' for single character wildcard match.
	//
	// The template can refer to parts of the test name matched by
	// the rule pattern using ${name}, where name refers to the capture
	// group (see pattern). To insert the literal '$', the sequence '$$'
	// should be used.
	//
	// Example.
	//   Following the same gtest example as for the pattern field,
	//   we may use the template:
	//     "gtest://%${testcase}%"
	//
	//   When instantiated for the above example, the result would be
	//   a failure association rule like:
	//     test LIKE "gtest://%ColorSpaceTest.testNullTransform%"
	//
	//   Note the use of ${testcase} to refer to the testname capture group
	//   specified in the pattern example.
	//
	// It is known that not all clusters can be precisely matched by
	// a LIKE expression. Nonetheless, Weetbix prefers LIKE expressions
	// as they are easier to comprehend and modify by users, and in
	// most cases, the added precision is not required.
	//
	// As such, your rule should try to ensure the generated LIKE statement
	// captures your clustering logic as best it can. Your LIKE expression
	// MUST match all test names matched by your regex pattern, and MAY
	// capture additional test names (though this is preferably minimised,
	// to reduce differences between the suggested clusters and eventual
	// bug clusters).
	//
	// Weetbix will automatically escape any '%' '_' and '\' in parts of
	// the matched test name before substitution to ensure captured parts
	// of the test name are matched literally and not interpreted.
	LikeTemplate string
}

// Compile produces a RuleEvaluator that can quickly evaluate
// whether a given test name matches the given test name
// clustering rule, and if so, return the test name LIKE
// expression that defines the cluster.
//
// As Compiling rules is slow, the result should be cached.
func (c *ClusteringRule) Compile() (RuleEvaluator, error) {
	re, err := regexp.Compile(c.Pattern)
	if err != nil {
		return nil, errors.Annotate(err, "parsing pattern").Err()
	}

	// Segments defines portions of the output LIKE expression,
	// which are either literal text found in the LikeTemplate,
	// or parts of the test name matched by Pattern.
	var segments []segment

	// The exclusive upper bound we have created segments for.
	lastIndex := 0

	// Analyze the specified LikeTemplate to identify the
	// location of all substitution expressions (of the form ${name})
	// and iterate through them.
	matches := substitutionRE.FindAllStringSubmatchIndex(c.LikeTemplate, -1)
	for _, match := range matches {
		// The start and end of the substitution expression (of the form ${name})
		// in c.LikeTemplate.
		matchStart := match[0]
		matchEnd := match[1]

		if matchStart > lastIndex {
			// There is some literal text between the start of the LikeTemplate
			// and the first substitution expression, or the last substitution
			// expression and the current one. This is literal
			// text that should be included in the output directly.
			literalText := c.LikeTemplate[lastIndex:matchStart]
			if err := lang.ValidateLikePattern(literalText); err != nil {
				return nil, errors.Annotate(err, "%q is not a valid standalone LIKE expression", literalText).Err()
			}
			segments = append(segments, &literalSegment{
				value: literalText,
			})
		}

		matchString := c.LikeTemplate[match[0]:match[1]]
		if matchString == "$" {
			return nil, fmt.Errorf("invalid use of the $ operator at position %v in %q ('$' not followed by '{name}' or '$'), "+
				"if you meant to include a literal $ character, please use $$", match[0], c.LikeTemplate)
		}
		if matchString == "$$" {
			// Insert the literal "$" into the output.
			segments = append(segments, &literalSegment{
				value: "$",
			})
		} else {
			// The name of the capture group that should be substituted at
			// the current position.
			name := c.LikeTemplate[match[2]:match[3]]

			// Find the index of the corresponding capture group in the
			// Pattern.
			submatchIndex := -1
			for i, submatchName := range re.SubexpNames() {
				if submatchName == "" {
					// Unnamed capturing groups can not be referred to.
					continue
				}
				if submatchName == name {
					submatchIndex = i
					break
				}
			}
			if submatchIndex == -1 {
				return nil, fmt.Errorf("like template contains reference to non-existant capturing group with name %q", name)
			}

			// Indicate we should include the value of that capture group
			// in the output.
			segments = append(segments, &submatchSegment{
				submatchIndex: submatchIndex,
			})
		}
		lastIndex = matchEnd
	}

	if lastIndex < len(c.LikeTemplate) {
		literalText := c.LikeTemplate[lastIndex:len(c.LikeTemplate)]
		if err := lang.ValidateLikePattern(literalText); err != nil {
			return nil, errors.Annotate(err, "%q is not a valid standalone LIKE expression", literalText).Err()
		}
		// Some text after all substitution expressions. This is literal
		// text that should be included in the output directly.
		segments = append(segments, &literalSegment{
			value: literalText,
		})
	}

	// Produce the evaluator. This is in the hot-path that is run
	// on every test result on every ingestion or config change,
	// so it should be fast. We do not want to be parsing regular
	// expressions or templates in here.
	evaluator := func(testName string) (like string, ok bool) {
		m := re.FindStringSubmatch(testName)
		if m == nil {
			return "", false
		}
		segmentValues := make([]string, len(segments))
		for i, s := range segments {
			segmentValues[i] = s.evaluate(m)
		}
		return strings.Join(segmentValues, ""), true
	}
	return evaluator, nil
}

// literalSegment is a part of a constructed string
// that is a constant string value.
type literalSegment struct {
	// The literal value that defines this segment.
	value string
}

func (c *literalSegment) evaluate(matches []string) string {
	return c.value
}

// submatchSegment is a part of a constructed string
// that is populated with a matched portion of
// another source string.
type submatchSegment struct {
	// The source string submatch index that defines this segment.
	submatchIndex int
}

func (m *submatchSegment) evaluate(matches []string) string {
	return likeRewriter.Replace(matches[m.submatchIndex])
}

// segment represents a part of a constructed string.
type segment interface {
	evaluate(matches []string) string
}

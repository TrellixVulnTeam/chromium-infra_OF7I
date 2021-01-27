package analyzer

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"infra/appengine/sheriff-o-matic/som/analyzer/step"
	"infra/appengine/sheriff-o-matic/som/client"
	"infra/appengine/sheriff-o-matic/som/model"
	"infra/monitoring/messages"
	"io/ioutil"
	"net/url"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/gae/service/info"
	"go.chromium.org/luci/gae/service/memcache"
	"go.chromium.org/luci/grpc/grpcutil"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const bqMemcacheFormat = "bq-%s"

const selectFrom = `
SELECT
  Project,
  Bucket,
  Builder,
  BuilderGroup,
  StepName,
  TestNamesFingerprint,
  TestNamesTrunc,
  NumTests,
  BuildIdBegin,
  BuildIdEnd,
  BuildNumberBegin,
  BuildNumberEnd,
  CPRangeOutputBegin,
  CPRangeOutputEnd,
  CPRangeInputBegin,
  CPRangeInputEnd,
  CulpritIdRangeBegin,
  CulpritIdRangeEnd,
  StartTime,
  BuildStatus
FROM
	` + "`%s.%s.sheriffable_failures`"

const selectFromWhere = selectFrom + `
WHERE
`

// TODO (nqmtuan): Filter the critical for other projects
// But firstly make sure it is working with chrome os first
const crosFailuresQuery = selectFromWhere + `
	project = "chromeos"
	AND bucket IN ("postsubmit")
	AND (critical != "NO" OR critical is NULL)
`

const fuchsiaFailuresQuery = selectFromWhere + `
	Project = %q
	AND Bucket = "global.roller"
	AND Builder NOT LIKE "%%bisect%%"
LIMIT
	1000
`

var androidFilterFunc = func(r failureRow) bool {
	if r.Project != "chrome" && r.Project != "chromium" {
		return false
	}
	excludedBuckets := []string{"try", "findit"}
	if sliceContains(excludedBuckets, r.Bucket) {
		return false
	}
	builderGroup := r.BuilderGroup.String()
	if sliceContains([]string{"internal.client.clank", "internal.client.clank_tot", "chromium.android"}, builderGroup) {
		return true
	}
	if builderGroup == "chromium" && r.Builder == "Android" {
		return true
	}
	if builderGroup == "chromium.webkit" && sliceContains([]string{"Android Builder", "Webkit Android (Nexus4)"}, r.Builder) {
		return true
	}
	validBuilders := []string{
		"android-arm-official-tests",
		"android-arm64-official-tests",
		"android-arm-beta-tests",
		"android-arm64-beta-tests",
		"android-arm-stable-tests",
		"android-arm64-stable-tests",
		"android-arm-beta",
		"android-arm64-beta",
		"android-arm64-stable",
		"android-arm64-stable",
		"android-arm-stable",
		"android-asan",
	}
	return sliceContains(validBuilders, r.Builder)
}

var chromiumFilterFunc = func(r failureRow) bool {
	if r.Project != "chrome" && r.Project != "chromium" {
		return false
	}
	builderGroup := r.BuilderGroup.String()
	validBuilderGroups := []string{
		"chrome",
		"chromium",
		"chromium.chromiumos",
		"chromium.linux",
		"chromium.mac",
		"chromium.memory",
		"chromium.win",
	}
	return sliceContains(validBuilderGroups, builderGroup) && r.Bucket == "ci"
}

var chromiumGPUFilterFunc = func(r failureRow) bool {
	if r.Project != "chrome" && r.Project != "chromium" {
		return false
	}
	if r.Bucket == "findit" {
		return false
	}
	validBuilderGroups := []string{
		"chromium.gpu",
		"chromium.gpu.fyi",
		"chromium.swangle",
	}
	return sliceContains(validBuilderGroups, r.BuilderGroup.String())
}

var chromiumPerfFilterFunc = func(r failureRow) bool {
	if r.Project != "chrome" && r.Project != "chromium" {
		return false
	}
	if strings.Contains(r.Builder, "bisect") {
		return false
	}
	excludedBuckets := []string{"try", "cq", "staging", "general", "findit"}
	if sliceContains(excludedBuckets, r.Bucket) {
		return false
	}
	return (r.Project == "chromium.perf" || r.BuilderGroup.String() == "chromium.perf") && (!r.BuilderGroup.Valid || !strings.HasSuffix(r.BuilderGroup.String(), ".fyi"))
}

var iosFilterFunc = func(r failureRow) bool {
	if r.Project != "chrome" && r.Project != "chromium" {
		return false
	}
	if r.Bucket == "findit" {
		return false
	}
	if r.Project == "chrome" && r.BuilderGroup.String() == "internal.bling.main" {
		return true
	}
	validBuilders := []string{
		"ios-device",
		"ios-simulator",
		"ios-simulator-full-configs",
		"ios-simulator-noncq",
	}
	if r.Project == "chromium" && r.BuilderGroup.String() == "chromium.mac" && sliceContains(validBuilders, r.Builder) {
		return true
	}
	return false
}

var chromeBrowserReleaseFilterFunc = func(r failureRow) bool {
	// TODO (nqmtuan): Remove the first part of the OR condition when we are post m85
	return (r.Project == "chromium" && strings.HasPrefix(r.Bucket, "ci-m")) || (strings.HasPrefix(r.Project, "chromium-m") && r.Bucket == "ci")
}

var chromiumClangFilterFunc = func(r failureRow) bool {
	if strings.Contains(r.Builder, "bisect") {
		return false
	}
	excludedBuckets := []string{"try", "cq", "staging", "general", "findit"}
	if sliceContains(excludedBuckets, r.Bucket) {
		return false
	}
	return (r.Project == "chromium.clang" || r.BuilderGroup.String() == "chromium.clang") && (!r.BuilderGroup.Valid || !strings.HasSuffix(r.BuilderGroup.String(), ".fyi"))
}

type failureRow struct {
	TestNamesFingerprint bigquery.NullInt64
	TestNamesTrunc       bigquery.NullString
	NumTests             bigquery.NullInt64
	StepName             string
	BuilderGroup         bigquery.NullString
	Builder              string
	Bucket               string
	Project              string
	BuildIDBegin         bigquery.NullInt64
	BuildIDEnd           bigquery.NullInt64
	BuildNumberBegin     bigquery.NullInt64
	BuildNumberEnd       bigquery.NullInt64
	CPRangeInputBegin    *GitCommit
	CPRangeInputEnd      *GitCommit
	CPRangeOutputBegin   *GitCommit
	CPRangeOutputEnd     *GitCommit
	CulpritIDRangeBegin  bigquery.NullInt64
	CulpritIDRangeEnd    bigquery.NullInt64
	StartTime            bigquery.NullTimestamp
	BuildStatus          string
}

// GitCommit represents a struct column for BQ query results.
type GitCommit struct {
	Project  bigquery.NullString
	Ref      bigquery.NullString
	Host     bigquery.NullString
	ID       bigquery.NullString
	Position bigquery.NullInt64
}

type bqBucket struct {
	Project string
	Bucket  string
}

type bqBuilder struct {
	Project string
	Bucket  string
	Builder string
}

// This type is a catch-all for every kind of failure. In a better,
// simpler design we wouldn't have to use this but it's here to make
// the transition from previous analyzer logic easier.
type bqFailure struct {
	Name            string `json:"step"`
	kind            string
	severity        messages.Severity
	Tests           []step.TestWithResult `json:"tests"`
	NumFailingTests int64                 `json:"num_failing_tests"`
}

func (b *bqFailure) Signature() string {
	return b.Name
}

func (b *bqFailure) Kind() string {
	return b.kind
}

func (b *bqFailure) Severity() messages.Severity {
	return b.severity
}

func (b *bqFailure) Title(bses []*messages.BuildStep) string {
	f := bses[0]
	prefix := fmt.Sprintf("%s failing", f.Step.Name)

	if b.NumFailingTests > 0 {
		prefix = fmt.Sprintf("%s (%d tests)", prefix, b.NumFailingTests)
	}

	if len(bses) == 1 {
		return fmt.Sprintf("%s on %s/%s", prefix, f.BuilderGroup.Name(), f.Build.BuilderName)
	}

	return fmt.Sprintf("%s on multiple builders", prefix)
}

// Generates the BigQuery SQL for the specified tree against the app environment specified
// in appID (ex: "sheriff-o-matic-staging" vs. "sheriff-o-matic").
func generateSQLQuery(ctx context.Context, tree string, appID string) (string, error) {
	bbProjectFilter := getBuildBucketProjectFilterFromTree(ctx, tree)
	if tree == "chromeos" {
		return fmt.Sprintf(crosFailuresQuery, appID, "chromeos"), nil
	}
	if tree == "fuchsia" {
		return fmt.Sprintf(fuchsiaFailuresQuery, appID, "fuchsia", bbProjectFilter), nil
	}
	return "", fmt.Errorf("invalid tree %q", tree)
}

// GetBigQueryAlerts generates alerts for currently failing build steps, using
// BigQuery to do most of the heavy lifting.
// Note that this returns alerts for all failing steps, so filtering should
// be applied on the return value.
// TODO: Some post-bq result merging with heuristics:
//   - Merge alerts for sets of multiple failing steps. Currently will return one alert
//     for each failing step on a builder. If step_a and step_b are failing on the same
//     builder or set of builders, they should be merged into a single alert.
func GetBigQueryAlerts(ctx context.Context, tree string) ([]*messages.BuildFailure, error) {
	var failureRows []failureRow
	var err error
	if shouldUseCache(tree) {
		failureRows, err = getFailureRowsForTree(ctx, tree)
		if err != nil {
			return nil, err
		}
	} else {
		appID := getAppID(ctx)
		queryStr, err := generateSQLQuery(ctx, tree, appID)
		if err != nil {
			return nil, err
		}
		failureRows, err = getFailureRowsForQuery(ctx, queryStr)
		if err != nil {
			return nil, err
		}
	}

	// TODO (nqmtuan): Remove this condition once crbug.com/1152170 is fixed.
	if tree != "fuchsia" {
		filteredRows, err := filterDeletedBuilders(ctx, failureRows)
		if err != nil {
			// Probably something is wrong with buildbucket, but this is not critical.
			// We should log the error and proceed.
			logging.Errorf(ctx, "error when filtering deleted builders: %v", err)
		} else {
			failureRows = filteredRows
		}
	}
	return processBQResults(ctx, failureRows)
}

func shouldUseCache(tree string) bool {
	trees := []string{"android", "chromium", "chromium.gpu", "ios", "chromium.perf", "chrome_browser_release", "chromium.clang"}
	return sliceContains(trees, tree)
}

func getFilterFuncForTree(tree string) (func(failureRow) bool, error) {
	switch tree {
	case "android":
		return androidFilterFunc, nil
	case "chromium":
		return chromiumFilterFunc, nil
	case "chromium.gpu":
		return chromiumGPUFilterFunc, nil
	case "chromium.perf":
		return chromiumPerfFilterFunc, nil
	case "ios":
		return iosFilterFunc, nil
	case "chrome_browser_release":
		return chromeBrowserReleaseFilterFunc, nil
	case "chromium.clang":
		return chromiumClangFilterFunc, nil
	default:
		return nil, fmt.Errorf("could not find filter function for tree %s", tree)
	}
}

func getFailureRowsForTree(ctx context.Context, tree string) ([]failureRow, error) {
	allFailureRows, err := retrieveFailureRowsFromMemcache(ctx, "chrome")
	if err != nil {
		return nil, err
	}
	filterFunc, err := getFilterFuncForTree(tree)
	if err != nil {
		return nil, err
	}
	failureRows := []failureRow{}
	for _, row := range allFailureRows {
		if filterFunc(row) {
			failureRows = append(failureRows, row)
		}
	}
	return failureRows, nil
}

func generateQueryForProject(appID string, project string) string {
	return fmt.Sprintf(selectFrom, appID, project)
}

// QueryBQForProject queries the sheriffable_failures for a project and stores the result in memcache.
func QueryBQForProject(ctx context.Context, project string) error {
	appID := getAppID(ctx)
	queryStr := generateQueryForProject(appID, project)
	failureRows, err := getFailureRowsForQuery(ctx, queryStr)
	if err != nil {
		return err
	}
	return storeFailureRowsToMemcache(ctx, failureRows, project)
}

func getFailureRowsForQuery(ctx context.Context, queryStr string) ([]failureRow, error) {
	failureRows := []failureRow{}
	appID := getAppID(ctx)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	client, err := bigquery.NewClient(ctx, appID)
	if err != nil {
		return nil, err
	}

	logging.Infof(ctx, "query: %s", queryStr)
	q := client.Query(queryStr)
	it, err := q.Read(ctx)
	if err != nil {
		return failureRows, err
	}

	for {
		var r failureRow
		err := it.Next(&r)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return failureRows, err
		}
		failureRows = append(failureRows, r)
	}
	return failureRows, nil
}

func getAppID(ctx context.Context) string {
	appID := info.AppID(ctx)
	logging.Infof(ctx, "app_id: %s", appID)
	if appID == "None" {
		appID = "sheriff-o-matic-staging"
	}
	return appID
}

// Queries BuildBucketProjectFilter from datastore on specified tree.
// Return the treeName if BuildBucketProjectFilter is not set.
func getBuildBucketProjectFilterFromTree(c context.Context, treeName string) string {
	q := datastore.NewQuery("Tree")
	trees := []*model.Tree{}
	if err := datastore.GetAll(c, q, &trees); err == nil {
		for _, tree := range trees {
			if tree.Name == treeName && tree.BuildBucketProjectFilter != "" {
				return strings.TrimSpace(tree.BuildBucketProjectFilter)
			}
		}
	}
	return treeName
}

func generateBuilderURL(project string, bucket string, builderName string) string {
	return fmt.Sprintf("https://ci.chromium.org/p/%s/builders/%s/%s", project, bucket, url.PathEscape(builderName))
}

func generateBuildURL(project string, bucket string, builderName string, buildID bigquery.NullInt64) string {
	if buildID.Valid {
		return fmt.Sprintf("%s/b%d", generateBuilderURL(project, bucket, builderName), buildID.Int64)
	}
	return "" // Go does not allow null value :(
}

func processBQResults(ctx context.Context, failureRows []failureRow) ([]*messages.BuildFailure, error) {
	ret := []*messages.BuildFailure{}
	for _, r := range failureRows {
		ab := generateAlertedBuilder(ctx, r)
		regressionRanges := getRegressionRanges(ab.LatestPassingRev, ab.FirstFailingRev)

		// Process tests.
		reason := &bqFailure{
			Name:     r.StepName,
			kind:     "basic",
			severity: messages.ReliableFailure,
		}
		if r.TestNamesFingerprint.Valid {
			reason.kind = "test"
			testNames := strings.Split(r.TestNamesTrunc.StringVal, "\n")
			sort.Strings(testNames)
			for _, testName := range testNames {
				reason.Tests = append(reason.Tests, step.TestWithResult{
					TestName: testName,
				})
			}
			reason.NumFailingTests = r.NumTests.Int64
		}

		bf := &messages.BuildFailure{
			StepAtFault: &messages.BuildStep{
				Step: &messages.Step{
					Name: r.StepName,
				},
			},
			Builders: []*messages.AlertedBuilder{ab},
			Reason: &messages.Reason{
				Raw: reason,
			},
			RegressionRanges: regressionRanges,
		}
		ret = append(ret, bf)
	}

	ret = filterHierarchicalSteps(ret)
	return ret, nil
}

func generateAlertedBuilder(ctx context.Context, r failureRow) *messages.AlertedBuilder {
	gitBegin := r.CPRangeOutputBegin
	if gitBegin == nil {
		gitBegin = r.CPRangeInputBegin
	}
	gitEnd := r.CPRangeOutputEnd
	if gitEnd == nil {
		gitEnd = r.CPRangeInputEnd
	}
	var latestPassingRev, firstFailingRev *messages.RevisionSummary
	if gitBegin != nil {
		latestPassingRev = &messages.RevisionSummary{
			Position: int(gitBegin.Position.Int64),
			Branch:   gitBegin.Ref.StringVal,
			Host:     gitBegin.Host.StringVal,
			Repo:     gitBegin.Project.StringVal,
			GitHash:  gitBegin.ID.StringVal,
		}
	}
	if gitEnd != nil {
		firstFailingRev = &messages.RevisionSummary{
			Position: int(gitEnd.Position.Int64),
			Branch:   gitEnd.Ref.StringVal,
			Host:     gitEnd.Host.StringVal,
			Repo:     gitEnd.Project.StringVal,
			GitHash:  gitEnd.ID.StringVal,
		}
	}
	return &messages.AlertedBuilder{
		Project:                  r.Project,
		Bucket:                   r.Bucket,
		Name:                     r.Builder,
		BuilderGroup:             r.BuilderGroup.StringVal,
		FirstFailure:             r.BuildIDBegin.Int64,
		LatestFailure:            r.BuildIDEnd.Int64,
		FirstFailureBuildNumber:  r.BuildNumberBegin.Int64,
		LatestFailureBuildNumber: r.BuildNumberEnd.Int64,
		URL:                      generateBuilderURL(r.Project, r.Bucket, r.Builder),
		FirstFailureURL:          generateBuildURL(r.Project, r.Bucket, r.Builder, r.BuildIDBegin),
		LatestFailureURL:         generateBuildURL(r.Project, r.Bucket, r.Builder, r.BuildIDEnd),
		LatestPassingRev:         latestPassingRev,
		FirstFailingRev:          firstFailingRev,
		NumFailingTests:          r.NumTests.Int64,
		BuildStatus:              r.BuildStatus,
	}
}

func getRegressionRanges(earliestRev, latestRev *messages.RevisionSummary) []*messages.RegressionRange {
	regressionRanges := []*messages.RegressionRange{}
	if latestRev != nil && earliestRev != nil {
		regRange := &messages.RegressionRange{
			Repo: earliestRev.Repo,
			Host: earliestRev.Host,
		}
		if earliestRev.GitHash != "" && latestRev.GitHash != "" {
			regRange.Revisions = []string{earliestRev.GitHash, latestRev.GitHash}
		}
		if earliestRev.Position != 0 && latestRev.Position != 0 {
			regRange.Positions = []string{
				fmt.Sprintf("%s@{#%d}", earliestRev.Branch, earliestRev.Position),
				fmt.Sprintf("%s@{#%d}", latestRev.Branch, latestRev.Position),
			}
		}
		regressionRanges = append(regressionRanges, regRange)
	}
	return regressionRanges
}

func builderKey(b *messages.AlertedBuilder) string {
	return fmt.Sprintf("%s/%s/%s", b.Project, b.Bucket, b.Name)
}

func filterHierarchicalSteps(failures []*messages.BuildFailure) []*messages.BuildFailure {
	ret := []*messages.BuildFailure{}
	// First group failures by builder.
	failuresByBuilder := map[string][]*messages.BuildFailure{}
	builders := map[string]*messages.AlertedBuilder{}
	for _, f := range failures {
		for _, b := range f.Builders {
			key := builderKey(b)
			builders[key] = b
			if _, ok := failuresByBuilder[key]; !ok {
				failuresByBuilder[key] = []*messages.BuildFailure{}
			}
			failuresByBuilder[key] = append(failuresByBuilder[key], f)
		}
	}

	filteredFailuresByBuilder := map[string]stringset.Set{}

	// For each builder, sort failing steps.
	for key, failures := range failuresByBuilder {
		sort.Sort(byStepName(failures))
		filteredFailures := stringset.New(0)
		// For each step in builder steps, if it's a prefix of the one after it,
		// ignore that step.
		for i, step := range failures {
			if i <= len(failures)-2 {
				nextStep := failures[i+1]
				if strings.HasPrefix(nextStep.StepAtFault.Step.Name, step.StepAtFault.Step.Name+"|") {
					// Skip this step since it has at least one child.
					continue
				}
			}
			filteredFailures.Add(step.StepAtFault.Step.Name)
		}
		filteredFailuresByBuilder[key] = filteredFailures
	}

	// Now filter out BuildFailures whose StepAtFault has been filtered out for
	// that builder.
	for _, failure := range failures {
		filteredBuilders := []*messages.AlertedBuilder{}
		for _, b := range failure.Builders {
			key := builderKey(b)
			filtered := filteredFailuresByBuilder[key]
			if filtered.Has(failure.StepAtFault.Step.Name) {
				filteredBuilders = append(filteredBuilders, b)
			}
		}
		if len(filteredBuilders) > 0 {
			failure.Builders = filteredBuilders
			ret = append(ret, failure)
		}
	}

	return ret
}

// TODO(seanmccullough): rename if we aren't sorting by step name, which may
// not be the most robust sorting method. Check if Step.Number is always
// populated, though that may not translate well because multiple builders are
// grouped by failing step and the same "step" may occur at different
// indexes in different builders.
type byStepName []*messages.BuildFailure

func (a byStepName) Len() int      { return len(a) }
func (a byStepName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byStepName) Less(i, j int) bool {
	return a[i].StepAtFault.Step.Name < a[j].StepAtFault.Step.Name
}

func sliceContains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if needle == item {
			return true
		}
	}
	return false
}

func storeFailureRowsToMemcache(ctx context.Context, failureRows []failureRow, project string) error {
	data, err := json.Marshal(failureRows)
	if err != nil {
		return err
	}
	val, err := zipData(data)
	if err != nil {
		return err
	}

	item := memcache.NewItem(ctx, fmt.Sprintf(bqMemcacheFormat, project)).SetValue(val)
	return memcache.Set(ctx, item)
}

func retrieveFailureRowsFromMemcache(ctx context.Context, project string) ([]failureRow, error) {
	failureRows := []failureRow{}
	// Read from memcache
	key := fmt.Sprintf(bqMemcacheFormat, project)
	item, err := memcache.GetKey(ctx, key)
	if err != nil {
		return nil, err
	}

	// Decompress using zlib.
	val, err := unzipData(item.Value())
	if err != nil {
		return nil, err
	}

	// Convert from JSON.
	if err := json.Unmarshal(val, &failureRows); err != nil {
		return nil, err
	}
	return failureRows, nil
}

func zipData(data []byte) ([]byte, error) {
	b := bytes.Buffer{}
	w := zlib.NewWriter(&b)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func unzipData(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var unzippedData []byte
	if unzippedData, err = ioutil.ReadAll(r); err != nil {
		return nil, err
	}
	if err := r.Close(); err != nil {
		return nil, err
	}
	return unzippedData, nil
}

func getBucketList(c context.Context, failureRows []failureRow) []*bqBucket {
	seen := map[bqBucket]bool{}
	result := []*bqBucket{}
	for _, r := range failureRows {
		b := bqBucket{
			Project: r.Project,
			Bucket:  r.Bucket,
		}
		if _, ok := seen[b]; !ok {
			seen[b] = true
			result = append(result, &b)
		}
	}
	return result
}

func getAvailableBuilders(c context.Context, cl client.BBBuildersClient, buckets []*bqBucket) ([]*buildbucketpb.BuilderItem, error) {
	result := []*buildbucketpb.BuilderItem{}
	for _, b := range buckets {
		builders, err := client.ListBuildersByBucket(c, cl, b.Project, b.Bucket)
		if err != nil {
			// Bucket not found, maybe it is deleted?
			if grpcutil.Code(err) == codes.NotFound {
				logging.Infof(c, "Bucket %v not found: %v", b, err)
				continue
			}
			return nil, err
		}
		result = append(result, builders...)
	}
	return result, nil
}

func filterDeletedBuildersWithClient(c context.Context, cl client.BBBuildersClient, failureRows []failureRow) ([]failureRow, error) {
	buckets := getBucketList(c, failureRows)
	builders, err := getAvailableBuilders(c, cl, buckets)
	if err != nil {
		return nil, err
	}
	logging.Infof(c, "There are %d available builders", len(builders))
	builderMap := map[bqBuilder]bool{}
	for _, b := range builders {
		bqb := bqBuilder{
			Project: b.Id.Project,
			Bucket:  b.Id.Bucket,
			Builder: b.Id.Builder,
		}
		builderMap[bqb] = true
	}

	result := []failureRow{}
	for i, row := range failureRows {
		bqb := bqBuilder{
			Project: row.Project,
			Bucket:  row.Bucket,
			Builder: row.Builder,
		}
		if _, ok := builderMap[bqb]; ok {
			result = append(result, failureRows[i])
		} else {
			logging.Debugf(c, "Builder %+v has been deleted", bqb)
		}
	}
	logging.Infof(c, "Before filtering deleted builders, there are %d failures", len(failureRows))
	logging.Infof(c, "After filtering deleted builders, there are %d failures", len(result))
	return result, nil
}

func filterDeletedBuilders(c context.Context, failureRows []failureRow) ([]failureRow, error) {
	cl, err := client.BuildersClient(c)
	if err != nil {
		return nil, err
	}
	return filterDeletedBuildersWithClient(c, cl, failureRows)
}

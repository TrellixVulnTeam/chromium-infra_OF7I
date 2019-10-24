// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.chromium.org/luci/common/data/stringset"

	tricium "infra/tricium/api/v1"
)

const (
	category            = "Metrics"
	dateFormat          = "2006-01-02"
	dateMilestoneFormat = "2006-01-02T15:04:05"
	histogramEndTag     = "</histogram>"
	obsoleteStartTag    = "<obsolete"
	ownerStartTag       = "<owner"

	oneOwnerError            = `[WARNING] It's a best practice to list multiple owners, so that there's no single point of failure for communication: https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Owners.`
	firstOwnerTeamError      = `[WARNING] Please list an individual as the primary owner for this metric: https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Owners.`
	oneOwnerTeamError        = `[WARNING] Please list an individual as the primary owner for this metric. Please also ensure to list multiple owners, so there's no single point of failure for communication: https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Owners.`
	noExpiryError            = `[ERROR] Please specify an expiry condition for this histogram: https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Histogram-Expiry.`
	badExpiryError           = `[ERROR] Could not parse histogram expiry. Please format as YYYY-MM-DD or MXX: https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Histogram-Expiry.`
	pastExpiryWarning        = `[WARNING] This expiry date is in the past. Did you mean to set an expiry date in the future?`
	farExpiryWarning         = `[WARNING] It's a best practice to choose an expiry that is at most one year out: https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Histogram-Expiry.`
	neverExpiryInfo          = `[INFO] The expiry should only be set to "never" in rare cases. Please double-check that this use of "never" is appropriate: https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Histogram-Expiry.`
	neverExpiryError         = `[ERROR] The expiry should only be set to "never" in rare cases. If you believe this use of "never" is appropriate, you must include an XML comment describing why, such as <!-- expires-never: "heartbeat" metric (internal: go/uma-heartbeats) -->: https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Histogram-Expiry.`
	milestoneFailure         = `[WARNING] Tricium failed to fetch milestone branch date. Please double-check that this milestone is correct, because the tool is currently not able to check for you.`
	obsoleteDateError        = `[WARNING] When marking a histogram as <obsolete>, please document when the histogram was removed, either as a date including a 2-digit month and 4-digit year, or a milestone in MXX format.`
	unitsMicrosecondsWarning = `[WARNING] Histograms with units="microseconds" should document whether the metrics is reported for all users or only users with high-resolution clocks. Note that reports from clients with low-resolution clocks (i.e. on Windows, ref. TimeTicks::IsHighResolution()) may cause the metric to have an abnormal distribution.`
	removedHistogramError    = `[ERROR] Do not delete histograms from histograms.xml. Instead, mark unused histograms as obsolete and annotate them with the date or milestone in the <obsolete> tag entry: https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Cleaning-Up-Histogram-Entries.`
	addedNamespaceWarning    = `[WARNING] Are you sure you want to add the namespace %s to histograms.xml? For most new histograms, it's appropriate to re-use one of the existing top-level histogram namespaces. For histogram names, the namespace is defined as everything preceding the first dot '.' in the name.`
	singleElementEnumWarning = `[WARNING] It looks like this is an enumerated histogram that contains only a single bucket. UMA metrics are difficult to interpret in isolation, so please either add one or more additional buckets that can serve as a baseline for comparison, or document what other metric should be used as a baseline during analysis. https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#enum-histograms.`
)

var (
	// We need a pattern for matching the histogram start tag because
	// there are other tags that share the "histogram" prefix like "histogram-suffixes"
	histogramStartPattern     = regexp.MustCompile(`^<histogram($|\s|>)`)
	neverExpiryCommentPattern = regexp.MustCompile(`^<!--\s?expires-never`)
	// Match date patterns of format YYYY-MM-DD.
	expiryDatePattern      = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])$`)
	expiryMilestonePattern = regexp.MustCompile(`^M([0-9]{2,3})$`)
	// Match years between 1970 and 2999.
	obsoleteYearPattern = regexp.MustCompile(`19[7-9][0-9]|2([0-9]{3})`)
	// Match double-digit or spelled-out months.
	obsoleteMonthPattern     = regexp.MustCompile(`([^0-9](0[1-9]|10|11|12)[^0-9])|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec`)
	obsoleteMilestonePattern = regexp.MustCompile(`M([0-9]{2,3})`)
	// Match valid summaries for histograms with units=microseconds.
	microsecondsSummary = regexp.MustCompile(`all\suser|(high|low)(\s|-)resolution`)

	// Now is an alias for time.Now, can be overwritten by tests.
	now              = time.Now
	getMilestoneDate = getMilestoneDateImpl
)

// histogram contains all info about a UMA histogram.
type histogram struct {
	Name     string   `xml:"name,attr"`
	Enum     string   `xml:"enum,attr"`
	Units    string   `xml:"units,attr"`
	Expiry   string   `xml:"expires_after,attr"`
	Obsolete string   `xml:"obsolete"`
	Owners   []string `xml:"owner"`
	Summary  string   `xml:"summary"`
}

// metadata contains metadata about histogram tags and required comments.
type metadata struct {
	HistogramLineNum      int
	OwnerLineNum          int
	ObsoleteLineNum       int
	HasNeverExpiryComment bool
	HistogramBytes        []byte
}

// milestone contains the branch point date of a particular milestone.
type milestone struct {
	Milestone int    `json:"mstone"`
	Date      string `json:"branch_point"`
}

type milestones struct {
	Milestones []milestone `json:"mstones"`
}

type changeMode int

const (
	// ADDED means a line was modified or added to a file.
	ADDED changeMode = iota
	// REMOVED means a line was removed from a file.
	REMOVED
)

func analyzeFile(filePath, inputDir, prevDir string, filesChanged *diffsPerFile, singletonEnums stringset.Set) []*tricium.Data_Comment {
	log.Printf("ANALYZING File: %s", filePath)
	var allComments []*tricium.Data_Comment
	inputPath := filepath.Join(inputDir, filePath)
	f := openFileOrDie(inputPath)
	defer closeFileOrDie(f)
	// Analyze added lines in file (if any).
	comments, addedHistograms, newNamespaces, namespaceLineNums := analyzeChangedLines(bufio.NewScanner(f), inputPath, filesChanged.addedLines[filePath], singletonEnums, ADDED)
	allComments = append(allComments, comments...)
	// Analyze removed lines in file (if any).
	tempPath := filepath.Join(prevDir, filePath)
	oldFile := openFileOrDie(tempPath)
	defer closeFileOrDie(oldFile)
	var emptySet stringset.Set
	_, removedHistograms, oldNamespaces, _ := analyzeChangedLines(bufio.NewScanner(oldFile), tempPath, filesChanged.removedLines[filePath], emptySet, REMOVED)
	// Identify if any histograms were removed
	allComments = append(allComments, findRemovedHistograms(inputPath, addedHistograms, removedHistograms)...)
	allComments = append(allComments, findAddedNamespaces(inputPath, newNamespaces, oldNamespaces, namespaceLineNums)...)
	return allComments
}

func analyzeChangedLines(scanner *bufio.Scanner, path string, linesChanged []int, singletonEnums stringset.Set, mode changeMode) ([]*tricium.Data_Comment, stringset.Set, stringset.Set, map[string]int) {
	var comments []*tricium.Data_Comment
	// meta is a struct that holds line numbers of different tags in histogram.
	var meta *metadata
	// currHistogram is a buffer that holds the current histogram.
	var currHistogram []byte
	// histogramStart is the starting line number for the current histogram.
	var histogramStart int
	// If any line in the histogram showed up as an added or removed line in the diff.
	var histogramChanged bool
	changedHistograms := make(stringset.Set)
	namespaces := make(stringset.Set)
	namespaceLineNums := make(map[string]int)
	lineNum := 1
	changedIndex := 0
	for scanner.Scan() {
		// Copying scanner.Scan() is necessary to ensure the scanner does not
		// overwrite the memory that stores currHistogram.
		newBytes := make([]byte, len(scanner.Bytes()))
		copy(newBytes, scanner.Bytes())
		if currHistogram != nil {
			// Add line to currHistogram if currently between some histogram tags.
			currHistogram = append(currHistogram, newBytes...)
		}
		line := strings.TrimSpace(scanner.Text())
		if histogramStartPattern.MatchString(line) {
			// Initialize currHistogram and metadata when a new histogram is encountered.
			histogramStart = lineNum
			meta = newMetadata(histogramStart)
			currHistogram = newBytes
			histogramChanged = false
		} else if strings.HasPrefix(line, histogramEndTag) {
			// Analyze entire histogram after histogram end tag is encountered.
			hist := bytesToHistogram(currHistogram, meta)
			namespace := strings.SplitN(hist.Name, ".", 2)[0]
			namespaces.Add(namespace)
			if namespaceLineNums[namespace] == 0 {
				namespaceLineNums[namespace] = meta.HistogramLineNum
			}
			if histogramChanged {
				changedHistograms.Add(hist.Name)
				// Only check new (added) hists are correct.
				if mode == ADDED {
					comments = append(comments, checkHistogram(path, hist, meta, singletonEnums)...)
				}
			}
			currHistogram = nil
		} else if strings.HasPrefix(line, ownerStartTag) {
			if meta.OwnerLineNum == histogramStart {
				meta.OwnerLineNum = lineNum
			}
		} else if strings.HasPrefix(line, obsoleteStartTag) {
			meta.ObsoleteLineNum = lineNum
		} else if neverExpiryCommentPattern.MatchString(line) {
			meta.HasNeverExpiryComment = true
		}
		if changedIndex < len(linesChanged) && lineNum == linesChanged[changedIndex] {
			histogramChanged = true
			changedIndex++
		}
		lineNum++
	}
	return comments, changedHistograms, namespaces, namespaceLineNums
}

func checkHistogram(path string, hist *histogram, meta *metadata, singletonEnums stringset.Set) []*tricium.Data_Comment {
	var comments []*tricium.Data_Comment
	if comment := checkOwners(path, hist, meta); comment != nil {
		comments = append(comments, comment)
	}
	if comment := checkUnits(path, hist, meta); comment != nil {
		comments = append(comments, comment)
	}
	if comment := checkObsolete(path, hist, meta); comment != nil {
		comments = append(comments, comment)
	}
	if comment := checkEnums(path, hist, meta, singletonEnums); comment != nil {
		comments = append(comments, comment)
	}
	comments = append(comments, checkExpiry(path, hist, meta)...)
	return comments
}

func bytesToHistogram(histBytes []byte, meta *metadata) *histogram {
	var hist *histogram
	if err := xml.Unmarshal(histBytes, &hist); err != nil {
		log.Panicf("WARNING: Failed to unmarshal histogram at line %d", meta.HistogramLineNum)
	}
	return hist
}

func checkOwners(path string, hist *histogram, meta *metadata) *tricium.Data_Comment {
	var comment *tricium.Data_Comment
	// Check that there is more than 1 owner
	if len(hist.Owners) <= 1 {
		comment = createOwnerComment(oneOwnerError, path, meta)
		log.Printf("ADDING Comment for %s at line %d: %s", hist.Name, comment.StartLine, "[ERROR]: One Owner")
	}
	// Check first owner is a not a team or OWNERS file.
	if len(hist.Owners) > 0 && (strings.Contains(hist.Owners[0], "-") || strings.Contains(hist.Owners[0], "OWNERS")) {
		if comment != nil {
			comment.Message = oneOwnerTeamError
		} else {
			comment = createOwnerComment(firstOwnerTeamError, path, meta)
		}
		log.Printf("ADDING Comment for %s at line %d: %s", hist.Name, comment.StartLine, "[ERROR]: First Owner Team")
	}
	return comment
}

func createOwnerComment(message, path string, meta *metadata) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  category + "/Owners",
		Message:   message,
		Path:      path,
		StartLine: int32(meta.OwnerLineNum),
	}
}

func checkUnits(path string, hist *histogram, meta *metadata) *tricium.Data_Comment {
	if strings.Contains(hist.Units, "microseconds") && !microsecondsSummary.MatchString(hist.Summary) {
		comment := &tricium.Data_Comment{
			Category:  category + "/Units",
			Message:   unitsMicrosecondsWarning,
			Path:      path,
			StartLine: int32(meta.HistogramLineNum),
		}
		log.Printf("ADDING Comment for %s at line %d: %s", hist.Name, comment.StartLine, "[ERROR]: Units Microseconds Bad Summary")
		return comment
	}
	return nil
}

func checkObsolete(path string, hist *histogram, meta *metadata) *tricium.Data_Comment {
	if hist.Obsolete != "" &&
		!obsoleteMilestonePattern.MatchString(hist.Obsolete) &&
		!(obsoleteYearPattern.MatchString(hist.Obsolete) &&
			obsoleteMonthPattern.MatchString(hist.Obsolete)) {
		comment := &tricium.Data_Comment{
			Category:  category + "/Obsolete",
			Message:   obsoleteDateError,
			Path:      path,
			StartLine: int32(meta.ObsoleteLineNum),
		}
		log.Printf("ADDING Comment for %s at line %d: %s", hist.Name, comment.StartLine, "[ERROR]: Obsolete no date")
		return comment
	}
	return nil
}

func checkExpiry(path string, hist *histogram, meta *metadata) []*tricium.Data_Comment {
	var commentMessage string
	var logMessage string
	if expiry := hist.Expiry; expiry == "" {
		commentMessage = noExpiryError
		logMessage = "[ERROR]: No Expiry"
	} else if expiry == "never" {
		if !meta.HasNeverExpiryComment {
			commentMessage = neverExpiryError
			logMessage = "[ERROR]: Never Expiry, No Comment"
		} else {
			commentMessage = neverExpiryInfo
			logMessage = "[INFO]: Never Expiry"
		}
	} else {
		dateMatch := expiryDatePattern.MatchString(expiry)
		milestoneMatch := expiryMilestonePattern.MatchString(expiry)
		if dateMatch {
			inputDate, err := time.Parse(dateFormat, expiry)
			if err != nil {
				log.Panicf("Failed to parse expiry date: %v", err)
			}
			processExpiryDateDiff(inputDate, &commentMessage, &logMessage)
		} else if milestoneMatch {
			milestone, err := strconv.Atoi(expiry[1:])
			if err != nil {
				log.Panicf("Failed to convert input milestone to integer: %v", err)
			}
			milestoneDate, err := getMilestoneDate(milestone)
			if err != nil {
				commentMessage = milestoneFailure
				logMessage = fmt.Sprintf("[WARNING] Milestone Fetch Failure: %v", err)
			} else {
				processExpiryDateDiff(milestoneDate, &commentMessage, &logMessage)
			}
		} else {
			commentMessage = badExpiryError
			logMessage = "[ERROR]: Expiry condition badly formatted"
		}
	}
	if commentMessage == "" {
		log.Panicf("Primary expiry comment should not be empty")
	}
	expiryComments := []*tricium.Data_Comment{createExpiryComment(commentMessage, path, meta)}
	log.Printf("ADDING Comment for %s at line %d: %s", hist.Name, meta.HistogramLineNum, logMessage)
	return expiryComments
}

func processExpiryDateDiff(inputDate time.Time, commentMessage, logMessage *string) {
	dateDiff := int(inputDate.Sub(now()).Hours()/24) + 1
	if dateDiff <= 0 {
		*commentMessage = pastExpiryWarning
		*logMessage = "[WARNING]: Expiry in past"
	} else if dateDiff >= 365 {
		*commentMessage = farExpiryWarning
		*logMessage = "[WARNING]: Expiry past one year"
	} else {
		*commentMessage = fmt.Sprintf("[INFO]: Expiry date is in %d days", dateDiff)
		*logMessage = *commentMessage
	}
}

func getMilestoneDateImpl(milestone int) (time.Time, error) {
	var milestoneDate time.Time
	url := fmt.Sprintf("https://chromiumdash.appspot.com/fetch_milestone_schedule?mstone=%d", milestone)
	milestoneClient := http.Client{
		Timeout: time.Second * 2,
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return milestoneDate, err
	}
	res, err := milestoneClient.Do(req)
	if err != nil {
		return milestoneDate, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return milestoneDate, err
	}
	newMilestones := milestones{}
	err = json.Unmarshal(body, &newMilestones)
	if err != nil {
		return milestoneDate, err
	}
	dateString := newMilestones.Milestones[0].Date
	log.Printf("Fetched branch date %s for milestone %d", dateString, milestone)
	milestoneDate, err = time.Parse(dateMilestoneFormat, dateString)
	if err != nil {
		log.Panicf("Failed to parse milestone date: %v", err)
	}
	return milestoneDate, nil
}

func createExpiryComment(message, path string, meta *metadata) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  category + "/Expiry",
		Message:   message,
		Path:      path,
		StartLine: int32(meta.HistogramLineNum),
	}
}

func checkEnums(path string, hist *histogram, meta *metadata, singletonEnums stringset.Set) *tricium.Data_Comment {
	if singletonEnums.Has(hist.Enum) && !strings.Contains(hist.Summary, "baseline") {
		log.Printf("ADDING Comment for %s at line %d: %s", hist.Name, meta.HistogramLineNum, "Single Element Enum No Baseline")
		return &tricium.Data_Comment{
			Category:  category + "/Enums",
			Message:   singleElementEnumWarning,
			Path:      path,
			StartLine: int32(meta.HistogramLineNum),
		}
	}
	return nil
}

func findRemovedHistograms(path string, addedHistograms stringset.Set, removedHistograms stringset.Set) []*tricium.Data_Comment {
	var comments []*tricium.Data_Comment
	allRemovedHistograms := removedHistograms.Difference(addedHistograms).ToSlice()
	if len(allRemovedHistograms) > 0 {
		comment := &tricium.Data_Comment{
			Category: category + "/Removed",
			Message:  removedHistogramError,
			Path:     path,
		}
		comments = append(comments, comment)
		log.Printf("ADDING Comment: [ERROR]: Removed Histogram")
	}
	return comments
}

func findAddedNamespaces(path string, addedNamespaces stringset.Set, removedNamespaces stringset.Set, namespaceLineNums map[string]int) []*tricium.Data_Comment {
	var comments []*tricium.Data_Comment
	allAddedNamespaces := addedNamespaces.Difference(removedNamespaces).ToSlice()
	sort.Strings(allAddedNamespaces)
	for _, namespace := range allAddedNamespaces {
		comment := &tricium.Data_Comment{
			Category:  category + "/Namespace",
			Message:   fmt.Sprintf(addedNamespaceWarning, namespace),
			Path:      path,
			StartLine: int32(namespaceLineNums[namespace]),
		}
		log.Printf("ADDING Comment for %s at line %d: %s", namespace, comment.StartLine, "[WARNING]: Added Namespace")
		comments = append(comments, comment)
	}
	return comments
}

// newMetadata is a constructor for creating a Metadata struct with defaultLineNum.
func newMetadata(defaultLineNum int) *metadata {
	return &metadata{
		HistogramLineNum: defaultLineNum,
		OwnerLineNum:     defaultLineNum,
	}
}

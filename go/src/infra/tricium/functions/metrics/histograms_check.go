// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/waigani/diffparser"
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

	oneOwnerError = `[WARNING] It's a best practice to list multiple owners,
so that there's no single point of failure for communication:
https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Owners.`
	firstOwnerTeamError = `[WARNING] Please list an individual as the primary owner for this metric:
https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Owners.`
	noExpiryError = `[ERROR] Please specify an expiry condition for this histogram:
https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Histogram-Expiry.`
	badExpiryError = `[ERROR] Could not parse histogram expiry. Please format as YYYY-MM-DD or MXX:
https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Histogram-Expiry.`
	pastExpiryWarning = `[WARNING] This expiry date is in the past. Did you mean to set an expiry date in the future?`
	farExpiryWarning  = `[WARNING] It's a best practice to choose an expiry that is at most one year out:
https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Histogram-Expiry.`
	neverExpiryInfo = `[INFO] The expiry should only be set to \"never\" in rare cases.
Please double-check that this use of \"never\" is appropriate:
https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Histogram-Expiry.`
	neverExpiryError = `[ERROR] Histograms that never expire must have an XML comment describing why,
such as <!-- expires-never: \"heartbeat\" metric (internal: go/uma-heartbeats) -->:
https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Histogram-Expiry.`
	milestoneFailure = `[WARNING] Tricium failed to fetch milestone branch date.
Please double-check that this milestone is correct, because the tool is currently not able to check for you.`
	obsoleteDateError = `[WARNING] When marking a histogram as <obsolete>,
please document when the histogram was removed,
either as a date including a 2-digit month and 4-digit year,
or a milestone in MXX format.`
	removedHistogramError = `[ERROR]: Do not delete %s from histograms.xml. 
Instead, mark unused histograms as obsolete and annotate them with the date or milestone in the <obsolete> tag entry:
https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Cleaning-Up-Histogram-Entries`
)

var (
	// We need a pattern for matching the histogram start tag because
	// there are other tags that share the "histogram" prefix like "histogram-suffixes"
	histogramStartPattern     = regexp.MustCompile(`^<histogram(\s|>)`)
	neverExpiryCommentPattern = regexp.MustCompile(`^<!--\s?expires-never`)
	// Match date patterns of format YYYY-MM-DD
	expiryDatePattern      = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])$`)
	expiryMilestonePattern = regexp.MustCompile(`^M([0-9]{2,3})$`)
	// Match years between 1970 and 2999
	obsoleteYearPattern = regexp.MustCompile(`19[7-9][0-9]|2([0-9]{3})`)
	// Match double-digit or spelled-out months
	obsoleteMonthPattern     = regexp.MustCompile(`([^0-9](0[1-9]|10|11|12)[^0-9])|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec`)
	obsoleteMilestonePattern = regexp.MustCompile(`M([0-9]{2,3})`)

	// Now is an alias for time.Now, can be overwritten by tests
	now              = time.Now
	getMilestoneDate = getMilestoneDateImpl
)

// Histogram contains all info about a UMA histogram
type Histogram struct {
	Name     string   `xml:"name,attr"`
	Enum     string   `xml:"enum,attr"`
	Units    string   `xml:"units,attr"`
	Expiry   string   `xml:"expires_after,attr"`
	Obsolete string   `xml:"obsolete"`
	Owners   []string `xml:"owner"`
	Summary  string   `xml:"summary"`
}

// Metadata contains metadata about histogram tags and required comments
type Metadata struct {
	HistogramLineNum      int
	OwnerLineNum          int
	ObsoleteLineNum       int
	HasNeverExpiryComment bool
	HistogramBytes        []byte
}

// Milestone contains the date of a particular milestone
type Milestone struct {
	Milestone int    `json:"mstone"`
	Date      string `json:"branch_point"`
}

// Milestones contains a list of milestones
type Milestones struct {
	Milestones []Milestone `json:"mstones"`
}

type diffsPerFile struct {
	addedLines   map[string][]int
	removedLines map[string][]int
}

type changeMode int

const (
	// ADDED means a line was modified or added to a file
	ADDED changeMode = iota
	// REMOVED means a line was removed from a file
	REMOVED
)

func main() {
	inputDir := flag.String("input", "", "Path to root of Tricium input")
	outputDir := flag.String("output", "", "Path to root of Tricium output")
	flag.Parse()
	if flag.NArg() != 0 {
		log.Fatalf("Unexpected argument.")
	}
	// Read Tricium input FILES data.
	input := &tricium.Data_Files{}
	if err := tricium.ReadDataType(*inputDir, input); err != nil {
		log.Fatalf("Failed to read FILES data: %v", err)
	}
	log.Printf("Read FILES data.")

	results := &tricium.Data_Results{}

	// Only add .xml files to filePaths.
	var filePaths []string
	for _, file := range input.Files {
		if !file.IsBinary && filepath.Ext(file.Path) == ".xml" {
			filePaths = append(filePaths, filepath.Join(*inputDir, file.Path))
		}
	}

	// Return early if no .xml files were modified.
	if len(filePaths) == 0 {
		return
	}

	filesChanged, err := getDiffsPerFile(filepath.Join(*inputDir, input.Patch))
	if err != nil {
		log.Fatalf("Failed to get diffs per file: %v", err)
	}

	// Set up the temporary directory where we'll put original files.
	// The temporary directory should be cleaned up before exiting.
	tempDir, err := ioutil.TempDir(*inputDir, "get-original-file")
	if err != nil {
		log.Fatalf("Failed to setup temporary directory: %v", err)
	}
	defer func() {
		if err = os.RemoveAll(tempDir); err != nil {
			log.Fatalf("Failed to clean up temporary directory %q: %v", tempDir, err)
		}
	}()
	log.Printf("Created temporary directory %q.", tempDir)

	// Original files will be put into tempDir.
	getOriginalFiles(filePaths, tempDir, filepath.Join(*inputDir, input.Patch))

	for _, filePath := range filePaths {
		results.Comments = append(results.Comments, analyzeFile(filePath, tempDir, filesChanged)...)
	}

	// Write Tricium RESULTS data.
	path, err := tricium.WriteDataType(*outputDir, results)
	if err != nil {
		log.Fatalf("Failed to write RESULTS data: %v", err)
	}
	log.Printf("Wrote RESULTS data to path %q.", path)
}

// getDiffsPerFile gets the added and removed line numbers for a particular file.
func getDiffsPerFile(patchPath string) (*diffsPerFile, error) {
	patch, err := ioutil.ReadFile(patchPath)
	if err != nil {
		return &diffsPerFile{}, err
	}
	diff, err := diffparser.Parse(string(patch))
	if err != nil {
		return &diffsPerFile{}, err
	}
	diffInfo := &diffsPerFile{
		addedLines:   map[string][]int{},
		removedLines: map[string][]int{},
	}
	for _, diffFile := range diff.Files {
		if diffFile.Mode == diffparser.DELETED {
			continue
		}
		for _, hunk := range diffFile.Hunks {
			for _, line := range hunk.WholeRange.Lines {
				if line.Mode == diffparser.ADDED {
					diffInfo.addedLines[diffFile.NewName] = append(diffInfo.addedLines[diffFile.NewName], line.Number)
				} else if line.Mode == diffparser.REMOVED {
					diffInfo.removedLines[diffFile.NewName] = append(diffInfo.removedLines[diffFile.NewName], line.Number)
				}
			}
		}
	}
	return diffInfo, nil
}

// getOriginalFiles gets files in parent commit, before the patch, and puts them in tempDir.
func getOriginalFiles(filePaths []string, tempDir string, patchPath string) {
	filesToCopy := append(filePaths, patchPath)
	for _, filePath := range filesToCopy {
		tempPath := filepath.Join(tempDir, filePath)
		// Note: Must use filepath.Dir rather than path.Dir to be compatible with Windows.
		if err := os.MkdirAll(filepath.Dir(tempPath), os.ModePerm); err != nil {
			log.Fatalf("Failed to create dirs for file: %v", err)
		}
		copyFile(filePath, tempPath)
	}
	// Only apply patch if patch is not empty.
	fi, err := os.Stat(patchPath)
	if err != nil {
		log.Fatalf("Failed to get file info for patch %s: %v", patchPath, err)
	}
	if fi.Size() > 0 {
		cmds := []*exec.Cmd{exec.Command("git", "init")}
		cmds = append(cmds, exec.Command("git", "apply", "-p1", "--reverse", patchPath))
		for _, cmd := range cmds {
			cmd.Dir = tempDir
			log.Printf("Running cmd: %s", cmd.Args)
			if err := cmd.Run(); err != nil {
				log.Fatalf("Failed to run command %s\n%v\n", cmd.Args, err)
			}
		}
	}
}

func copyFile(sourceFile string, destFile string) {
	input, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		log.Fatalf("Failed to read file %s while copying file %s to %s: %v", sourceFile, sourceFile, destFile, err)
	}
	if err = ioutil.WriteFile(destFile, input, 0644); err != nil {
		log.Fatalf("Failed to write file %s while copying file %s to %s: %v", destFile, sourceFile, destFile, err)
	}
}

func analyzeFile(inputPath string, tempDir string, filesChanged *diffsPerFile) []*tricium.Data_Comment {
	log.Printf("ANALYZING File: %s", inputPath)
	var allComments []*tricium.Data_Comment
	f := openFileOrDie(inputPath)
	defer closeFileOrDie(f)
	// Analyze added lines in file (if any).
	comments, addedHistograms := analyzeChangedLines(bufio.NewScanner(f), inputPath, filesChanged.addedLines[inputPath], ADDED)
	allComments = append(allComments, comments...)
	// Analyze removed lines in file (if any).
	tempPath := filepath.Join(tempDir, inputPath)
	oldFile := openFileOrDie(tempPath)
	defer closeFileOrDie(oldFile)
	_, removedHistograms := analyzeChangedLines(bufio.NewScanner(oldFile), tempPath, filesChanged.removedLines[inputPath], REMOVED)
	// Identify any removed histograms.
	allRemovedHistograms := removedHistograms.Difference(addedHistograms).ToSlice()
	sort.Strings(allRemovedHistograms)
	for _, histogram := range allRemovedHistograms {
		comment := &tricium.Data_Comment{
			Category: fmt.Sprintf("%s/%s", category, "Removed"),
			Message:  fmt.Sprintf(removedHistogramError, histogram),
			Path:     inputPath,
		}
		log.Printf("ADDING Comment for %s: %s", histogram, "[ERROR]: Removed Histogram")
		allComments = append(allComments, comment)
	}
	return allComments
}

func analyzeChangedLines(scanner *bufio.Scanner, path string, linesChanged []int, mode changeMode) ([]*tricium.Data_Comment, stringset.Set) {
	var comments []*tricium.Data_Comment
	// metadata is a struct that holds line numbers of different tags in histogram.
	var metadata *Metadata
	// currHistogram is a buffer that holds the current histogram.
	var currHistogram []byte
	// histogramStart is the starting line number for the current histogram.
	var histogramStart int
	// histogramChanged is true if any line in the histogram showed up as an added or removed line in the diff.
	var histogramChanged bool
	changedHistograms := make(stringset.Set)
	lineNum := 1
	changedIndex := 0
	for scanner.Scan() {
		if changedIndex < len(linesChanged) && lineNum == linesChanged[changedIndex] {
			histogramChanged = true
			changedIndex++
		}
		line := strings.TrimSpace(scanner.Text())
		if currHistogram != nil {
			// Add line to currHistogram if currently between some histogram tags.
			currHistogram = append(currHistogram, scanner.Bytes()...)
		}
		if histogramStartPattern.MatchString(line) {
			// Initialize currHistogram and metadata when a new histogram is encountered.
			histogramStart = lineNum
			metadata = newMetadata(histogramStart)
			currHistogram = scanner.Bytes()
		} else if strings.HasPrefix(line, histogramEndTag) {
			// Analyze entire histogram after histogram end tag is encountered.
			if histogramChanged {
				histogram := bytesToHistogram(currHistogram, metadata)
				changedHistograms.Add(histogram.Name)
				// Only check new (added) histograms are correct.
				if mode == ADDED {
					comments = append(comments, checkHistogram(path, currHistogram, metadata)...)
				}
			}
			currHistogram = nil
			histogramChanged = false
		} else if strings.HasPrefix(line, ownerStartTag) {
			if metadata.OwnerLineNum == histogramStart {
				metadata.OwnerLineNum = lineNum
			}
		} else if strings.HasPrefix(line, obsoleteStartTag) {
			metadata.ObsoleteLineNum = lineNum
		} else if neverExpiryCommentPattern.MatchString(line) {
			metadata.HasNeverExpiryComment = true
		}
		lineNum++
	}
	return comments, changedHistograms
}

func checkHistogram(path string, histBytes []byte, metadata *Metadata) []*tricium.Data_Comment {
	var comments []*tricium.Data_Comment
	histogram := bytesToHistogram(histBytes, metadata)
	if comment := checkNumOwners(path, histogram, metadata); comment != nil {
		comments = append(comments, comment)
	}
	if comment := checkNonTeamOwner(path, histogram, metadata); comment != nil {
		comments = append(comments, comment)
	}
	if comment := checkObsolete(path, histogram, metadata); comment != nil {
		comments = append(comments, comment)
	}
	comments = append(comments, checkExpiry(path, histogram, metadata)...)
	return comments
}

func bytesToHistogram(histBytes []byte, metadata *Metadata) Histogram {
	var histogram Histogram
	if err := xml.Unmarshal(histBytes, &histogram); err != nil {
		log.Fatalf("WARNING: Failed to unmarshal histogram at line %d", metadata.HistogramLineNum)
	}
	return histogram
}

func checkNumOwners(path string, histogram Histogram, metadata *Metadata) *tricium.Data_Comment {
	if len(histogram.Owners) <= 1 {
		comment := createOwnerComment(oneOwnerError, path, metadata)
		log.Printf("ADDING Comment for %s at line %d: %s", histogram.Name, comment.StartLine, "[ERROR]: One Owner")
		return comment
	}
	return nil
}

func checkNonTeamOwner(path string, histogram Histogram, metadata *Metadata) *tricium.Data_Comment {
	if len(histogram.Owners) > 0 && strings.Contains(histogram.Owners[0], "-") {
		comment := createOwnerComment(firstOwnerTeamError, path, metadata)
		log.Printf("ADDING Comment for %s at line %d: %s", histogram.Name, comment.StartLine, "[ERROR]: First Owner Team")
		return comment
	}
	return nil
}

func createOwnerComment(message string, path string, metadata *Metadata) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  fmt.Sprintf("%s/%s", category, "Owners"),
		Message:   message,
		Path:      path,
		StartLine: int32(metadata.OwnerLineNum),
	}
}

func checkObsolete(path string, histogram Histogram, metadata *Metadata) *tricium.Data_Comment {
	if histogram.Obsolete != "" &&
		!obsoleteMilestonePattern.MatchString(histogram.Obsolete) &&
		!(obsoleteYearPattern.MatchString(histogram.Obsolete) &&
			obsoleteMonthPattern.MatchString(histogram.Obsolete)) {
		comment := &tricium.Data_Comment{
			Category:  fmt.Sprintf("%s/%s", category, "Obsolete"),
			Message:   obsoleteDateError,
			Path:      path,
			StartLine: int32(metadata.ObsoleteLineNum),
		}
		log.Printf("ADDING Comment for %s at line %d: %s", histogram.Name, comment.StartLine, "[ERROR]: Obsolete no date")
		return comment
	}
	return nil
}

func checkExpiry(path string, histogram Histogram, metadata *Metadata) []*tricium.Data_Comment {
	var commentMessage string
	var logMessage string
	var extraComment *tricium.Data_Comment
	if expiry := histogram.Expiry; expiry == "" {
		commentMessage = noExpiryError
		logMessage = "[ERROR]: No Expiry"
	} else if expiry == "never" {
		commentMessage = neverExpiryInfo
		logMessage = "[INFO]: Never Expiry"
		// Add second Tricium comment if an expiry of never has no comment.
		if !metadata.HasNeverExpiryComment {
			extraComment = createExpiryComment(neverExpiryError, path, metadata)
			logMessage += " & [ERROR]: No Comment"
		}
	} else {
		dateMatch := expiryDatePattern.MatchString(expiry)
		milestoneMatch := expiryMilestonePattern.MatchString(expiry)
		if dateMatch {
			inputDate, err := time.Parse(dateFormat, expiry)
			if err != nil {
				log.Fatalf("Failed to parse expiry date: %v", err)
			}
			processExpiryDateDiff(inputDate, &commentMessage, &logMessage)
		} else if milestoneMatch {
			milestone, err := strconv.Atoi(expiry[1:])
			if err != nil {
				log.Fatalf("Failed to convert input milestone to integer: %v", err)
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
		log.Fatalf("Primary expiry comment should not be empty")
	}
	expiryComments := []*tricium.Data_Comment{createExpiryComment(commentMessage, path, metadata)}
	if extraComment != nil {
		expiryComments = append(expiryComments, extraComment)
	}
	log.Printf("ADDING Comment for %s at line %d: %s", histogram.Name, metadata.HistogramLineNum, logMessage)
	return expiryComments
}

func processExpiryDateDiff(inputDate time.Time, commentMessage *string, logMessage *string) {
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
	newMilestones := Milestones{}
	err = json.Unmarshal(body, &newMilestones)
	if err != nil {
		return milestoneDate, err
	}
	dateString := newMilestones.Milestones[0].Date
	log.Printf("Fetched branch date %s for milestone %d", dateString, milestone)
	milestoneDate, err = time.Parse(dateMilestoneFormat, dateString)
	if err != nil {
		log.Fatalf("Failed to parse milestone date: %v", err)
	}
	return milestoneDate, nil
}

func createExpiryComment(message string, path string, metadata *Metadata) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
		Message:   message,
		Path:      path,
		StartLine: int32(metadata.HistogramLineNum),
	}
}

func openFileOrDie(path string) *os.File {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to open file: %v, path: %s", err, path)
	}
	return f
}

func closeFileOrDie(f *os.File) {
	if err := f.Close(); err != nil {
		log.Fatalf("Failed to close file: %v", err)
	}
}

// newMetadata is a constructor for creating a Metadata struct with defaultLineNum.
func newMetadata(defaultLineNum int) *Metadata {
	return &Metadata{
		HistogramLineNum: defaultLineNum,
		OwnerLineNum:     defaultLineNum,
	}
}

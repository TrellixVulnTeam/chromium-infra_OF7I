// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
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
	unitsMicrosecondsWarning = `[WARNING] Histograms with units="microseconds" should document 
whether the metrics is reported for all users or only users with high-resolution clocks.
Note that reports from clients with low-resolution clocks (i.e. on Windows, ref. 
|TimeTicks::IsHighResolution()|) may cause the metric to have an abnormal distribution.`
	removedHistogramError = `[ERROR]: Do not delete %s from histograms.xml. 
Instead, mark unused histograms as obsolete and annotate them with the date or milestone in the <obsolete> tag entry:
https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#Cleaning-Up-Histogram-Entries`
	addedNamespaceWarning = `[WARNING]: Are you sure you want to add the namespace %s to histograms.xml?
For most new histograms, it's appropriate to re-use one of the existing 
top-level histogram namespaces. For histogram names, the namespace 
is defined as everything preceding the first dot '.' in the name.`
	singleElementEnumWarning = `[WARNING]: It looks like this is an enumerated histogram that contains only a single bucket. 
UMA metrics are difficult to interpret in isolation, so please either 
add one or more additional buckets that can serve as a baseline for comparison, 
or document what other metric should be used as a baseline during analysis. 
https://chromium.googlesource.com/chromium/src/+/HEAD/tools/metrics/histograms/README.md#enum-histograms.`
)

var (
	// We need a pattern for matching the histogram start tag because
	// there are other tags that share the "histogram" prefix like "histogram-suffixes"
	histogramStartPattern     = regexp.MustCompile(`^<histogram($|\s|>)`)
	neverExpiryCommentPattern = regexp.MustCompile(`^<!--\s?expires-never`)
	// Match date patterns of format YYYY-MM-DD
	expiryDatePattern      = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])$`)
	expiryMilestonePattern = regexp.MustCompile(`^M([0-9]{2,3})$`)
	// Match years between 1970 and 2999
	obsoleteYearPattern = regexp.MustCompile(`19[7-9][0-9]|2([0-9]{3})`)
	// Match double-digit or spelled-out months
	obsoleteMonthPattern     = regexp.MustCompile(`([^0-9](0[1-9]|10|11|12)[^0-9])|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec`)
	obsoleteMilestonePattern = regexp.MustCompile(`M([0-9]{2,3})`)
	// Match valid summaries for histograms with units=microseconds
	microsecondsSummary = regexp.MustCompile(`all\suser|(high|low)(\s|-)resolution`)

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

// enum contains all the data about a particular enum.
type enum struct {
	Name     string `xml:"name,attr"`
	Elements []struct {
		Value string `xml:"value,attr"`
		Label string `xml:"label,attr"`
	} `xml:"int"`
}

// enumFile contains all the data in an enums file.
type enumFile struct {
	Enums struct {
		EnumList []enum `xml:"enum"`
	} `xml:"enums"`
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
	prevDir := flag.String("previous", "", "Path to directory with previous versions of changed files")
	patchPath := flag.String("patch", "", "Path to patch of changed files")
	enumsPath := flag.String("enums", "src/tools/metrics/histograms/enums.xml", "Path to enums file")
	flag.Parse()
	// This is a temporary way for us to get recipes to work without breaking the current analyzer.
	var filePaths []string
	var filesChanged *diffsPerFile
	var err error
	if *prevDir != "" {
		filePaths = flag.Args()
		filesChanged, err = getDiffsPerFile(filePaths, *patchPath)
		if err != nil {
			log.Panicf("Failed to get diffs per file: %v", err)
		}
	} else {
		// Read Tricium input FILES data.
		input := &tricium.Data_Files{}
		if err := tricium.ReadDataType(*inputDir, input); err != nil {
			log.Panicf("Failed to read FILES data: %v. Did you specify a Tricium-compatible input directory with -input?", err)
		}
		log.Printf("Read FILES data.")
		// Only add histogram.xml to filePaths.
		for _, file := range input.Files {
			if !file.IsBinary && filepath.Base(file.Path) == "histograms.xml" {
				filePaths = append(filePaths, file.Path)
			}
		}
		// We need this outside the if statement since it will be used in getDiffsPerFile later.
		*patchPath = input.Patch
		// Only get previous files if .xml files were modified.
		if len(filePaths) != 0 {
			// Set up the temporary directory where we'll put previous files and apply the patch on them.
			// The temporary directory should be cleaned up before exiting.
			tempDir, err := ioutil.TempDir(*inputDir, "get-previous-file")
			if err != nil {
				log.Panicf("Failed to setup temporary directory: %v", err)
			}
			defer func() {
				if err = os.RemoveAll(tempDir); err != nil {
					log.Panicf("Failed to clean up temporary directory %q: %v", tempDir, err)
				}
			}()
			log.Printf("Created temporary directory %q.", tempDir)
			*prevDir = filepath.Join(tempDir, *inputDir)
			// Previous files will be put into prevDir.
			getPreviousFiles(filePaths, *inputDir, *prevDir, *patchPath)
		}
		filesChanged, err = getDiffsPerFile(filePaths, filepath.Join(*inputDir, *patchPath))
		if err != nil {
			log.Panicf("Failed to get diffs per file: %v", err)
		}
	}
	singletonEnums := getSingleElementEnums(filepath.Join(*inputDir, *enumsPath))

	results := &tricium.Data_Results{}
	for _, filePath := range filePaths {
		results.Comments = append(results.Comments, analyzeFile(filePath, *inputDir, *prevDir, filesChanged, singletonEnums)...)
	}

	// Write Tricium RESULTS data.
	path, err := tricium.WriteDataType(*outputDir, results)
	if err != nil {
		log.Panicf("Failed to write RESULTS data: %v. Did you specify an output directory with -output?", err)
	}
	log.Printf("Wrote RESULTS data to path %q.", path)
}

// getDiffsPerFile gets the added and removed line numbers for a particular file.
func getDiffsPerFile(filePaths []string, patchPath string) (*diffsPerFile, error) {
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
	fileSet := stringset.NewFromSlice(filePaths...)
	for _, diffFile := range diff.Files {
		if diffFile.Mode == diffparser.DELETED || !fileSet.Has(diffFile.NewName) {
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

// getPreviousFiles unapplies a patch in copied files.
// It copies filePaths files from inputDir into prevDir,
// then applies the patch file at patchPath reversed on the copied files.
// patchPath must be relative to inputDir.
func getPreviousFiles(filePaths []string, inputDir, prevDir, patchPath string) {
	filesToCopy := append(filePaths, patchPath)
	for _, filePath := range filesToCopy {
		tempPath := filepath.Join(prevDir, filePath)
		// Note: Must use filepath.Dir rather than path.Dir to be compatible with Windows.
		if err := os.MkdirAll(filepath.Dir(tempPath), os.ModePerm); err != nil {
			log.Panicf("Failed to create dirs for file: %v", err)
		}
		copyFile(filepath.Join(inputDir, filePath), tempPath)
	}
	// Only apply patch if patch is not empty.
	fi, err := os.Stat(filepath.Join(inputDir, patchPath))
	if err != nil {
		log.Panicf("Failed to get file info for patch %s: %v", patchPath, err)
	}
	if fi.Size() > 0 {
		cmds := []*exec.Cmd{exec.Command("git", "init")}
		for _, filePath := range filePaths {
			cmds = append(cmds, exec.Command("git", "apply", "-p1", "--reverse", "--include="+filePath, patchPath))
		}
		for _, c := range cmds {
			var stderr bytes.Buffer
			c.Dir = prevDir
			c.Stderr = &stderr
			log.Printf("Running cmd: %s", c.Args)
			if err := c.Run(); err != nil {
				log.Panicf("Failed to run command %s\n%v\nStderr: %s", c.Args, err, stderr.String())
			}
		}
	}
}

func copyFile(sourceFile, destFile string) {
	input, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		log.Panicf("Failed to read file %s while copying file %s to %s: %v", sourceFile, sourceFile, destFile, err)
	}
	if err = ioutil.WriteFile(destFile, input, 0644); err != nil {
		log.Panicf("Failed to write file %s while copying file %s to %s: %v", destFile, sourceFile, destFile, err)
	}
}

func getSingleElementEnums(inputPath string) stringset.Set {
	singletonEnums := make(stringset.Set)
	f := openFileOrDie(inputPath)
	defer closeFileOrDie(f)
	enumBytes, err := ioutil.ReadAll(f)
	if err != nil {
		log.Panicf("Failed to read enums into buffer: %v. Did you specify the enums file correctly with -enums?", err)
	}
	var enumFile enumFile
	if err := xml.Unmarshal(enumBytes, &enumFile); err != nil {
		log.Panicf("Failed to unmarshal enums: %v", err)
	}
	for _, enum := range enumFile.Enums.EnumList {
		if len(enum.Elements) == 1 {
			singletonEnums.Add(enum.Name)
		}
	}
	return singletonEnums
}

func analyzeFile(filePath, inputDir, prevDir string, filesChanged *diffsPerFile, singletonEnums stringset.Set) []*tricium.Data_Comment {
	log.Printf("ANALYZING File: %s", filePath)
	var allComments []*tricium.Data_Comment
	inputPath := filepath.Join(inputDir, filePath)
	f := openFileOrDie(inputPath)
	defer closeFileOrDie(f)
	// Analyze added lines in file (if any)
	comments, addedHistograms, newNamespaces, namespaceLineNums := analyzeChangedLines(bufio.NewScanner(f), inputPath, filesChanged.addedLines[filePath], singletonEnums, ADDED)
	allComments = append(allComments, comments...)
	// Analyze removed lines in file (if any)
	tempPath := filepath.Join(prevDir, filePath)
	oldFile := openFileOrDie(tempPath)
	defer closeFileOrDie(oldFile)
	var emptySet stringset.Set
	_, removedHistograms, oldNamespaces, _ := analyzeChangedLines(bufio.NewScanner(oldFile), tempPath, filesChanged.removedLines[filePath], emptySet, REMOVED)
	// Identify any removed histograms
	allComments = append(allComments, findRemovedHistograms(inputPath, addedHistograms, removedHistograms)...)
	allComments = append(allComments, findAddedNamespaces(inputPath, newNamespaces, oldNamespaces, namespaceLineNums)...)
	return allComments
}

func analyzeChangedLines(scanner *bufio.Scanner, path string, linesChanged []int, singletonEnums stringset.Set, mode changeMode) ([]*tricium.Data_Comment, stringset.Set, stringset.Set, map[string]int) {
	var comments []*tricium.Data_Comment
	// metadata is a struct that holds line numbers of different tags in histogram.
	var metadata *Metadata
	// currHistogram is a buffer that holds the current histogram.
	var currHistogram []byte
	// histogramStart is the starting line number for the current histogram.
	var histogramStart int
	// If any line in the histogram showed up as an added or removed line in the diff
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
			metadata = newMetadata(histogramStart)
			currHistogram = newBytes
			histogramChanged = false
		} else if strings.HasPrefix(line, histogramEndTag) {
			// Analyze entire histogram after histogram end tag is encountered
			histogram := bytesToHistogram(currHistogram, metadata)
			namespace := strings.SplitN(histogram.Name, ".", 2)[0]
			namespaces.Add(namespace)
			namespaceLineNums[namespace] = metadata.HistogramLineNum
			if histogramChanged {
				changedHistograms.Add(histogram.Name)
				// Only check new (added) histograms are correct
				if mode == ADDED {
					comments = append(comments, checkHistogram(path, currHistogram, metadata, singletonEnums)...)
				}
			}
			currHistogram = nil
		} else if strings.HasPrefix(line, ownerStartTag) {
			if metadata.OwnerLineNum == histogramStart {
				metadata.OwnerLineNum = lineNum
			}
		} else if strings.HasPrefix(line, obsoleteStartTag) {
			metadata.ObsoleteLineNum = lineNum
		} else if neverExpiryCommentPattern.MatchString(line) {
			metadata.HasNeverExpiryComment = true
		}
		if changedIndex < len(linesChanged) && lineNum == linesChanged[changedIndex] {
			histogramChanged = true
			changedIndex++
		}
		lineNum++
	}
	return comments, changedHistograms, namespaces, namespaceLineNums
}

func checkHistogram(path string, histBytes []byte, metadata *Metadata, singletonEnums stringset.Set) []*tricium.Data_Comment {
	var comments []*tricium.Data_Comment
	histogram := bytesToHistogram(histBytes, metadata)
	if comment := checkNumOwners(path, histogram, metadata); comment != nil {
		comments = append(comments, comment)
	}
	if comment := checkNonTeamOwner(path, histogram, metadata); comment != nil {
		comments = append(comments, comment)
	}
	if comment := checkUnits(path, histogram, metadata); comment != nil {
		comments = append(comments, comment)
	}
	if comment := checkObsolete(path, histogram, metadata); comment != nil {
		comments = append(comments, comment)
	}
	if comment := checkEnums(path, histogram, metadata, singletonEnums); comment != nil {
		comments = append(comments, comment)
	}
	comments = append(comments, checkExpiry(path, histogram, metadata)...)
	return comments
}

func bytesToHistogram(histBytes []byte, metadata *Metadata) Histogram {
	var histogram Histogram
	if err := xml.Unmarshal(histBytes, &histogram); err != nil {
		log.Panicf("WARNING: Failed to unmarshal histogram at line %d", metadata.HistogramLineNum)
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

func createOwnerComment(message, path string, metadata *Metadata) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  category + "/Owners",
		Message:   message,
		Path:      path,
		StartLine: int32(metadata.OwnerLineNum),
	}
}

func checkUnits(path string, histogram Histogram, metadata *Metadata) *tricium.Data_Comment {
	if strings.Contains(histogram.Units, "microseconds") && !microsecondsSummary.MatchString(histogram.Summary) {
		comment := &tricium.Data_Comment{
			Category:  category + "/Units",
			Message:   unitsMicrosecondsWarning,
			Path:      path,
			StartLine: int32(metadata.HistogramLineNum),
		}
		log.Printf("ADDING Comment for %s at line %d: %s", histogram.Name, comment.StartLine, "[ERROR]: Units Microseconds Bad Summary")
		return comment
	}
	return nil
}

func checkObsolete(path string, histogram Histogram, metadata *Metadata) *tricium.Data_Comment {
	if histogram.Obsolete != "" &&
		!obsoleteMilestonePattern.MatchString(histogram.Obsolete) &&
		!(obsoleteYearPattern.MatchString(histogram.Obsolete) &&
			obsoleteMonthPattern.MatchString(histogram.Obsolete)) {
		comment := &tricium.Data_Comment{
			Category:  category + "/Obsolete",
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
	expiryComments := []*tricium.Data_Comment{createExpiryComment(commentMessage, path, metadata)}
	if extraComment != nil {
		expiryComments = append(expiryComments, extraComment)
	}
	log.Printf("ADDING Comment for %s at line %d: %s", histogram.Name, metadata.HistogramLineNum, logMessage)
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
	newMilestones := Milestones{}
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

func createExpiryComment(message, path string, metadata *Metadata) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  category + "/Expiry",
		Message:   message,
		Path:      path,
		StartLine: int32(metadata.HistogramLineNum),
	}
}

func checkEnums(path string, histogram Histogram, metadata *Metadata, singletonEnums stringset.Set) *tricium.Data_Comment {
	if singletonEnums.Has(histogram.Enum) && !strings.Contains(histogram.Summary, "baseline") {
		log.Printf("ADDING Comment for %s at line %d: %s", histogram.Name, metadata.HistogramLineNum, "Single Element Enum No Baseline")
		return &tricium.Data_Comment{
			Category:  category + "/Enums",
			Message:   singleElementEnumWarning,
			Path:      path,
			StartLine: int32(metadata.HistogramLineNum),
		}
	}
	return nil
}

func findRemovedHistograms(path string, addedHistograms stringset.Set, removedHistograms stringset.Set) []*tricium.Data_Comment {
	var comments []*tricium.Data_Comment
	allRemovedHistograms := removedHistograms.Difference(addedHistograms).ToSlice()
	sort.Strings(allRemovedHistograms)
	for _, histogram := range allRemovedHistograms {
		comment := &tricium.Data_Comment{
			Category: category + "/Removed",
			Message:  fmt.Sprintf(removedHistogramError, histogram),
			Path:     path,
		}
		log.Printf("ADDING Comment for %s: %s", histogram, "[ERROR]: Removed Histogram")
		comments = append(comments, comment)
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

func openFileOrDie(path string) *os.File {
	f, err := os.Open(path)
	if err != nil {
		log.Panicf("Failed to open file: %v, path: %s", err, path)
	}
	return f
}

func closeFileOrDie(f *os.File) {
	if err := f.Close(); err != nil {
		log.Panicf("Failed to close file: %v", err)
	}
}

// newMetadata is a constructor for creating a Metadata struct with defaultLineNum.
func newMetadata(defaultLineNum int) *Metadata {
	return &Metadata{
		HistogramLineNum: defaultLineNum,
		OwnerLineNum:     defaultLineNum,
	}
}

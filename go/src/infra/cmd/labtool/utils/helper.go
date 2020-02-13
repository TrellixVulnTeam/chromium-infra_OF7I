// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/libs/fleet/protos"
)

// TimeFormat for all timestamps handled by labtools
var timeFormat = "2006-01-02-15:04:05"

// The tab writer which defines the write format
var tw = tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)

// The formatter for log and result file names
var logFileExp = regexp.MustCompile(`[\d]{4}(-[\d]{1,2}){3}(:[\d]{1,2}){2}-log$`)
var resFileExp = regexp.MustCompile(`[\d]{4}(-[\d]{1,2}){3}(:[\d]{1,2}){2}-res$`)

// The length of the string list that an asset will be converted to.
// 1 for ID and 6 for location
// TODO: find a better way to count the length
const lenOfAssetStringList = 7

// States used in result file.
const successState = "Success"
const failureState = "Failure"

// AssetStats to store the statistics of a chops asset
type AssetStats struct {
	Asset    *fleet.ChopsAsset
	Action   string
	ErrorMsg string
}

// LogStats to store the statistics of any given run
type LogStats struct {
	LogPath string
	ResPath string
	Tstamp  time.Time
	// The times that we scan an asset in the run
	ScannedAssetCount int
	// The times that we scan a location in the run
	ScannedLocationCount int
	SuccessfulAssetScan  int
	FailedAssetScan      int

	ScannedAssets    map[string]*AssetStats
	ScannedLocations map[string]bool
	MismatchedAssets map[string]*AssetStats
	// The failure when generating the stats
	FailureMsg []string
}

// LogStatsList refers to a list of log stats.
type LogStatsList []*LogStats

// LogStats sort functions
func (l LogStatsList) Less(i, j int) bool { return l[i].Tstamp.After(l[j].Tstamp) }
func (l LogStatsList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l LogStatsList) Len() int           { return len(l) }

// populateStatistics generates the stats of a round of scans by log and res file.
func populateStatistics(logPath, resPath string, tStamp time.Time) (*LogStats, error) {
	//TODO: Add counting Success, Failure and Move rates
	lstats := &LogStats{
		LogPath: logPath,
		ResPath: resPath,
		Tstamp:  tStamp,
	}

	if err := lstats.populateLogFile(); err != nil {
		return nil, err
	}
	if err := lstats.populateResFile(); err != nil {
		return nil, err
	}
	return lstats, nil
}

func (lstats *LogStats) populateLogFile() error {
	scannedAssets := make(map[string]*AssetStats)
	scannedLocations := make(map[string]bool)
	logF, err := os.Open(lstats.LogPath)
	defer logF.Close()
	if err != nil {
		return err
	}
	recs, err := csv.NewReader(logF).ReadAll()
	if err == nil {
		for _, i := range recs {
			a, _ := stringListToAsset(i)
			scannedAssets[a.GetId()] = &AssetStats{
				Asset: a,
			}
			locationStr := locationToStringList(a.GetLocation())
			scannedLocations[strings.Join(locationStr, "")] = true
		}
		lstats.ScannedAssetCount = len(recs)
		lstats.ScannedLocationCount = len(scannedLocations)
	} else {
		return errors.Annotate(err, "fail to read file %s", lstats.LogPath).Err()
	}

	lstats.ScannedAssets = scannedAssets
	lstats.ScannedLocations = scannedLocations
	return nil
}

// Should be called after populateLogFile()
func (lstats *LogStats) populateResFile() error {
	mismatchedAssets := make(map[string]*AssetStats)
	resF, err := os.Open(lstats.ResPath)
	defer resF.Close()
	if err != nil {
		return err
	}
	recs, err := csv.NewReader(resF).ReadAll()
	if err == nil {
		for _, line := range recs {
			assetTag := line[1]
			if lstats.ScannedAssets[assetTag] == nil {
				mismatchedAssets[assetTag] = &AssetStats{
					Asset:    nil,
					ErrorMsg: "Asset exists in result file, but not in log file",
				}
				continue
			}
			lstats.ScannedAssets[assetTag].Action = line[len(line)-2]
			switch line[0] {
			case successState:
				lstats.SuccessfulAssetScan++
			default:
				lstats.FailedAssetScan++
				lstats.ScannedAssets[assetTag].ErrorMsg = line[len(line)-1]
			}
		}
	} else {
		return errors.Annotate(err, "fail to read file %s", lstats.ResPath).Err()
	}
	lstats.MismatchedAssets = mismatchedAssets
	return nil
}

func (lstats *LogStats) parseAssets() (map[string][]*AssetStats, map[string][]*AssetStats) {
	success := make(map[string][]*AssetStats, 0)
	failed := make(map[string][]*AssetStats, 0)
	for _, a := range lstats.ScannedAssets {
		if a.ErrorMsg != "" {
			failed[a.Action] = append(failed[a.Action], a)
		} else {
			success[a.Action] = append(success[a.Action], a)
		}
	}
	return success, failed
}

// used for writing csv entries
func assetToStringList(a *fleet.ChopsAsset) []string {
	if a == nil {
		return nil
	}
	res := []string{a.GetId()}
	return append(res, locationToStringList(a.GetLocation())...)
}

func locationToStringList(location *fleet.Location) []string {
	return []string{
		location.GetLab(),
		location.GetAisle(),
		location.GetRow(),
		location.GetRack(),
		location.GetShelf(),
		location.GetPosition(),
	}
}

// stringListToAsset converts String array of size lenOfAssetStringList to Asset object
func stringListToAsset(csv []string) (a *fleet.ChopsAsset, err error) {
	if len(csv) != lenOfAssetStringList {
		//TODO: Add error obj creation here
		return nil, nil
	}
	return &fleet.ChopsAsset{
		Id: csv[0],
		Location: &fleet.Location{
			Lab:      csv[1],
			Aisle:    csv[2],
			Row:      csv[3],
			Rack:     csv[4],
			Shelf:    csv[5],
			Position: csv[6],
		},
	}, nil
}

// PrintLogStatsAndResult prints the stats and results for a specified audit scan run.
func PrintLogStatsAndResult(l *LogStats, index int) {
	defer tw.Flush()
	PrintLogStats(LogStatsList{l}, 1)

	fmt.Fprintln(tw, "\nSuccessful assets:")
	fmt.Fprintln(tw, "Action\t\tNumber of Assets")
	successAssets, failedAssets := l.parseAssets()
	for action, a := range successAssets {
		fmt.Fprintln(tw, fmt.Sprintf("%s\t\t%d", action, len(a)))
	}
	if len(failedAssets) == 0 {
		fmt.Fprintln(tw, "\nNo failed assets")
		return
	}
	fmt.Fprintln(tw, "\nFailed assets:")
	fmt.Fprintln(tw, "Asset Tag\t\tAction\t\tLocation\t\tError")
	for _, assets := range failedAssets {
		for _, a := range assets {
			out := fmt.Sprintf("%s\t\t%s\t\t%s\t\t%s", a.Asset.GetId(), a.Action, a.Asset.GetLocation(), a.ErrorMsg)
			fmt.Fprintln(tw, out)
		}
	}
}

func printLogStatsTitle() {
	fmt.Fprintln(tw, "Index\t\tTime\t\tAssets Scanned\tUnique Assets\tUnique Locations\tSuccessful Assets\tFailed Assets\t")
}

// PrintLogStats prints infos for a batch of audit scan runs.
func PrintLogStats(l LogStatsList, limit int) {
	defer tw.Flush()
	fmt.Fprintln(tw, "\nOverall Stats:")
	if len(l) > limit && limit > 0 {
		l = l[:limit]
	}
	fmt.Fprintln(tw, "Index\t\tTime\t\tAssets Scanned\tUnique Assets\tUnique Locations\tSuccessful Assets\tFailed Assets\t")
	for i, lstats := range l {
		printOneLog(i, lstats, tw)
	}
}

func printOneLog(index int, lstats *LogStats, tw *tabwriter.Writer) {
	if len(lstats.FailureMsg) > 0 {
		out := fmt.Sprintf("%d\t\tErrors:\t\t\t\t\t\t%s\t", index, strings.Join(lstats.FailureMsg, "; "))
		fmt.Fprintln(tw, out)
		return
	}
	out := fmt.Sprintf(
		"%d\t\t%s\t\t%d\t%d\t%d\t%d\t%d\t",
		index,
		lstats.Tstamp.Format(timeFormat),
		lstats.ScannedAssetCount,
		len(lstats.ScannedAssets),
		len(lstats.ScannedLocations),
		lstats.SuccessfulAssetScan,
		lstats.FailedAssetScan,
	)
	fmt.Fprintln(tw, out)
}

// ListLogs lists the logs and return the stats for each of the audit runs.
func ListLogs(dir string) (LogStatsList, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}
	res := []*LogStats{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(fmt.Sprintf("Fail to walk through %s", path))
			return err
		}
		if logFileExp.MatchString(info.Name()) {
			timeStampStr := strings.Trim(info.Name(), "-log")
			resPath := filepath.Join(dir, timeStampStr+"-res")
			logPath := filepath.Join(dir, info.Name())
			stats := getStats(logPath, resPath, timeStampStr)
			res = append(res, stats)
		}
		return err
	})
	return res, err
}

func getStats(logPath, resPath, tstampStr string) *LogStats {
	tStamp, err := time.Parse(timeFormat, tstampStr)
	if err != nil {
		return &LogStats{
			FailureMsg: []string{fmt.Sprintf("Fail to parse timestamp in filename: %s", logPath)},
		}
	}

	if _, err := os.Stat(resPath); err != nil {
		return &LogStats{
			FailureMsg: []string{fmt.Sprintf("Fail to locate result file: %s", resPath)},
			Tstamp:     tStamp,
		}
	}
	stats, err := populateStatistics(logPath, resPath, tStamp)
	if err != nil {
		return &LogStats{
			FailureMsg: []string{err.Error()},
			Tstamp:     tStamp,
		}
	}
	return stats
}

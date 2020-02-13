// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"infra/libs/fleet/protos"
)

// TimeFormat for all timestamps handled by labtools
var timeFormat = "2006-01-02-15:04:05"

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
		lstats.FailureMsg = append(lstats.FailureMsg, fmt.Sprintf("Fail to read file %s: %s\n", lstats.LogPath, err.Error()))
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
		lstats.FailureMsg = append(lstats.FailureMsg, fmt.Sprintf("Fail to read file %s: %s\n", lstats.ResPath, err.Error()))
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

func printLogStatsAndResult(l *LogStats, index int) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	defer w.Flush()
	if len(l.FailureMsg) > 0 {
		fmt.Fprintln(w, "\nErrors in reading stats")
		for _, msg := range l.FailureMsg {
			fmt.Fprintln(w, msg)
		}
		return
	}

	fmt.Fprintln(w, "\nOverall Stats:")
	fmt.Fprintln(w, "Index\t\tTime\t\tAssets Scanned\tUnique Assets\tUnique Locations\tSuccessful Assets\tFailed Assets\t")
	ts := l.Tstamp.Format(timeFormat)
	out := fmt.Sprintf("%d\t\t%s\t\t%d\t%d\t%d\t%d\t%d\t", index, ts, l.ScannedAssetCount, len(l.ScannedAssets), len(l.ScannedLocations), l.SuccessfulAssetScan, l.FailedAssetScan)
	fmt.Fprintln(w, out)

	fmt.Fprintln(w, "\nSuccessful assets:")
	fmt.Fprintln(w, "Action\t\tNumber of Assets")
	successAssets, failedAssets := l.parseAssets()
	for action, a := range successAssets {
		fmt.Fprintln(w, fmt.Sprintf("%s\t\t%d", action, len(a)))
	}
	if len(failedAssets) == 0 {
		fmt.Fprintln(w, "\nNo failed assets")
		return
	}
	fmt.Fprintln(w, "\nFailed assets:")
	fmt.Fprintln(w, "Asset Tag\t\tAction\t\tLocation\t\tError")
	for _, assets := range failedAssets {
		for _, a := range assets {
			out = fmt.Sprintf("%s\t\t%s\t\t%s\t\t%s", a.Asset.GetId(), a.Action, a.Asset.GetLocation(), a.ErrorMsg)
			fmt.Fprintln(w, out)
		}
	}
}

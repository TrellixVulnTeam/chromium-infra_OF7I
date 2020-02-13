// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	fleetAPI "infra/appengine/cros/lab_inventory/api/v1"
	"infra/libs/fleet/protos"
)

// Updater is used to asynchronously update to datastore and write logs while
// doing so. This helps to save work and restart the application in case an update
// fails
type Updater struct {
	dtChannel   chan *fleet.ChopsAsset   // Used to send asset data to registration system
	logFile     *csv.Writer              // Log for all the input data received
	logFilePath string                   // The path of the log file
	resFile     *csv.Writer              // Log for all the input data with success/failure
	resFilePath string                   // The path of the res file
	logFileLoc  string                   // Directory to save all the logs to
	timeStamp   time.Time                // The time that this updater is run and logged
	client      fleetAPI.InventoryClient // RPC client
	ctx         context.Context
	wg          sync.WaitGroup // Used to sync updater thread
}

// Maximum number of the assets in the same location is 10.
const maxAssetPerLocation = 10

func makeFile(logDir, filename string) (*os.File, error) {
	of, err := os.Create(filepath.Join(logDir, filename))
	if err != nil {
		return nil, err
	}
	return of, nil
}

// NewUpdater create a new updater
func NewUpdater(ctx context.Context, c fleetAPI.InventoryClient, logDir string) (u *Updater, err error) {
	// Logfiles are prefixed with timestamp
	curTime := time.Now()
	timestamp := curTime.Format(timeFormat)
	logFileName := timestamp + "-log"
	resFileName := timestamp + "-res"
	logFile, err := makeFile(logDir, logFileName)
	if err != nil {
		return nil, err
	}
	resFile, err := makeFile(logDir, resFileName)
	if err != nil {
		return nil, err
	}

	channel := make(chan *fleet.ChopsAsset, maxAssetPerLocation)
	u = &Updater{
		logFile:     csv.NewWriter(logFile),
		logFilePath: filepath.Join(logDir, logFileName),
		resFile:     csv.NewWriter(resFile),
		resFilePath: filepath.Join(logDir, resFileName),
		timeStamp:   curTime,
		client:      c,
		dtChannel:   channel,
		logFileLoc:  logDir,
		ctx:         ctx,
	}

	// Update waitgroup before starting the routine
	u.wg.Add(1)
	go u.updateRoutine()
	return
}

// AddAsset asynchronously adds asset
func (u *Updater) AddAsset(assetList []*fleet.ChopsAsset) {
	defer u.logFile.Flush()
	for _, a := range assetList {
		// Write input log and then update the channel
		u.logFile.Write(assetToStringList(a))
		u.dtChannel <- a
	}
}

// Close terminates the Updater object
func (u *Updater) Close() {
	close(u.dtChannel)
	u.wg.Wait()
	u.logFile.Flush()
	u.resFile.Flush()
	logStats, err := populateStatistics(u.logFilePath, u.resFilePath, u.timeStamp)
	if err != nil {
		fmt.Printf("Fail to generate statistics for this round of scan: %s\n", err.Error())
	}
	PrintLogStatsAndResult(logStats, 0)
}

/* updateRoutine does the following
*  1. Collects all available assets from dtChannel
*  2. Queries if any of the assets are available on the database
*  3. Adds the assets that are not available on the database
*  4. Updates the assets that exist on the database
*  5. Writes the results to the log
 */
func (u *Updater) updateRoutine() {
	for {
		a, ok := <-u.dtChannel
		if !ok {
			// If the channel is closed. Exit the routine
			u.wg.Done()
			return
		}
		all := make(map[string]*fleet.ChopsAsset)
		all[a.GetId()] = a
		// Collect all the available assets on the channel
		for i := 0; i < len(u.dtChannel); i++ {
			a = <-u.dtChannel
			all[a.GetId()] = a
		}

		// Checkif any of the assets exist in the registration system
		nonExistingAssets, existingAssets, err := u.checkExistingAssets(all)
		if err != nil {
			for _, ast := range all {
				u.logResults(failureState, ast, "GET", err.Error())
			}
			continue
		}

		// Add non-existing assets to registration system
		if len(nonExistingAssets) > 0 {
			addRes, err := u.client.AddAssets(u.ctx, &fleetAPI.AssetList{
				Asset: nonExistingAssets,
			})
			if err != nil {
				// RPC Error, get recorded
				for _, a := range nonExistingAssets {
					u.logResults(failureState, a, "ADD", err.Error())
				}
			}
			// Write to log for both successful and failed transactions
			for _, a := range addRes.Passed {
				u.logResults(successState, a.Asset, "ADD", "")
			}
			for _, a := range addRes.Failed {
				u.logResults(successState, a.Asset, "ADD", a.ErrorMsg)
			}
		}

		// Update existing assets to registration system
		if len(existingAssets) > 0 {
			updateRes, err := u.client.UpdateAssets(u.ctx, &fleetAPI.AssetList{
				Asset: existingAssets,
			})
			if err != nil {
				// RPC Error, get recorded
				for _, a := range existingAssets {
					u.logResults(failureState, a, "UPDATE", err.Error())
				}
				continue
			}
			// Write to log for both successful and failed transactions
			for _, a := range updateRes.Passed {
				u.logResults(successState, all[a.Asset.GetId()], "UPDATE", "")
			}
			for _, a := range updateRes.Failed {
				u.logResults(failureState, all[a.Asset.GetId()], "UPDATE", a.ErrorMsg)
			}
		}
	}
}

func (u *Updater) checkExistingAssets(all map[string]*fleet.ChopsAsset) ([]*fleet.ChopsAsset, []*fleet.ChopsAsset, error) {
	ids := make([]string, 0, len(all))
	for id := range all {
		ids = append(ids, id)
	}
	getRes, err := u.client.GetAssets(u.ctx, &fleetAPI.AssetIDList{
		Id: ids,
	})
	if err != nil {
		return nil, nil, err
	}

	nonExistingAssets := make([]*fleet.ChopsAsset, 0, len(getRes.GetFailed()))
	for _, a := range getRes.Failed {
		nonExistingAssets = append(nonExistingAssets, all[a.Asset.GetId()])
	}
	existingAssets := make([]*fleet.ChopsAsset, 0, len(getRes.GetPassed()))
	for _, a := range getRes.Passed {
		id := a.Asset.GetId()
		existingAssets = append(existingAssets, all[id])
	}
	return nonExistingAssets, existingAssets, nil
}

// Writes log to results log. First entry is Success/Failure, followed by a
// single asset-location entry, a string indicator to indicate if it's
// updation, addition, or get. Then an error message if existing is followed.
// If the asset to log is nil, return immediately.
func (u *Updater) logResults(state string, asset *fleet.ChopsAsset, action, err string) {
	if asset == nil {
		fmt.Println("return: will not log nil asset in result file")
		return
	}
	defer u.resFile.Flush()
	status := []string{state}
	status = append(status, assetToStringList(asset)...)
	status = append(status, action, err)
	u.resFile.Write(status)
}

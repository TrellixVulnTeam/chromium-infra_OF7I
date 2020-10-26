// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"archive/zip"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	v1 "github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/proto"

	kpb "infra/cmd/package_index/kythe/proto"
)

// unitKey is used to identify a unique compilation unit for testing.
type unitKey struct {
	corpus     string
	sourceFile string
}

func TestGnTargets(t *testing.T) {
	t.Parallel()
	Convey("GN Targets", t, func() {
		// Setup.
		chanSize := 10
		numRoutines := 2

		tmpdir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		// Initialize indexPack.
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
		defer cancel()
		kzipPath := filepath.Join(tmpdir, "out.kzip")
		testDir := filepath.Join(cwd, "testdata")
		rootDir := filepath.Join(testDir, "input.expected")
		gnPath := filepath.Join(rootDir, "src", "out", "Debug", "gn_targets.json")
		if err != nil {
			t.Fatal(err)
		}
		ip := newIndexPack(ctx, kzipPath, rootDir, "src/out/Debug", "compdb", gnPath, "kzip",
			"chromium-test", "linux", false)

		// Read expected units and place into a map.
		unitMap := make(map[unitKey]string)
		units, err := ioutil.ReadDir(filepath.Join(testDir, "units.expected"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range units {
			content, err := ioutil.ReadFile(filepath.Join(testDir, "units.expected", f.Name()))
			if err != nil {
				t.Fatal(err)
			}
			indexedCompilation := &kpb.IndexedCompilation{}
			err = v1.UnmarshalText(string(content), indexedCompilation)
			if err != nil {
				t.Fatal(err)
			}
			unit := indexedCompilation.GetUnit()
			unitMap[unitKey{unit.GetVName().GetCorpus(), unit.GetSourceFile()[0]}] = unit.String()
		}

		Convey("Parse and process GN targets", func() {
			// Parse GN targets.
			gnTargets := NewGnTargets(gnPath)
			gnTargets.dataWg.Add(numRoutines)
			gnTargets.kzipDataWg.Add(numRoutines)
			gnTargets.unitWg.Add(numRoutines)

			// Process GN target data files.
			dataFileChannel := make(chan string, chanSize)
			unitProtoChannel := make(chan *kpb.CompilationUnit, chanSize)
			for i := 0; i < numRoutines; i++ {
				go func() {
					// Process GN files.
					err := gnTargets.ProcessGnTargets(ip.ctx, ip.rootPath, ip.outDir, ip.corpus, ip.buildConfig, ip.hashMaps,
						dataFileChannel, unitProtoChannel)
					if err != nil {
						t.Fatal(err)
					}
				}()
			}

			// Convert data files to kzipEntries.
			kzipEntryChannel := make(chan kzipEntry, chanSize)
			for i := 0; i < numRoutines; i++ {
				go func() {
					ip.dataFileToKzipEntry(ctx, dataFileChannel, kzipEntryChannel)

					// Signal done for GN compilation unit processing.
					gnTargets.kzipDataWg.Done()
				}()
			}

			// Convert unit protos to kzipEntries.
			var kzipUnitWg sync.WaitGroup
			kzipUnitWg.Add(numRoutines)
			for i := 0; i < numRoutines; i++ {
				go func() {
					ip.unitFileToKzipEntry(ctx, unitProtoChannel, kzipEntryChannel)
					kzipUnitWg.Done()
				}()
			}

			// Close dataFileChannel and unitProtoChannel after all GN targets have been processed and sent.
			go func() {
				gnTargets.dataWg.Wait()
				close(dataFileChannel)
			}()
			go func() {
				gnTargets.unitWg.Wait()
				close(unitProtoChannel)
			}()

			// Close kzipEntryChannel after all entries have been sent.
			go func() {
				kzipUnitWg.Wait()
				close(kzipEntryChannel)
			}()

			// Wait for kzip writes to finish.
			var writeWg sync.WaitGroup
			writeWg.Add(1)
			go func() {
				err := ip.writeToKzip(kzipEntryChannel)
				if err != nil {
					t.Fatal(err)
				}
				writeWg.Done()
			}()
			writeWg.Wait()

			Convey("Kzip contains files and units for mojom and proto targets", func() {
				// Check kzip exists.
				_, err = os.Stat(kzipPath)
				So(err, ShouldEqual, nil)

				r, err := zip.OpenReader(kzipPath)
				if err != nil {
					t.Fatal(err)
				}
				defer r.Close()

				// Get units and files.
				var unitInfo []*zip.File
				var dataInfo []*zip.File
				for _, zipInfo := range r.File {
					if strings.Contains(zipInfo.Name, "pbunits") && zipInfo.Name != "kzip/pbunits/" {
						unitInfo = append(unitInfo, zipInfo)
					} else if strings.Contains(zipInfo.Name, "files") && zipInfo.Name != "kzip/files/" {
						dataInfo = append(dataInfo, zipInfo)
					}
				}

				// Check generated data files match expected.
				files, _ := ioutil.ReadDir(filepath.Join(testDir, "files.expected"))
				So(len(dataInfo), ShouldEqual, len(files))

				for _, file := range dataInfo {
					rcData, err := file.Open()
					if err != nil {
						t.Fatal(err)
					}
					defer rcData.Close()
					dataContentOut := make([]byte, file.UncompressedSize64)
					_, err = io.ReadFull(rcData, dataContentOut)
					if err != nil {
						t.Fatal(err)
					}

					_, name := filepath.Split(file.Name)
					dataContentExpected, err := ioutil.ReadFile(
						filepath.Join(testDir, "files.expected", name))
					if err != nil {
						t.Fatal(err)
					}

					So(dataContentOut, ShouldResemble, dataContentExpected)
				}

				// Check generated unit protos match expected.
				So(len(unitInfo), ShouldEqual, len(unitMap))
				for _, file := range unitInfo {
					rcUnit, err := file.Open()
					if err != nil {
						t.Fatal(err)
					}
					defer rcUnit.Close()

					unitContentOut := make([]byte, file.UncompressedSize64)
					_, err = io.ReadFull(rcUnit, unitContentOut)
					if err != nil {
						t.Fatal(err)
					}

					indexedCompilationOut := &kpb.IndexedCompilation{}
					err = proto.Unmarshal(unitContentOut, indexedCompilationOut)
					if err != nil {
						t.Fatal(err)
					}
					unitOut := indexedCompilationOut.GetUnit()

					So(unitOut.String(), ShouldResemble,
						unitMap[unitKey{unitOut.GetVName().GetCorpus(), unitOut.GetSourceFile()[0]}])
				}
			})
		})
	})
}

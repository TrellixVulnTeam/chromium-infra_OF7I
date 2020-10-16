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

	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	kpb "infra/cmd/package_index/kythe/proto"
)

func TestWriteToKzip(t *testing.T) {
	t.Parallel()
	Convey("Write entries to kzip", t, func() {
		// Setup.
		chanSize := 10
		numRoutines := 2

		tmpdir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		// Create existing kzip directory.
		existDir := filepath.Join(tmpdir, "existing_kzips/")
		os.Mkdir(existDir, 0777)

		// Initialize indexPack.
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
		defer cancel()
		kzipPath, err := filepath.Abs(filepath.Join(tmpdir, "out.kzip"))
		if err != nil {
			t.Fatal(err)
		}
		ip := newIndexPack(ctx, kzipPath, tmpdir, "src/out", "compdb", "gn", existDir,
			"corpus", "linux", false)

		// Create existing kzip.
		javaKzip, err := os.Create(filepath.Join(existDir, "java.kzip"))
		if err != nil {
			t.Fatal(err)
		}

		w := zip.NewWriter(javaKzip)
		w.Create("kzip/")
		w.Create("kzip/files/")
		w.Create("kzip/units/")

		dataFileContent := []byte("test file")
		dataFileName := "kzip/files/testfile"
		f, err := w.Create(dataFileName)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.Write(dataFileContent)
		if err != nil {
			t.Fatal(err)
		}

		indexedCompilation := &kpb.IndexedCompilation{
			Unit: &kpb.CompilationUnit{
				Argument:  []string{"arg1", "arg2"},
				OutputKey: "key",
			},
		}

		unitFileContent, err := protojson.Marshal(indexedCompilation)
		if err != nil {
			t.Fatal(err)
		}
		f, err = w.Create("kzip/units/testunit")
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.Write(unitFileContent)
		if err != nil {
			t.Fatal(err)
		}
		w.Close()
		javaKzip.Close()

		Convey("Write to kzip", func() {
			// Parse existing java kzips.
			existingKzipChannel := make(chan string, chanSize)
			go func() {
				err := ip.mergeExistingKzips(existingKzipChannel)
				if err != nil {
					t.Fatal(err)
				}
			}()

			// Take existing java kzips and send info to kzipEntryChannel.
			var kzipWg sync.WaitGroup
			kzipEntryChannel := make(chan kzipEntry, chanSize)
			kzipSet := NewConcurrentSet(0)
			kzipWg.Add(numRoutines)
			for i := 0; i < numRoutines; i++ {
				go func() {
					err := ip.processExistingKzips(ctx, existingKzipChannel, kzipEntryChannel, kzipSet)
					if err != nil {
						t.Fatal(err)
					}
					kzipWg.Done()
				}()
			}

			// Close kzipEntryChannel after all entries have been sent.
			go func() {
				kzipWg.Wait()
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

			Convey("Kzip output be well formatted and have files from existing kzip", func() {
				// Check kzip exists.
				_, err = os.Stat(kzipPath)
				So(err, ShouldEqual, nil)

				r, err := zip.OpenReader(kzipPath)
				if err != nil {
					t.Fatal(err)
				}
				defer r.Close()

				var fnames []string
				var dataInfo *zip.File
				var unitInfo *zip.File
				for _, zipInfo := range r.File {
					if strings.Contains(zipInfo.Name, "pbunits") && zipInfo.Name != "kzip/pbunits/" {
						unitInfo = zipInfo
					}
					if strings.Contains(zipInfo.Name, "files") && zipInfo.Name != "kzip/files/" {
						dataInfo = zipInfo
					}
					fnames = append(fnames, zipInfo.Name)
				}

				// Check directory structure.
				So(fnames, ShouldContain, "kzip/")
				So(fnames, ShouldContain, "kzip/files/")
				So(fnames, ShouldContain, "kzip/pbunits/")

				// Check data file.
				So(fnames, ShouldContain, dataFileName)

				rcData, err := dataInfo.Open()
				if err != nil {
					t.Fatal(err)
				}
				defer rcData.Close()

				dataContentOut := make([]byte, dataInfo.UncompressedSize64)
				_, err = io.ReadFull(rcData, dataContentOut)
				if err != nil {
					t.Fatal(err)
				}
				So(dataContentOut, ShouldResemble, dataFileContent)

				// Check unit file.
				So(unitInfo, ShouldNotEqual, nil)
				rcUnit, err := unitInfo.Open()
				if err != nil {
					t.Fatal(err)
				}
				defer rcUnit.Close()

				content := make([]byte, unitInfo.UncompressedSize64)
				_, err = io.ReadFull(rcUnit, content)
				if err != nil {
					t.Fatal(err)
				}

				indexedCompilationOut := &kpb.IndexedCompilation{}
				err = proto.Unmarshal(content, indexedCompilationOut)
				if err != nil {
					t.Fatal(err)
				}

				injectUnitBuildDetails(ctx, indexedCompilation.GetUnit(), ip.buildConfig)

				So(indexedCompilationOut.GetUnit().String(), ShouldEqual, indexedCompilation.GetUnit().String())
			})
		})
	})
}

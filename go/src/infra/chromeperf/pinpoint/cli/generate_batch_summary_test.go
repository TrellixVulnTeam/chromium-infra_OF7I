// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"fmt"
	"infra/chromeperf/pinpoint/proto"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"
)

func validateFile(path, expected string) {
	_, err := os.Stat(path)
	So(err, ShouldBeNil)

	content, err := ioutil.ReadFile(path)
	So(err, ShouldBeNil)
	fmt.Println("Actual:\n" + string(content) + "\n\n")
	fmt.Println("Expected:\n" + expected + "\n\n")
	fmt.Println(cmp.Diff(string(content), expected))
	So(cmp.Equal(string(content), expected), ShouldBeTrue)
}

func TestGenerateCSVsFromBatch(t *testing.T) {
	Convey("A batch .txt and a preset should generate a set of .csvs", t, func() {
		jobs := []*proto.Job{
			{
				Name: "jobs/legacy-15865492320000",
			},
			{
				Name: "jobs/legacy-14cd7aa2320000",
			},
			{
				Name: "jobs/legacy-162f8892320000",
			},
		}

		p := preset{}
		report := make(map[string]batchSummaryReportSpec)
		metrics := []batchSummaryReportMetric{
			{
				Name: "largestContentfulPaint",
			},
			{
				Name: "timeToFirstContentfulPaint",
			},
			{
				Name: "overallCumulativeLayoutShift",
			},
			{
				Name: "totalBlockingTime",
			},
		}
		p.BatchSummaryReportSpec = &report
		report["loading.desktop"] = batchSummaryReportSpec{
			Metrics: &metrics,
		}
		report["loading.mobile"] = batchSummaryReportSpec{
			Metrics: &metrics,
		}

		summarizer := generateBatchSummary{}
		summarizer.baseCommandRun.workDir = "testdata/generate-batch-summary-artifacts"

		csvDir, err := ioutil.TempDir("", "tmp_batch_analyze")
		if err != nil {
			log.Fatal(err)
		}

		err = summarizer.analyzeArtifactsAndGenerateCSVs(csvDir, p, "fakebatch", jobs)
		So(err, ShouldBeNil)

		csvDir = filepath.Join(csvDir, "fakebatch_summaries")
		validateFile(filepath.Join(csvDir, "cover.csv"),
			",A,B\n"+
				"Commit,https://chromium-review.googlesource.com/q/63bd8c0402f260c28a5e7d9dd3f8ffd46028e68a,https://chromium-review.googlesource.com/q/b0b3650d9d5f3ba05267a16913534f0fcca0a688\n"+
				"Applied Change,,https://chromium-review.googlesource.com/q/2776374/2\n")
		validateFile(filepath.Join(csvDir, "loading.desktop.csv"),
			"URL,Cfg,Story,,largestContentfulPaint,,,timeToFirstContentfulPaint,,,overallCumulativeLayoutShift,,,totalBlockingTime,,\n"+
				",,,pval,Median (A),Median (B),pval,Median (A),Median (B),pval,Median (A),Median (B),pval,Median (A),Median (B)\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/14cd7aa2320000,mac-10_12_laptop_low_end-perf,Naver_warm,0.046429,626.00000,653.00000,0.247451,144.56300,144.08800,0.637186,0.00020,0.00000,NaN,0.00000,0.00000\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/162f8892320000,linux-perf,Yandex_cold,0.015656,154.00000,138.00000,0.528849,85.19700,109.19300,NaN,0.05790,0.05790,NaN,0.00000,0.00000\n")
		validateFile(filepath.Join(csvDir, "loading.mobile.csv"),
			"URL,Cfg,Story,,largestContentfulPaint,,,timeToFirstContentfulPaint,,,overallCumulativeLayoutShift,,,totalBlockingTime,,\n"+
				",,,pval,Median (A),Median (B),pval,Median (A),Median (B),pval,Median (A),Median (B),pval,Median (A),Median (B)\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/15865492320000,android-pixel2-perf,Amazon,0.241588,356.00000,403.00000,0.481251,309.44700,323.92100,NaN,0.00000,0.00000,NaN,0.00000,0.00000\n")

		content, err := ioutil.ReadFile("testdata/generate-batch-summary-raw-expected.csv")
		So(err, ShouldBeNil)
		validateFile(filepath.Join(csvDir, "raw.csv"), string(content))
	})
}

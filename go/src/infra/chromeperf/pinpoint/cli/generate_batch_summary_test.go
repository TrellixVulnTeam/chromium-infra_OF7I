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
		validateFile(filepath.Join(csvDir, "raw.csv"),
			"URL,Benchmark,Cfg,Story,Metric,pval,Median (A),Median (B),Raw (A),Raw (B)\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/15865492320000,loading.mobile,android-pixel2-perf,Amazon,largestContentfulPaint,0.241588,356.00000,403.00000,306.000000;306.000000;310.000000;310.000000;317.000000;317.000000;317.000000;317.000000;356.000000;356.000000;357.000000;357.000000;374.000000;374.000000;480.000000;480.000000;490.000000;490.000000;550.000000;550.000000,319.000000;319.000000;324.000000;324.000000;347.000000;347.000000;353.000000;353.000000;403.000000;403.000000;407.000000;407.000000;416.000000;416.000000;438.000000;438.000000;443.000000;443.000000;554.000000;554.000000\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/15865492320000,loading.mobile,android-pixel2-perf,Amazon,timeToFirstContentfulPaint,0.481251,309.44700,323.92100,271.870000;273.602000;284.188000;306.157000;309.447000;316.958000;317.513000;356.494000;357.496000;374.003000,277.624000;282.523000;312.162000;319.100000;323.921000;328.232000;344.661000;346.755000;353.503000;367.622000\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/15865492320000,loading.mobile,android-pixel2-perf,Amazon,overallCumulativeLayoutShift,NaN,0.00000,0.00000,,\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/15865492320000,loading.mobile,android-pixel2-perf,Amazon,totalBlockingTime,NaN,0.00000,0.00000,,\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/14cd7aa2320000,loading.desktop,mac-10_12_laptop_low_end-perf,Naver_warm,largestContentfulPaint,0.046429,626.00000,653.00000,608.000000;608.000000;608.000000;614.000000;614.000000;614.000000;614.000000;614.000000;614.000000;615.000000;615.000000;615.000000;626.000000;626.000000;626.000000;637.000000;637.000000;637.000000;648.000000;648.000000;648.000000;652.000000;652.000000;652.000000;668.000000;668.000000;668.000000;690.000000;690.000000;690.000000,600.000000;600.000000;600.000000;611.000000;611.000000;611.000000;641.000000;641.000000;641.000000;645.000000;645.000000;645.000000;653.000000;653.000000;653.000000;658.000000;658.000000;658.000000;667.000000;667.000000;667.000000;676.000000;676.000000;676.000000;684.000000;684.000000;684.000000;888.000000;888.000000;888.000000\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/14cd7aa2320000,loading.desktop,mac-10_12_laptop_low_end-perf,Naver_warm,timeToFirstContentfulPaint,0.247451,144.56300,144.08800,130.636000;134.966000;140.643000;142.619000;144.563000;147.758000;148.146000;149.817000;153.711000;154.822000,127.773000;129.117000;139.965000;141.102000;144.088000;144.304000;146.576000;147.083000;147.191000;148.034000\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/14cd7aa2320000,loading.desktop,mac-10_12_laptop_low_end-perf,Naver_warm,overallCumulativeLayoutShift,0.637186,0.00020,0.00000,0.000000;0.000000;0.000000;0.000200;0.000200;0.000400;0.000400;0.000400;0.000400;0.000400,0.000000;0.000000;0.000000;0.000000;0.000000;0.000200;0.000400;0.000400;0.000400;0.000400\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/14cd7aa2320000,loading.desktop,mac-10_12_laptop_low_end-perf,Naver_warm,totalBlockingTime,NaN,0.00000,0.00000,,\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/162f8892320000,loading.desktop,linux-perf,Yandex_cold,largestContentfulPaint,0.015656,154.00000,138.00000,137.000000;137.000000;137.000000;137.000000;137.000000;140.000000;140.000000;140.000000;140.000000;140.000000;142.000000;142.000000;142.000000;142.000000;142.000000;152.000000;152.000000;152.000000;152.000000;152.000000;154.000000;154.000000;154.000000;154.000000;154.000000;158.000000;158.000000;158.000000;158.000000;158.000000;160.000000;160.000000;160.000000;160.000000;160.000000;162.000000;162.000000;162.000000;162.000000;162.000000;179.000000;179.000000;179.000000;179.000000;179.000000;201.000000;201.000000;201.000000;201.000000;201.000000,125.000000;125.000000;125.000000;125.000000;125.000000;128.000000;128.000000;128.000000;128.000000;128.000000;134.000000;134.000000;134.000000;134.000000;134.000000;138.000000;138.000000;138.000000;138.000000;138.000000;138.000000;138.000000;138.000000;138.000000;138.000000;154.000000;154.000000;154.000000;154.000000;154.000000;154.000000;154.000000;154.000000;154.000000;154.000000;165.000000;165.000000;165.000000;165.000000;165.000000;173.000000;173.000000;173.000000;173.000000;173.000000;182.000000;182.000000;182.000000;182.000000;182.000000\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/162f8892320000,loading.desktop,linux-perf,Yandex_cold,timeToFirstContentfulPaint,0.528849,85.19700,109.19300,77.532000;77.952000;78.922000;85.134000;85.197000;87.281000;104.633000;136.349000;140.701000;142.019000,79.178000;80.168000;81.279000;95.925000;109.193000;112.495000;125.211000;127.888000;138.119000;138.590000\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/162f8892320000,loading.desktop,linux-perf,Yandex_cold,overallCumulativeLayoutShift,NaN,0.05790,0.05790,0.057900;0.057900;0.057900;0.057900;0.057900;0.057900;0.057900;0.057900;0.057900;0.057900,0.057900;0.057900;0.057900;0.057900;0.057900;0.057900;0.057900;0.057900;0.057900;0.057900\n"+
				"https://pinpoint-dot-chromeperf.appspot.com/job/162f8892320000,loading.desktop,linux-perf,Yandex_cold,totalBlockingTime,NaN,0.00000,0.00000,,\n")
	})
}

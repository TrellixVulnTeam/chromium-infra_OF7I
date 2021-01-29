// Copyright 2014 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// ninja_log_trace_viewer converts .ninja_log into trace-viewer formats.
//
// usage:
//  $ go run ninja_log_trace_viewer.go --filename out/Release/.ninja_log --output trace.json
//
//  $ go run ninja_log_trace_viewer.go \
//    --filename out/Release/.ninja_log --browser
//
package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"time"

	"infra/appengine/chromium_build_stats/ninjalog"
)

var (
	filename   = flag.String("filename", ".ninja_log", "filename of .ninja_log")
	traceJSON  = flag.String("trace_json", "", "output filename as trace.json")
	browser    = flag.String("browser", "x-www-browser", "browser to launch")
	cpuprofile = flag.String("cpuprofile", "", "file to write cpu profile")
)

// ui.perfetto.dev allows http://127.0.0.1:9001
const jsonPort = 9001

func reader(fname string, rd io.Reader) (io.Reader, error) {
	if filepath.Ext(fname) != ".gz" {
		return bufio.NewReaderSize(rd, 512*1024), nil
	}
	return gzip.NewReader(bufio.NewReaderSize(rd, 512*1024))
}

func convert(fname string) ([]ninjalog.Trace, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rd, err := reader(fname, f)
	if err != nil {
		return nil, err
	}

	njl, err := ninjalog.Parse(fname, rd)
	if err != nil {
		return nil, err
	}
	steps := ninjalog.Dedup(njl.Steps)
	flow := ninjalog.Flow(steps, false)
	return ninjalog.ToTraces(flow, 1), nil
}

func output(fname string, js []byte) (err error) {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()
	_, err = f.Write(js)
	return err
}

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	traces, err := convert(*filename)
	if err != nil {
		log.Fatal(err)
	}
	js, err := json.Marshal(traces)
	if err != nil {
		log.Fatal(err)
	}
	if *traceJSON != "" {
		if err = output(*traceJSON, js); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Generated trace json to %s\n", *traceJSON)
		return
	}
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, "ok")
	})
	http.HandleFunc("/trace.json", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "https://ui.perfetto.dev")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(js)
		if err != nil {
			log.Print(err)
		}
	})
	jsonURL := fmt.Sprintf("http://127.0.0.1:%d", jsonPort)
	viewURL := fmt.Sprintf("https://ui.perfetto.dev/#!/?url=%s", jsonURL)
	if *browser != "" {
		go func() {
			for {
				if _, err = http.Get(jsonURL); err == nil {
					break
				}
				fmt.Printf("waiting for %s\n", jsonURL)
				time.Sleep(1 * time.Second)
			}
			if err = exec.Command(*browser, viewURL).Run(); err != nil {
				log.Fatal(err)
			}
		}()
	}
	fmt.Printf("listening on :%d\n", jsonPort)
	fmt.Printf("you can browse trace view on %s\n", viewURL)
	if err = http.ListenAndServe(fmt.Sprintf(":%d", jsonPort), nil); err != nil {
		log.Fatal(err)
	}
}

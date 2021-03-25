// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/*
The labpack program allows to run repair tasks for ChromeOS devices in the lab.
For more information please read go/AdminRepair.
Managed by Chrome Fleet Software (go/chrome-fleet-software).
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.SetPrefix(fmt.Sprintf("%s: ", filepath.Base(os.Args[0])))
	log.Printf("Starting with args: %s", os.Args)
	a := parseArgs()
	out, err := json.Marshal(a)
	if err != nil {
		log.Printf("Error: %s", err)
	}
	log.Printf("Parsed args: %s", string(out))
	log.Printf("Exited successfully")
}

type args struct {
	isolatedOutdir      string
	logdogAnnotationURL string
	taskName            string
}

func parseArgs() *args {
	a := &args{}

	flag.StringVar(&a.taskName, "task-name", "",
		"Name of the task to run.")
	flag.StringVar(&a.logdogAnnotationURL, "logdog-annotation-url", "",
		"LogDog annotation URL, like logdog://HOST/PROJECT/PREFIX/+/annotations")
	flag.StringVar(&a.isolatedOutdir, "isolated-outdir", "",
		"Directory to place isolated output into. Generate no isolated output if not set.")
	flag.Parse()

	return a
}

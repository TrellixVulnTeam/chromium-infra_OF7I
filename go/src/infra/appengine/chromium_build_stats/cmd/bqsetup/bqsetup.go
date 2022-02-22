// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build ignore

// Binary bqsetup is a tool that create/update BigQuery table/transfer config for ninja log.
//
// Usage:
//  $ go run bqsetup.go -project chromium-build-stats-staging -table test
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"infra/appengine/chromium_build_stats/ninjalog"
)

var (
	project    = flag.String("project", "", "project ID setup BigQuery table & transfer config for ninja log")
	table      = flag.String("table", "", "table name used to store ninja log")
	initialize = flag.Bool("initialize", false, "DANGEROUS: This re-creates BigQuery table if set.")
)

func main() {
	flag.Parse()

	if *project == "" {
		fmt.Println("project is not set")
		os.Exit(1)
	}

	if *table == "" {
		fmt.Println("table is not set")
		os.Exit(1)
	}

	ctx := context.Background()

	if *initialize {
		fmt.Printf("You are re-creating table %s in project %s.\n", *table, *project)
		fmt.Printf("Enter %s.%s to confirm: ", *project, *table)
		var projTable string
		fmt.Scan(&projTable)

		if projTable != *project+"."+*table {
			fmt.Printf("wrong project/table pair is given: %q\n", projTable)
			os.Exit(1)
		}

		if err := ninjalog.CreateBQTable(ctx, *project, *table); err != nil {
			fmt.Printf("failed to create BigQuery table for project %s table %s: %v\n", *project, *table, err)
			os.Exit(1)
		}
	} else {
		if err := ninjalog.UpdateBQTable(ctx, *project, *table); err != nil {
			fmt.Printf("failed to update BigQuery table for project %s table %s: %v\n", *project, *table, err)
			os.Exit(1)
		}
	}

	if _, err := ninjalog.CreateTransferConfig(ctx, *project, *table); err != nil {
		fmt.Printf("failed to create BigQuery transfer for project %s table %s: %v\n", *project, *table, err)
		os.Exit(1)
	}
}

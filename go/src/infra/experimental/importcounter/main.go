// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command importcounter will calculate and print per-package and aggregate metrics
// about Go dependencies. Ex:
//
//   go run main.go --patterns go.chromium.org/luci/...
//
// Will print CSV to stdout, listing each subpackage of go.chromium.org/luci/...
// along with some per-subpackge counts on each row. Finally, it will produce
// some aggregate counts for the entire set of packages.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

var (
	patterns stringListValue
)

func init() {
	flag.Var(&patterns, "patterns", "package path patterns to examine")
}

type stats struct {
	// PackageCount is the number of packages analyzed.
	PackageCount int
	// TotalImportCount is the total number of import statements,
	// treating each line inside a factored import block as an individual import.
	TotalImportCount int
	// DistinctImportCount is the total number of distinct packages imported.
	DistinctImportCount int
	// TotalExternalImportCount is the total number of import statements
	// for packages outside of the path patterns specified in the patterns
	// flag.
	TotalExternalImportCount int
	// DistinctExternalImportCount is the total number of distinct external
	// see TotalExternalImportCount above) packages imported.
	DistinctExternalImportCount int
	// FileCount is the number of files analyzed.
	FileCount int
	// LOC is the total number of lines of code in the analyzed files.
	LOC int
}

// loc returns the total lines of code contained in the go source files
// located in goFiles.
func loc(goFiles []string) int {
	sum := 0
	for _, fn := range goFiles {
		data, err := ioutil.ReadFile(fn)
		if err != nil {
			log.Fatalf("reading %s: %+v", fn, err)
		}
		sum += len(strings.Split(string(data), "\n"))
	}
	return sum
}

// patMatch returns true if pkg is a package name that equals or falls under
// pat, if pat ends with "/...".
func patMatch(pat string, pkg string) bool {
	// TODO: something more advanced. This won't work in every case.
	prefix := strings.Replace(pat, "/...", "", -1)
	return strings.HasPrefix(pkg, prefix)
}

// countExternal returns the number of package names (keys) in pkgs that do not match the
// patterns in in patterns.
func countExternal(patterns []string, pkgs map[string]*packages.Package) int {
	ret := 0
	for k := range pkgs {
		external := false
		for _, pat := range patterns {
			if !patMatch(pat, k) {
				external = true
			}
		}
		if external {
			ret++
		}
	}
	return ret
}

func main() {
	flag.Parse()
	cfg := &packages.Config{
		// Consider using packages.NeedDeps if we want more info about imported packages.
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		log.Fatalf("loading packages: %+v", err)
	}
	sum := stats{
		PackageCount: len(pkgs),
	}

	importSet := map[string]*packages.Package{}
	out := [][]string{}
	for _, p := range pkgs {
		totalImports := len(p.Imports)
		totalExternalImports := countExternal(patterns, p.Imports)
		totalLOC := loc(p.GoFiles)
		fileCount := len(p.GoFiles)

		sum.TotalImportCount += totalImports
		sum.TotalExternalImportCount += totalExternalImports
		for _, imp := range p.Imports {
			importSet[imp.PkgPath] = imp
		}
		sum.FileCount += fileCount
		sum.LOC += totalLOC
		out = append(out, []string{
			p.PkgPath,
			fmt.Sprintf("%d", totalImports),
			fmt.Sprintf("%d", totalExternalImports),
			fmt.Sprintf("%d", fileCount),
			fmt.Sprintf("%d", totalLOC),
		})
	}

	w := csv.NewWriter(os.Stdout)
	w.WriteAll(out)

	sum.DistinctImportCount = len(importSet)
	sum.DistinctExternalImportCount = countExternal(patterns, importSet)
	fmt.Printf("Aggregate: %+v\n", sum)
}

// stringListValue is a flag.Value that accumulates strings.
// e.g. --flag=one --flag=two would produce []string{"one", "two"}.
type stringListValue []string

func newStringListValue(val []string, p *[]string) *stringListValue {
	*p = val
	return (*stringListValue)(p)
}

func (ss *stringListValue) Get() interface{} { return []string(*ss) }

func (ss *stringListValue) String() string { return fmt.Sprintf("%q", *ss) }

func (ss *stringListValue) Set(s string) error { *ss = append(*ss, s); return nil }

// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package git

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"infra/rts/filegraph"
	"infra/rts/filegraph/internal/gitutil"

	. "github.com/smartystreets/goconvey/convey"
)

// BenchmarkE2E measures performance of this package end to end:
//  - Builds the graph from a git checkout
//  - Serializes it.
//  - Deserializes it.
//  - Runs queries from some files
//
// Uses the git checkout at path in the FILEGRAPH_BENCH_CHECKOUT environment
// variable. If it is not defined, then uses current repo (luci-go.git).
func BenchmarkE2E(b *testing.B) {
	ctx := context.Background()
	repoDir := benchRepoDir(b)

	var g *Graph

	// First, build the graph from scratch.
	b.Run("Build", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			g = &Graph{}
			if err := g.Update(ctx, repoDir, "refs/heads/master", UpdateOptions{}); err != nil {
				b.Fatal(err)
			}
		}
		b.StopTimer()
		printStats(g, b)
	})

	// Serialize it.
	buf := &bytes.Buffer{}
	b.Run("Write", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf.Reset()
			if err := g.Write(buf); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Deserialize it.
	b.Run("Read", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			r := bufio.NewReader(bytes.NewReader(buf.Bytes()))
			if err := g.Read(r); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Run queries for each top-level file.
	for _, n := range g.root.children {
		n := n
		if len(n.children) > 0 {
			continue
		}
		b.Run("Query-"+strings.TrimPrefix(n.name, "//"), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				q := filegraph.Query{Sources: []filegraph.Node{n}, EdgeReader: &EdgeReader{}}
				q.Run(func(*filegraph.ShortestPath) bool {
					return true
				})
			}
		})
	}
}

func TestE2E(t *testing.T) {
	t.Parallel()

	repoDir := benchRepoDir(t)
	Convey(`E2E`, t, func() {
		ctx := context.Background()

		// Build the graph from scratch.
		g := &Graph{}
		err := g.Update(ctx, repoDir, "refs/remotes/origin/master", UpdateOptions{})
		So(err, ShouldBeNil)

		// Ensure each file in the repo is also present in the graph.
		gitListFiles(ctx, repoDir, "origin/master", func(file string) {
			n := g.node("//" + file)
			t.Log(file)
			So(n, ShouldNotBeNil)
		})
	})
}

func gitListFiles(ctx context.Context, dir, ref string, callback func(file string)) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "ls-files", ref)
	stdout, err := cmd.StdoutPipe()
	So(err, ShouldBeNil)
	So(cmd.Start(), ShouldBeNil)

	scan := bufio.NewScanner(stdout)
	for scan.Scan() {
		callback(scan.Text())
	}
	So(scan.Err(), ShouldBeNil)
	So(cmd.Wait(), ShouldBeNil)
}

func printStats(g *Graph, b *testing.B) {
	nodes := 0
	edges := 0
	g.root.visit(func(n *node) bool {
		nodes++
		edges += len(n.edges)
		return true
	})
	b.Logf("%d nodes, %d edges", nodes, edges)
}

func benchRepoDir(b testing.TB) string {
	if repoDir := os.Getenv("FILEGRAPH_BENCH_CHECKOUT"); repoDir != "" {
		return repoDir
	}

	cwd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	repoDir, err := gitutil.TopLevel(cwd)
	if err != nil {
		b.Fatal(err)
	}
	return repoDir
}

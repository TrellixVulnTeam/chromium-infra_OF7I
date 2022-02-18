// Copyright 2022 The LUCI Authors.
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

package main

import (
	"context"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	if len(os.Args) < 2 {
		log.Fatalf("not enough arguments passed")
	}
	// Check for the presence of "--" between the wrapper & the cmd
	// bbagent already resolves the cmd to absolute path
	args := os.Args[1:]
	if args[0] != "--" {
		log.Fatalf("there should be  a `--` between the wrapper and the cmd")
	}
	args = args[1:]

	if err := RunInNsjail(ctx, args); err != nil {
		log.Fatalf("running command: %s", err.Error())
	}
}

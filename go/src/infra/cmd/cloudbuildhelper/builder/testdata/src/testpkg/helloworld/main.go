// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	fmt.Printf("Hello, world: %q\n", os.Args)
	if len(os.Args) > 1 {
		if err := ioutil.WriteFile(os.Args[1], []byte("Hello!"), 0600); err != nil {
			panic(err)
		}
	}
}

// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package templates contains embedded plugin template content.
package templates

import (
	"embed"
	"fmt"
	"io/ioutil"
	"path"
)

//go:embed _plugin/*
//go:embed default.cfg
var content embed.FS

// Plugin returns plugin template files.
func Plugin() map[string][]byte {
	const pluginRoot = "_plugin"

	dir, err := content.ReadDir(pluginRoot)
	if err != nil {
		panic(fmt.Sprintf("failed to scan %q: %s", pluginRoot, err))
	}

	out := make(map[string][]byte, len(dir))
	for _, ent := range dir {
		if ent.IsDir() {
			panic("You added a directory to _plugin and need to change the code now to scan it recursively. Good luck!")
		}
		out[ent.Name()] = bodyOf(path.Join(pluginRoot, ent.Name()))
	}

	return out
}

// Config returns a config file template body.
func Config() []byte {
	return bodyOf("default.cfg")
}

// bodyOf returns a body of an embedded file (which must exist).
func bodyOf(path string) []byte {
	f, err := content.Open(path)
	if err != nil {
		panic(fmt.Sprintf("%q is not embedded: %s", path, err))
	}
	defer f.Close()
	body, err := ioutil.ReadAll(f)
	if err != nil {
		panic(fmt.Sprintf("failed to read embedded %q: %s", path, err))
	}
	return body
}

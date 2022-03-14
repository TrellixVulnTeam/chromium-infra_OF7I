// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package model

type CompileLogs struct {
	NinjaLog  *NinjaLog
	StdOutLog string
}

type NinjaLog struct {
	Failures []*NinjaLogFailure `json:"failures"`
}

type NinjaLogFailure struct {
	Dependencies []string `json:"dependencies"`
	Output       string   `json:"output"`
	OutputNodes  []string `json:"output_nodes"`
	Rule         string   `json:"rule"`
}

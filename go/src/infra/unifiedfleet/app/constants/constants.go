// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package constants

// Global integers
const (
	DefaultPageSize int32 = 100
	MaxPageSize     int32 = 1000
)

// Global Constant Strings
const (
	NilEntity         string = "Invalid input - no Entity to add/update"
	EmptyID           string = "Invalid input - Entity ID/Name is empty"
	InvalidCharacters string = "Invalid input - Entity ID/Name must contain only 4-63 characters, ASCII letters, numbers, dash and underscore."
	InvalidPageSize   string = "Invalid input - page_size should be non-negative"
	InvalidPageToken  string = "Invalid Page Token."
	AlreadyExists     string = "Entity already exists."
	NotFound          string = "Entity not found."
	InternalError     string = "Internal Server Error"
)

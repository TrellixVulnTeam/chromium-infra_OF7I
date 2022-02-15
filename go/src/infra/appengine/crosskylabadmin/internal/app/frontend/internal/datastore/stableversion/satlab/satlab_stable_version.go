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

package satlab

const SatlabStableVersionKind = "satlab_stable_version"

// SatlabStableVersionEntry is a stable version entry.
type SatlabStableVersionEntry struct {
	_kind string `gae:"$kind,satlab_stable_version"`
	// ID has the following format. Earlier entries have precedence over later ones.
	//
	//   hostname     -- for a version scoped to a specific device.
	//   board|model  -- for a version scoped to just the board and the model.
	//
	ID string `gae:"$id"`
	// Base64Req stores a base64-encoded SetSatlabStableVersionRequest.
	// This request can be used to reconstitute this entire entry if required.
	Base64Req string `gae:"base64_req"`
	OS        string `gae:"os"`
	FW        string `gae:"fw"`
	FWImage   string `gae:"fw_image"`
}

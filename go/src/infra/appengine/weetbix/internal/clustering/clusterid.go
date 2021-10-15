// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import (
	"encoding/hex"
	"fmt"
)

// ClusterID represents the identity of a cluster. The LUCI Project is
// omitted as it is assumed to be implicit from the context.
type ClusterID struct {
	// Algorithm is the name of the clustering algorithm that identified
	// the cluster.
	Algorithm string
	// ID is the cluster identifier returned by the algorithm. This is
	// at most 16 bytes.
	ID []byte
}

// Key returns a value that can be used to uniquely identify the Cluster.
// This is designed for cases where it is desirable for cluster IDs
// to be used as keys in a map.
func (c *ClusterID) Key() string {
	return fmt.Sprintf("%s:%s", c.Algorithm, hex.EncodeToString(c.ID))
}

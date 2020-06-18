// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/google"
)

type state struct {
	r result_flow.State
	e error
}

func runWithDeadline(ctx context.Context, f func(chan state)) (result_flow.State, error) {
	ch := make(chan state, 1)
	go f(ch)
	select {
	case <-ctx.Done():
		return result_flow.State_TIMED_OUT, fmt.Errorf("ctp command hit the deadline")
	case res := <-ch:
		return res.r, res.e
	}
}

// readJSONPb reads a JSON string from inFile and unpacks it as a proto.
// Unexpected fields are ignored.
func readJSONPb(inFile string, payload proto.Message) error {
	r, err := os.Open(inFile)
	if err != nil {
		return errors.Annotate(err, "read JSON pb").Err()
	}
	defer r.Close()

	unmarshaler := jsonpb.Unmarshaler{AllowUnknownFields: true}
	if err := unmarshaler.Unmarshal(r, payload); err != nil {
		return errors.Annotate(err, "read JSON pb").Err()
	}
	return nil
}

// writeJSONPb writes a JSON encoded proto to outFile.
func writeJSONPb(outFile string, payload proto.Message) error {
	dir := filepath.Dir(outFile)
	// Create the directory if it doesn't exist.
	if err := os.MkdirAll(dir, 0777); err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}

	w, err := os.Create(outFile)
	if err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}
	defer w.Close()

	marshaler := jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, payload); err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}
	return nil
}

func getDeadline(rq *timestamp.Timestamp, defaultTimeoutSec int) time.Time {
	if rq != nil {
		return google.TimeFromProto(rq)
	}
	return time.Now().Add(time.Second * time.Duration(defaultTimeoutSec))
}

func verifySource(s *result_flow.Source) (*result_flow.Source, error) {
	var missing []string
	if s.GetPubsub() != nil {
		missing = verifySubscription(s.GetPubsub(), missing)
	} else {
		missing = append(missing, "source PubSub config")
	}

	if s.GetBb() != nil {
		missing = verifyBuildbucket(s.GetBb(), missing)
	} else {
		missing = append(missing, "source Buildbucket Config")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("Found missing fields: %s", strings.Join(missing, ", "))
	}
	return s, nil
}

func verifySubscription(s *result_flow.PubSubConfig, missing []string) []string {
	if s.GetProject() == "" {
		missing = append(missing, "source PubSub Project")
	}
	if s.GetSubscription() == "" {
		missing = append(missing, "source PubSub Subscription")
	}
	return missing
}

func verifyBuildbucket(s *result_flow.BuildbucketConfig, missing []string) []string {
	if s.GetHost() == "" {
		missing = append(missing, "source Buildbucket Host")
	}
	if s.GetProject() == "" {
		missing = append(missing, "source Buildbucket Project")
	}
	if s.GetBucket() == "" {
		missing = append(missing, "source Buildbucket Bucket")
	}
	if s.GetBuilder() == "" {
		missing = append(missing, "source Buildbucket Builder")
	}
	return missing
}

func verifyTarget(t *result_flow.Target) (*result_flow.Target, error) {
	var missing []string
	if t.GetBq() != nil {
		missing = verifyBigquery(t.GetBq(), missing)
	} else {
		missing = append(missing, "target Bigquery Config")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("Found missing fields: %s", strings.Join(missing, ", "))
	}
	return t, nil
}

func verifyBigquery(t *result_flow.BigqueryConfig, missing []string) []string {
	if t.GetProject() == "" {
		missing = append(missing, "target Bigquery Project")
	}
	if t.GetDataset() == "" {
		missing = append(missing, "target Bigquery Dataset")
	}
	if t.GetTable() == "" {
		missing = append(missing, "target Bigquery Table")
	}
	return missing
}

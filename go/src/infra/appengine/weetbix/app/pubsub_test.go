// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"sort"

	cvv0 "go.chromium.org/luci/cv/api/v0"

	"infra/appengine/weetbix/internal/tasks/taskspb"

	// Needed to ensure task class is registered.
	_ "infra/appengine/weetbix/internal/services/resultingester"
)

func makeReq(blob []byte) io.ReadCloser {
	msg := struct {
		Message struct {
			Data []byte
		}
		Attributes map[string]interface{}
	}{struct{ Data []byte }{Data: blob}, nil}
	jmsg, _ := json.Marshal(msg)
	return ioutil.NopCloser(bytes.NewReader(jmsg))
}

func expectedTasks(run *cvv0.Run) []*taskspb.IngestTestResults {
	res := make([]*taskspb.IngestTestResults, 0, len(run.Tryjobs))
	for _, tj := range run.Tryjobs {
		if tj.GetResult() == nil {
			continue
		}
		t := &taskspb.IngestTestResults{
			CvRun: run,
			Build: &taskspb.Build{
				Host: bbHost,
				Id:   tj.GetResult().GetBuildbucket().GetId(),
			},
			PartitionTime: run.CreateTime,
		}
		res = append(res, t)
	}
	return res
}

func sortTasks(tasks []*taskspb.IngestTestResults) []*taskspb.IngestTestResults {
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Build.Id < tasks[j].Build.Id })
	return tasks
}

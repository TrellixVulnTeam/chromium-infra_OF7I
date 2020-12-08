// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	cloudtask "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
)

const (
	cloudTaskGoRoutines int = 50
	saveCursorAfter     int = 10000
)

var issuePrivateCache IssuePrivateCacheMap

// CloudTaskPayload represents document in CloudTask (payload)
type CloudTaskPayload struct {
	Path       string
	Private    bool
	EntityKind string
}

type taskProcessor struct {
	ctClient *cloudtask.Client
	ch       <-chan CloudTaskPayload

	ctQueuePath           string
	processPageURL        string
	processServiceAccount string
}

// processCloudTaskQueue drains channel and creates Cloud Task.
func (tp *taskProcessor) process(ctx context.Context) {
	for cloudTaskPayload := range tp.ch {
		payload, err := json.Marshal(cloudTaskPayload)
		if err != nil {
			log.Panic(err)
		}

		// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#CreateTaskRequest
		req := &taskspb.CreateTaskRequest{
			Parent: tp.ctQueuePath,
			Task: &taskspb.Task{
				// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#HttpRequest
				MessageType: &taskspb.Task_HttpRequest{
					HttpRequest: &taskspb.HttpRequest{
						HttpMethod: taskspb.HttpMethod_POST,
						Url:        tp.processPageURL,
						Body:       payload,
						AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
							OidcToken: &taskspb.OidcToken{
								ServiceAccountEmail: tp.processServiceAccount,
							},
						},
					},
				},
			},
		}

		_, err = tp.ctClient.CreateTask(ctx, req)
		if err != nil {
			log.Panic(err)
		}
	}
}

// scanner queries entire datastore Kind and mutates `doc` with results.
func scanner(ctx context.Context, dsClient *datastore.Client, kind string,
	doc interface{}, fields []string, cb func(key *datastore.Key)) {
	log.Printf("Scanning %s", kind)
	defer log.Printf("Done scanning %s", kind)

	cs := NewCursorSaver(ctx, dsClient, kind)
	cursor := cs.RestoreCursor(ctx)
	q := datastore.NewQuery(kind)
	if cursor != nil {
		q.Start(*cursor)
	}
	if len(fields) == 0 {
		q = q.KeysOnly()
	} else {
		q = q.Project(fields...)
	}
	t := dsClient.Run(ctx, q)
	for i := 1; ; i++ {
		key, err := t.Next(doc)
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Panic(err)
		}
		cb(key)
		if i%saveCursorAfter == 0 {
			c, err := t.Cursor()
			if err == nil {
				log.Printf("Processed %d of %s", i, kind)
				cs.UpdateCursor(ctx, &c)
			}
		}
	}
}

func scanIssues(ctx context.Context, c chan<- CloudTaskPayload, dsClient *datastore.Client) {
	issue := &Issue{}
	scanner(ctx, dsClient, "Issue", issue, []string{"private"}, func(key *datastore.Key) {
		c <- CloudTaskPayload{
			Path:       fmt.Sprintf("/%d", key.ID),
			Private:    issue.Private,
			EntityKind: "Issue",
		}
	})
}

func scanPatchSets(ctx context.Context, c chan<- CloudTaskPayload, dsClient *datastore.Client) {
	scanner(ctx, dsClient, "PatchSet", &struct{}{}, []string{}, func(key *datastore.Key) {
		pID := key.Parent.ID
		c <- CloudTaskPayload{
			Path:       fmt.Sprintf("/%d/patchset/%d", pID, key.ID),
			Private:    issuePrivateCache.IsPrivate(ctx, key.Parent),
			EntityKind: "PatchSet",
		}
	})

}

func scanPatches(ctx context.Context, c chan<- CloudTaskPayload, dsClient *datastore.Client) {
	patch := &Patch{}
	scanner(ctx, dsClient, "Patch", patch, []string{"filename"}, func(key *datastore.Key) {
		pID := key.Parent.ID
		gpKey := key.Parent.Parent
		gpID := gpKey.ID
		private := issuePrivateCache.IsPrivate(ctx, gpKey)
		c <- CloudTaskPayload{
			Path:       fmt.Sprintf("/%d/patchset/%d/%d", gpID, pID, key.ID),
			Private:    private,
			EntityKind: "Patch",
		}
		c <- CloudTaskPayload{
			Path:       fmt.Sprintf("/%d/diff/%d/%s", gpID, pID, patch.Filename),
			Private:    private,
			EntityKind: "Patch",
		}
	})
}

// StartBackgroundProcess initializes all variables and prefetches private
// issues. Then, it creates cloudTaskGoRoutines for writing cloud tasks and one
// go routine for each Kind that needs to be imported.
func StartBackgroundProcess(ctx context.Context) {
	dsClient, err := datastore.NewClient(ctx, os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		log.Panic(err)
	}

	ctClient, err := cloudtask.NewClient(ctx)
	if err != nil {
		log.Panic(err)
	}

	processPageURL := os.Getenv("PROCESS_PAGE_URL")
	if processPageURL == "" {
		log.Panic("envvar PROCESS_PAGE_URL not defined")
	}

	processServiceAccount := os.Getenv("PROCESS_SERVICE_ACCOUNT")
	if processServiceAccount == "" {
		log.Panic("envvar PROCESS_SERVICE_ACCOUNT not defined")
	}

	log.Printf("Queriying Issues for private changes")
	issuePrivateCache = NewIssuePrivateCacheMap(ctx, dsClient)
	log.Printf("Done querying issues for private changes")

	// Logic to create cloud tasks
	ch := make(chan CloudTaskPayload)
	tp := taskProcessor{
		ctClient: ctClient,
		ch:       ch,
		ctQueuePath: fmt.Sprintf(
			"projects/%s/locations/us-central1/queues/static-gen",
			os.Getenv("GOOGLE_CLOUD_PROJECT")),
		processPageURL:        processPageURL,
		processServiceAccount: processServiceAccount,
	}
	wgCT := &sync.WaitGroup{}
	wgCT.Add(cloudTaskGoRoutines)
	for i := 0; i < cloudTaskGoRoutines; i++ {
		go func() {
			tp.process(ctx)
			wgCT.Done()
		}()
	}

	// Logic to scan datastore entries
	wgS := &sync.WaitGroup{}
	wgS.Add(3)
	go func() {
		scanIssues(ctx, ch, dsClient)
		wgS.Done()
	}()
	go func() {
		scanPatchSets(ctx, ch, dsClient)
		wgS.Done()
	}()
	go func() {
		scanPatches(ctx, ch, dsClient)
		wgS.Done()
	}()

	wgS.Wait()
	close(ch)
	log.Printf("Waiting for all tasks to be processed")
	wgCT.Wait()
	log.Printf("Done creating cloud tasks")
}

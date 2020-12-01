package main

import (
	"context"
	"log"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
)

// IssuePrivateCacheMap holds private status for given Issue.
type IssuePrivateCacheMap map[int64]struct{}

// NewIssuePrivateCacheMap queries datastore for all private changes and stores
// results in-memory.
func NewIssuePrivateCacheMap(ctx context.Context, dsClient *datastore.Client) IssuePrivateCacheMap {
	m := make(IssuePrivateCacheMap)
	q := datastore.NewQuery("Issue").Filter("private=", true).KeysOnly()
	doc := struct{}{}
	for t := dsClient.Run(ctx, q); ; {
		key, err := t.Next(doc)
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Panic(err)
		}
		m[key.ID] = struct{}{}
	}
	return m
}

// IsPrivate returns true if given issue is private.
func (m IssuePrivateCacheMap) IsPrivate(ctx context.Context, key *datastore.Key) bool {
	_, ok := m[key.ID]
	return ok
}

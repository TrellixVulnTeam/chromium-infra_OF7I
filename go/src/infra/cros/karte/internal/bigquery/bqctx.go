// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bigquery

import "context"

type key string

// ProjectKey is the key used to store the current BigQuery project in the current context.
const projectKey = key("project key")

// ClientKey is the key used to store the current client in the current context.
const clientKey = key("client key")

// UseProject produces a new context with the given project.
func UseProject(ctx context.Context, project string) context.Context {
	return context.WithValue(ctx, projectKey, project)
}

// GetProject gets the current project out of the context.
func GetProject(ctx context.Context) string {
	project := ctx.Value(projectKey)
	// Panic with a specific, trackable error message if the project is not set.
	// Otherwise, all we see is an error about failing to convert an interface{} to a string.
	if project == nil {
		panic("project from context is unexpectedly nil")
	}
	// This cast can still fail, if it does that indicates a serious problem (We somehow stored a non-string in the context).
	return project.(string)
}

// UseClient produces a new context with the given client.
func UseClient(ctx context.Context, client Client) context.Context {
	return context.WithValue(ctx, clientKey, client)
}

// GetClient gets the current client out of the context.
func GetClient(ctx context.Context) Client {
	client := ctx.Value(clientKey)
	if client == nil {
		panic("client from context is unexpectedly nil")
	}
	return client.(Client)
}

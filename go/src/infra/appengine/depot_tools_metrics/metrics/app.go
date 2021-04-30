// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package main stores the reported JSON metrics from depot_tools into a
// BigQuery table.
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"infra/appengine/depot_tools_metrics/schema"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/appengine/gaeauth/server"
	"go.chromium.org/luci/appengine/gaemiddleware/standard"
	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/router"
	"google.golang.org/appengine"
)

const (
	projectID            string = "cit-cli-metrics"
	datasetID            string = "metrics"
	tableID              string = "depot_tools"
	serviceAccountSuffix string = "@chops-service-accounts.iam.gserviceaccount.com"
)

func main() {
	r := router.New()
	standard.InstallHandlers(r)

	m := standard.Base().Extend(
		auth.Authenticate(&server.OAuth2Method{Scopes: []string{server.EmailScope}}),
		CheckUploadAllowed,
	)

	r.GET("/should-upload", m, shouldUploadHandler)
	r.POST("/upload", m, uploadHandler)

	http.DefaultServeMux.Handle("/", r)
	appengine.Main()
}

// CheckUploadAllowed continues if the request in coming from a corp machine (proxy
// for "is a Googler") or from a service account. Exits with a 403 status code
// otherwise.
func CheckUploadAllowed(c *router.Context, next router.Handler) {
	id := auth.CurrentIdentity(c.Context)
	switch {
	// The request comes from a service account.
	case isServiceAccount(id):
		next(c)
	// TRUSTED_IP_REQUEST=1 means the request is coming from a corp machine.
	case c.Request.Header.Get("X-AppEngine-Trusted-IP-Request") == "1":
		next(c)
	default:
		http.Error(c.Writer, "Access Denied: You're not on corp.", http.StatusForbidden)
	}
}

func isServiceAccount(id identity.Identity) bool {
	return id.Kind() == identity.User && strings.HasSuffix(id.Value(), serviceAccountSuffix)
}

// shouldUploadHandler handles the '/should-upload' endpoint, which is used by
// depot_tools to check whether it should collect and upload metrics.
func shouldUploadHandler(c *router.Context) {
	fmt.Fprintf(c.Writer, "Success")
}

// uploadHandler handles the '/upload' endpoint, which is used by depot_tools
// to upload the collected metrics in a JSON format. It enforces the schema
// defined in 'metrics_schema.json' and writes the data to the BigQuery table
// projectID.datasetID.tableID.
func uploadHandler(c *router.Context) {
	var metrics schema.Metrics
	if err := jsonpb.Unmarshal(c.Request.Body, &metrics); err != nil {
		logging.Errorf(c.Context, "Could not extract metrics: %v", err)
		http.Error(c.Writer, err.Error(), http.StatusBadRequest)
		return
	}

	// Ignore metrics.BotMetrics values from non-service-accounts.
	if !isServiceAccount(auth.CurrentIdentity(c.Context)) {
		metrics.BotMetrics = nil
	}

	if err := checkConstraints(&metrics); err != nil {
		logging.Errorf(c.Context, "The metrics don't obey constraints: %v", err)
		http.Error(c.Writer, err.Error(), http.StatusBadRequest)
		return
	}

	reportDepotToolsMetrics(c.Context, &metrics)

	ctx := appengine.WithContext(c.Context, c.Request)
	if err := putMetrics(ctx, &metrics); err != nil {
		logging.Errorf(c.Context, "Could not write to BQ: %v", err)
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(c.Writer, "Success")
}

// putMetrics extracts the Metrics from the request and streams them into the
// BigQuery table.
func putMetrics(ctx context.Context, metrics *schema.Metrics) error {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return err
	}

	up := bq.NewUploader(ctx, client, datasetID, tableID)
	up.SkipInvalidRows = true
	up.IgnoreUnknownValues = true

	return up.Put(ctx, metrics)
}

// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
)

const rietveldBucket = "chromiumcodereview"

// Attrs contains information for a GS object
type Attrs struct {
	Private     bool
	StatusCode  int
	ContentType string
}

func main() {
	http.Handle("/", http.HandlerFunc(pathHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// pathHandler handles /<path> to access gs://chromiumcodereview/<path>.
func pathHandler(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	// Remove trailing slashes, so that '/<issue>/' works as well as '/<issue>'.
	path := strings.TrimSuffix(req.URL.Path, "/")

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("failed to create storage client: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	obj := client.Bucket(rietveldBucket).Object(path)

	attrs, err := fetchAttrs(ctx, obj)
	if err != nil {
		log.Printf("failed to fetch attributes for %s: %v", path, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if attrs.Private {
		// TODO(crbug.com/1146637): Redirect to login page and get user email.
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}

	reader, err := obj.NewReader(ctx)
	if err != nil {
		log.Printf("failed to fetch %s: %v", path, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.WriteHeader(attrs.StatusCode)
	w.Header().Set("Content-Type", attrs.ContentType)

	_, err = io.Copy(w, reader)
	if err != nil {
		log.Printf("failed to copy %s: %v", path, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func fetchAttrs(ctx context.Context, obj *storage.ObjectHandle) (*Attrs, error) {
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, err
	}

	private, ok := attrs.Metadata["Rietveld-Private"]
	if !ok {
		return nil, errors.New("expected object metadata to contain Rietveld-Private attribute")
	}

	statusCodeStr, ok := attrs.Metadata["Status-Code"]
	if !ok {
		return nil, errors.New("expected object metadata to contain Status-Code attribute")
	}
	statusCode, err := strconv.Atoi(statusCodeStr)
	if err != nil {
		return nil, errors.New("expected object Status-Code attribute to be an integer: %v")
	}

	return &Attrs{
		Private:     private == "True",
		StatusCode:  statusCode,
		ContentType: attrs.ContentType,
	}, nil
}

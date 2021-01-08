// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"google.golang.org/api/idtoken"
)

const rietveldBucket = "chromiumcodereview"
const privateProjectID = "chromiumcodereview-private"
const privateProjectURL = "https://" + privateProjectID + ".appspot.com"

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
		// Private issues should only be accessed by a project protected with an
		// IAP. Redirect to a protected project if necessary.
		projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
		if projectID != privateProjectID {
			log.Printf("redirecting to %s", privateProjectURL+path)
			http.Redirect(w, req, privateProjectURL+path, http.StatusMovedPermanently)
			return
		}
		// Validate that the IAP JWT is valid and the user is authorized when trying
		// to access a private issue.
		err = authorize(ctx, req)
		if err != nil {
			log.Printf("not authorized: %v", err)
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}
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
		return nil, fmt.Errorf("expected object Status-Code attribute to be an integer: %v", statusCode)
	}

	return &Attrs{
		Private:     private == "True",
		StatusCode:  statusCode,
		ContentType: attrs.ContentType,
	}, nil
}

// authorize validates the JWT token present in the request and ensures that the
// user is authorized to view private issues.
func authorize(ctx context.Context, req *http.Request) error {
	jwt := req.Header.Get("X-Goog-IAP-JWT-Assertion")
	if len(jwt) == 0 {
		return errors.New("X-Goog-IAP-JWT-Assertion header not present")
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	projectNumber, err := metadata.NumericProjectID()
	if err != nil {
		return fmt.Errorf("failed to fetch project number: %v", err)
	}

	aud := fmt.Sprintf("/projects/%s/apps/%s", projectNumber, projectID)
	payload, err := idtoken.Validate(ctx, jwt, aud)
	if err != nil {
		return fmt.Errorf("idtoken.Validate: %v", err)
	}

	email, ok := payload.Claims["email"].(string)
	if !ok {
		return errors.New("email not present in JWT claims")
	}
	if !strings.HasSuffix(email, "@google.com") && !strings.HasSuffix(email, "@chromium.org") {
		return fmt.Errorf("not a @google.com or @chromium.org account: %v", email)
	}

	return nil
}

// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command tag-manager scans registered container repo and updates image tags
// based on defined policies.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/google"
)

var (
	serviceAccountJSON = flag.String("service-account-json", "", "Path to JSON file with service account credentials to use (Default is to use GCP env auth)")
)

func main() {
	if err := innerMain(); err != nil {
		log.Fatalf("tag-manager: %s", err)
	}
	log.Printf("Done")
}

const (
	// latestOfficial always indicates the latest (i.e. newest) official
	// image.
	latestOfficial = "latest-official"
	canary         = "canary"
	prod           = "prod"
)

var (
	latestOfficialPolicy    = &latestPolicy{tag: latestOfficial}
	canaryMaxDistancePolicy = &maxDistancePolicy{
		tagToControl:    canary,
		tagToFollow:     latestOfficial,
		maxVersionNewer: 0,
		maxVersionOlder: 0,
	}
	prodMaxDistancePolicy = &maxDistancePolicy{
		tagToControl:    prod,
		tagToFollow:     canary,
		maxVersionNewer: 0,
		maxVersionOlder: 1,
	}
)

func innerMain() error {
	flag.Parse()
	var auth authn.Authenticator
	if *serviceAccountJSON != "" {
		content, err := os.ReadFile(*serviceAccountJSON)
		if err != nil {
			return fmt.Errorf("read credential %q: %s", *serviceAccountJSON, err)
		}
		auth = google.NewJSONKeyAuthenticator(string(content))
	} else {
		log.Printf("No service account key specified, will use the GCP env auth")
		var err error
		auth, err = google.NewEnvAuthenticator()
		if err != nil {
			return fmt.Errorf("get GCP env auth: %s", err)
		}
	}

	// Please ensure the official tag regex matches the whole tag, i.e. starting
	// with '^' and ending with '$'.
	// The order of tag policies matters! Because tag policies may depend on
	// each other, e.g. policy of "prod" may depend on policy of "canary".
	// Please add the dependent policy first.
	data := []struct {
		repo *gcrRepo
		app  *appConfig
	}{
		// Images used by Drone service.
		{
			&gcrRepo{"gcr.io/chromeos-drone-images/drone", auth},
			newAppConfig(
				`^\d{8}T\d{6}-chromeos-test$`, latestOfficialPolicy, canaryMaxDistancePolicy, prodMaxDistancePolicy,
			),
		},
		// Images used by caching service.
		{
			&gcrRepo{"gcr.io/chromeos-cacheserver-images/nginx", auth},
			newAppConfig(`^\d{8}T\d{6}-chromeos-test$`, latestOfficialPolicy),
		},
		{
			&gcrRepo{"gcr.io/chromeos-cacheserver-images/conf_creator", auth},
			newAppConfig(`^\d{8}T\d{6}$`, latestOfficialPolicy),
		},
		{
			&gcrRepo{"gcr.io/chromeos-cacheserver-images/nginx_access_log_metrics", auth},
			newAppConfig(`^\d{8}T\d{6}$`, latestOfficialPolicy),
		},
		{
			&gcrRepo{"gcr.io/chromeos-cacheserver-images/gsa_server", auth},
			newAppConfig(`^\d{8}T\d{6}$`, latestOfficialPolicy),
		},
		// Image used by RPM service.
		{
			&gcrRepo{"gcr.io/chromeos-rpmserver-images/rpm", auth},
			newAppConfig(`^\d{8}T\d{6}-cloudbuild$`, latestOfficialPolicy),
		},
		// Image used by K8s Metrics service.
		{
			&gcrRepo{"gcr.io/cros-lab-servers/k8s-metrics", auth},
			newAppConfig(`^\d{8}T\d{6}$`, latestOfficialPolicy),
		},
	}
	ch := make(chan string, len(data))
	var wg sync.WaitGroup
	for _, d := range data {
		wg.Add(1)
		go func(a *appConfig, r *gcrRepo) {
			defer wg.Done()

			if err := a.apply(r); err != nil {
				log.Printf("%q: Apply config failed: %s", r.Name(), err)
				ch <- fmt.Sprintf("%q", r.Name())
			}
		}(d.app, d.repo)
	}
	wg.Wait()
	close(ch)
	var names []string
	for n := range ch {
		names = append(names, n)
	}
	if len(names) > 0 {
		return fmt.Errorf("apply failed on %s", strings.Join(names, ", "))
	}
	return nil
}
